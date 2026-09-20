package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestStudentCorrectionsReconcileDependenciesAndRegenerateArtifacts(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "student correction integration"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic correction year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-correction", "Synthetic Correction Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Correction Room")
	require.NoError(t, err)
	target, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Consented", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	adult, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{LegalGivenName: "Guardian", LegalFamilyName: "Synthetic", Email: stringPointer("guardian-correction@example.test")})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: target.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)

	service := people.New(harness.Database)
	regenerated := false
	service.WithArtifactRegenerator(func(_ context.Context, _ *data.Tx, sourceID, targetID ids.XID) (int, error) {
		regenerated = sourceID != "" && targetID == target.ID
		return 2, nil
	})
	placeholder, err := service.CreatePlaceholderStudent(ctx, string(organizationID), year.ID, actor, people.PlaceholderStudentInput{
		LegalGivenName: "Unknown", LegalFamilyName: "A", GradeLevelID: grade.ID, HomeroomID: homeroom.ID, Reason: "awaiting signed correction",
	})
	require.NoError(t, err)
	require.True(t, placeholder.IsPlaceholder)
	require.Equal(t, "placeholder", placeholder.Provenance)

	_, err = factory.CreateGuardianRelationship(ctx, year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: placeholder.ID, RelationshipType: data.GuardianRelationshipParent})
	require.Error(t, err, "database guard must keep placeholders out of guardian scope")

	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Correction Programme")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, placeholder.ID)
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Correction Session", []time.Time{time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	_, err = factory.CreateSessionNonParticipation(ctx, year.ID, programRow.ID, session.ID, placeholder.ID, "synthetic exclusion")
	require.NoError(t, err)

	result, err := service.ReconcilePlaceholderStudent(ctx, string(organizationID), year.ID, placeholder.ID, target.ID, actor, "matched signed guardian registration")
	require.NoError(t, err)
	require.Equal(t, int64(1), result.ProgramMembershipsMoved)
	require.Equal(t, int64(1), result.SessionNonParticipationsMoved)
	require.Equal(t, 2, result.ArtifactsRegenerated)
	require.True(t, regenerated)

	students, err := service.ListStudents(ctx, string(organizationID), year.ID, true)
	require.NoError(t, err)
	var deletedPlaceholder data.Student
	for _, student := range students {
		if student.ID == placeholder.ID {
			deletedPlaceholder = student
		}
	}
	require.NotNil(t, deletedPlaceholder.DeletedAt)
	memberships, err := program.New(harness.Database).ListMemberships(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.Equal(t, target.ID, memberships[0].StudentID)
	nonParticipants, err := program.New(harness.Database).ListSessionNonParticipations(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
	require.NoError(t, err)
	require.Len(t, nonParticipants, 1)
	require.Equal(t, target.ID, nonParticipants[0].StudentID)

	objectType := "student"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	var reconciliationAudit bool
	for _, entry := range entries {
		if entry.Action == string(audit.ActionStudentReconciliation) {
			reconciliationAudit = true
			require.Equal(t, "matched signed guardian registration", entry.Reason.String)
			require.True(t, entry.Reason.Valid)
		}
	}
	require.True(t, reconciliationAudit)
}

func TestStudentCorrectionCannotReconcileWithoutConsent(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "student correction consent test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic consent correction year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-consent-correction", "Synthetic Consent Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Consent Room")
	require.NoError(t, err)
	target, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Target", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	placeholder, err := people.New(harness.Database).CreatePlaceholderStudent(ctx, string(organizationID), year.ID, actor, people.PlaceholderStudentInput{LegalGivenName: "Unknown", LegalFamilyName: "B", GradeLevelID: grade.ID, HomeroomID: homeroom.ID, Reason: "synthetic placeholder"})
	require.NoError(t, err)
	_, err = people.New(harness.Database).ReconcilePlaceholderStudent(ctx, string(organizationID), year.ID, placeholder.ID, target.ID, actor, "attempt without consent")
	require.Error(t, err)
	require.True(t, errors.Is(err, people.ErrReconciliationTargetNotConsented), "err=%v", err)
}
