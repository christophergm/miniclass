package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestArtifactTenantIsolationAndCurrentAssignments(t *testing.T) {
	h := testharness.Open(t)
	ctx := h.Context
	org := h.MintOrganization(t)
	otherOrg := h.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "synthetic artifact test"}
	f := factories.New(h.Database, string(org), actor)
	year, err := f.CreateSchoolYear(ctx, "Synthetic artifact year")
	require.NoError(t, err)
	grade, err := f.CreateGradeLevel(ctx, year.ID, "artifact-grade", "1")
	require.NoError(t, err)
	room, err := f.CreateHomeroom(ctx, year.ID, "Synthetic room")
	require.NoError(t, err)
	prog, err := f.CreateProgram(ctx, year.ID, "Synthetic program")
	require.NoError(t, err)
	session, err := f.CreateSession(ctx, year.ID, prog.ID, "Synthetic session", []time.Time{time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := f.CreateOffering(ctx, year.ID, prog.ID, session.ID, "Synthetic offering", "Synthetic description", nil, 20, grade.ID, grade.ID, "Gym", "Gate", "Synthetic instructions", nil)
	require.NoError(t, err)
	student, err := f.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Alex", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: room.ID})
	require.NoError(t, err)
	service := program.New(h.Database)
	_, err = service.AddMembership(ctx, string(org), actor, year.ID, prog.ID, student.ID)
	require.NoError(t, err)
	err = h.Database.InTenant(ctx, string(org), actor, func(ctx context.Context, tx *data.Tx) error {
		assignment, err := tx.CreateAssignment(ctx, data.CreateAssignmentInput{SchoolYearID: year.ID, ProgramID: prog.ID, SessionID: session.ID, StudentID: student.ID, OfferingID: offering.ID, Origin: "manual", RealizedQuality: "neutral"})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "assignment", ObjectID: &assignment.ID, SchoolYearID: &year.ID, Reason: "synthetic artifact fixture", ChangeSummary: []byte(`{}`)})
	})
	require.NoError(t, err)
	otherProgram, err := f.CreateProgram(ctx, year.ID, "Other synthetic program")
	require.NoError(t, err)
	for _, kind := range []program.ArtifactKind{program.ArtifactClassList, program.ArtifactHomeroomDismissal} {
		doc, err := service.GenerateArtifact(ctx, string(org), year.ID, prog.ID, session.ID, kind)
		require.NoError(t, err)
		require.Len(t, doc.Sections, 1)
		require.Empty(t, doc.Warnings)
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		require.Contains(t, string(raw), "Alex Synthetic")
		_, err = service.GenerateArtifact(ctx, string(otherOrg), year.ID, prog.ID, session.ID, kind)
		require.ErrorIs(t, err, pgx.ErrNoRows)

		_, err = service.GenerateArtifact(ctx, string(org), year.ID, otherProgram.ID, session.ID, kind)
		require.ErrorIs(t, err, pgx.ErrNoRows)
	}
	require.NoError(t, people.New(h.Database).DeleteStudent(ctx, string(org), year.ID, student.ID, actor))
	doc, err := service.GenerateArtifact(ctx, string(org), year.ID, prog.ID, session.ID, program.ArtifactClassList)
	require.NoError(t, err)
	raw, err := json.Marshal(doc)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "Alex Synthetic")
	require.Contains(t, string(raw), "Deleted student")
	require.Empty(t, doc.Warnings)
}
