package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardianrecords"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/stretchr/testify/require"
)

func TestGuardianRecordsUsePrivacySafeLiveScopeAndWarnings(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "guardian records integration"}
	peopleService := people.New(harness.Database)
	tenant := newGuardianFixture(t, harness, peopleService, actor, "GuardianRecords")

	second, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Casey", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
		ExternalIdentifier: stringPtr("private-external-id"),
	})
	require.NoError(t, err)
	_, err = peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Placeholder", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	foreign := newGuardianFixture(t, harness, peopleService, actor, "GuardianRecordsForeign")

	_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{AdultID: tenant.adult.ID, StudentID: tenant.student.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)
	principal := auth.GuardianPrincipal{AdultID: tenant.adult.ID, OrganizationID: tenant.organizationID, SchoolYearID: tenant.year.ID, Email: "guardian@example.test"}
	service := guardianrecords.New(harness.Database)

	candidates, err := service.FindCandidates(ctx, principal, guardianrecords.CandidateInput{GivenName: " casey ", FamilyName: "SYNTHETIC"})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, second.ID, candidates[0].ID)
	require.Equal(t, "Guardian Room", candidates[0].HomeroomLabel)
	// The response type has no external identifier, email, or other guardian edge fields.
	selected, err := service.Select(ctx, principal, second.ID, data.GuardianRelationshipParent, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.NoError(t, err)
	require.Equal(t, second.ID, selected.ID)

	_, err = service.Update(ctx, principal, foreign.student.ID, guardianrecords.UpdateInput{LegalGivenName: stringPtr("Blocked")}, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.ErrorIs(t, err, guardianrecords.ErrOutOfScope)

	newGrade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(tenant.organizationID), tenant.year.ID, actor, "records-second", "Records Second")
	require.NoError(t, err)
	updated, err := service.Update(ctx, principal, second.ID, guardianrecords.UpdateInput{GradeLevelID: &newGrade.ID}, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.NoError(t, err)
	require.Len(t, updated.Warnings, 1)
	require.Equal(t, "student-attributes-review", updated.Warnings[0].Code)

	created, err := service.Create(ctx, principal, guardianrecords.CreateInput{LegalGivenName: "New", LegalFamilyName: "Synthetic", GradeLevelID: tenant.gradeID, HomeroomID: tenant.homeroomID, RelationshipType: data.GuardianRelationshipParent}, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.NoError(t, err)
	require.NotEqual(t, second.ID, created.ID)
	var relationships []data.GuardianRelationship
	err = harness.Database.InTenantRead(ctx, string(tenant.organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		relationships, err = tx.ListGuardianRelationships(ctx, tenant.year.ID, data.GuardianRelationshipFilter{AdultID: tenant.adult.ID})
		return err
	})
	require.NoError(t, err)
	require.Len(t, relationships, 3)
}
