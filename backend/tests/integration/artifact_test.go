package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
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
	// A later preference disagrees with the stored neutral quality (SPEC §8.6).
	var userID ids.XID
	require.NoError(t, h.Migrator.QueryRow(ctx, `insert into users (provider_subject, email) values ($1, $2) returning id`, "artifact-admin-"+string(org), "synthetic-artifact-"+string(org)+"@example.test").Scan(&userID))
	_, err = h.Migrator.Exec(ctx, `insert into organization_members (organization_id, user_id, role) values ($1, $2, 'administrator')`, org, userID)
	require.NoError(t, err)
	preferenceActor := audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "synthetic artifact preference administrator"}
	require.NoError(t, h.Database.InTenant(ctx, string(org), preferenceActor, func(ctx context.Context, tx *data.Tx) error {
		submission, _, err := tx.CreateRankedChoiceSubmission(ctx, year.ID, prog.ID, session.ID, student.ID, data.PreferenceChannelAdministratorOnBehalf, nil, []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceNotInterested}})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "ranked_choice_submission", ObjectID: &submission.ID, SchoolYearID: &year.ID, Reason: "synthetic later preference", ChangeSummary: []byte(`{}`)})
	}))
	require.NoError(t, h.Database.InTenantRead(ctx, string(org), func(ctx context.Context, tx *data.Tx) error {
		answers, err := tx.ListCurrentSessionResultAnswers(ctx, year.ID, prog.ID, session.ID)
		if err != nil {
			return err
		}
		require.Equal(t, []data.SessionResultAnswer{{StudentID: student.ID, OfferingID: offering.ID, Answer: data.RankedChoiceNotInterested}}, answers)
		return nil
	}))
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
		studentBlocks := 0
		for _, block := range doc.Sections[0].Blocks {
			if block.Kind == "numbered" || block.Kind == "bullet" {
				require.Equal(t, "neutral", block.Quality, "quality must retain the stored result despite the later not-interested preference")
				studentBlocks++
			} else {
				require.Empty(t, block.Quality)
			}
		}
		require.Equal(t, 1, studentBlocks)
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
