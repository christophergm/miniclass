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
	otherGuardian, err := peopleService.Create(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Other", LegalFamilyName: "Guardian", ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)
	_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{AdultID: otherGuardian.ID, StudentID: tenant.student.ID, RelationshipType: data.GuardianRelationshipGrandparent})
	require.NoError(t, err)
	deletedGuardian, err := peopleService.Create(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Deleted", LegalFamilyName: "Guardian", ParticipationIntent: adultIntentPtr(data.AdultParticipationHelp),
	})
	require.NoError(t, err)
	_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{AdultID: deletedGuardian.ID, StudentID: tenant.student.ID, RelationshipType: data.GuardianRelationshipOther})
	require.NoError(t, err)
	require.NoError(t, peopleService.Delete(ctx, string(tenant.organizationID), tenant.year.ID, deletedGuardian.ID, actor))
	_, err = peopleService.CreateGuardianRelationship(ctx, string(foreign.organizationID), foreign.year.ID, actor, people.GuardianRelationshipCreateInput{AdultID: foreign.adult.ID, StudentID: foreign.student.ID, RelationshipType: data.GuardianRelationshipGuardian})
	require.NoError(t, err)
	principal := auth.GuardianPrincipal{AdultID: tenant.adult.ID, OrganizationID: tenant.organizationID, SchoolYearID: tenant.year.ID, Email: "guardian@example.test"}
	service := guardianrecords.New(harness.Database)
	students, err := service.List(ctx, principal)
	require.NoError(t, err)
	require.Len(t, students, 1, "guardian list remains scoped to the authenticated guardian")
	require.Equal(t, tenant.student.ID, students[0].ID)
	require.Equal(t, []guardianrecords.OtherGuardian{{
		LegalGivenName: "Other", LegalFamilyName: "Guardian", RelationshipType: data.GuardianRelationshipGrandparent,
	}}, students[0].OtherGuardians, "only other active guardians in the same tenant and year are disclosed")
	guardianVocabulary, err := service.Vocabulary(ctx, principal)
	require.NoError(t, err)
	require.Equal(t, []guardianrecords.VocabularyOption{{ID: tenant.gradeID, Label: "Relationship Grade GuardianRecords"}}, guardianVocabulary.GradeLevels)
	require.Equal(t, []guardianrecords.VocabularyOption{{ID: tenant.homeroomID, Label: "Relationship Room GuardianRecords"}}, guardianVocabulary.Homerooms)
	profile, err := service.GetProfile(ctx, principal)
	require.NoError(t, err)
	require.Equal(t, tenant.adult.LegalGivenName, profile.LegalGivenName)
	require.Equal(t, tenant.adult.LegalFamilyName, profile.LegalFamilyName)

	candidates, err := service.FindCandidates(ctx, principal, guardianrecords.CandidateInput{GivenName: " casey ", FamilyName: "SYNTHETIC"})
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	require.Equal(t, second.ID, candidates[0].ID)
	require.Equal(t, "Relationship Room GuardianRecords", candidates[0].HomeroomLabel)

	crossNameStudent, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Alex", PreferredGivenName: stringPtr("Lex"), LegalFamilyName: "CrossName", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	for _, input := range []guardianrecords.CandidateInput{
		{GivenName: "Alex", FamilyName: "CrossName"},
		{GivenName: "Lex", FamilyName: "CrossName"},
		{GivenName: "Different", PreferredName: "Alex", FamilyName: "CrossName"},
		{GivenName: "Different", PreferredName: "Lex", FamilyName: "CrossName"},
	} {
		matches, err := service.FindCandidates(ctx, principal, input)
		require.NoError(t, err)
		require.Len(t, matches, 1)
		require.Equal(t, crossNameStudent.ID, matches[0].ID)
	}

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
