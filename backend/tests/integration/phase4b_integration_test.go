package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/guardianrecords"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestPhase4BGuardianPrivacyScopeAndDeletionBoundaries(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "phase 4b privacy integration"}
	peopleService := people.New(harness.Database)
	tenant := newGuardianFixture(t, harness, peopleService, actor, "Phase4BPrivacy")
	foreign := newGuardianFixture(t, harness, peopleService, actor, "Phase4BForeign")
	tenantFactory := factories.New(harness.Database, string(tenant.organizationID), actor)

	preferred := "CJ"
	externalIdentifier := "private-external-identifier"
	target, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Casey", LegalFamilyName: "Sensitive", PreferredGivenName: &preferred,
		GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID, ExternalIdentifier: &externalIdentifier,
	})
	require.NoError(t, err)
	placeholder, err := peopleService.CreatePlaceholderStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.PlaceholderStudentInput{
		LegalGivenName: "Casey", LegalFamilyName: "Sensitive", GradeLevelID: tenant.gradeID, HomeroomID: tenant.homeroomID,
		Reason: "synthetic unregistered child",
	})
	require.NoError(t, err)
	require.True(t, placeholder.IsPlaceholder)

	otherYear, err := schoolyear.New(harness.Database).Create(ctx, string(tenant.organizationID), actor, "Synthetic other phase 4b year")
	require.NoError(t, err)
	otherGrade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(tenant.organizationID), otherYear.ID, actor, "phase4b-other-grade", "Synthetic Other Grade")
	require.NoError(t, err)
	otherHomeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(tenant.organizationID), otherYear.ID, actor, "Synthetic Other Room", nil)
	require.NoError(t, err)
	_, err = peopleService.CreateStudent(ctx, string(tenant.organizationID), otherYear.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Casey", LegalFamilyName: "Sensitive", GradeLevelID: xidPtr(otherGrade.ID), HomeroomID: otherHomeroom.ID,
	})
	require.NoError(t, err)
	foreignStudent, err := peopleService.CreateStudent(ctx, string(foreign.organizationID), foreign.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Casey", LegalFamilyName: "Sensitive", GradeLevelID: xidPtr(foreign.gradeID), HomeroomID: foreign.homeroomID,
	})
	require.NoError(t, err)

	_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: tenant.adult.ID, StudentID: tenant.student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)
	principal := auth.GuardianPrincipal{AdultID: tenant.adult.ID, OrganizationID: tenant.organizationID, SchoolYearID: tenant.year.ID, Email: "guardian@example.test"}
	records := guardianrecords.New(harness.Database)

	candidates, err := records.FindCandidates(ctx, principal, guardianrecords.CandidateInput{GivenName: " casey ", FamilyName: "SENSITIVE"})
	require.NoError(t, err)
	require.Len(t, candidates, 1, "matching is limited to the current tenant and school year and excludes placeholders")
	require.Equal(t, target.ID, candidates[0].ID)

	scoped, err := records.List(ctx, principal)
	require.NoError(t, err)
	require.Len(t, scoped, 1)
	require.Equal(t, tenant.student.ID, scoped[0].ID)
	_, err = records.Update(ctx, principal, target.ID, guardianrecords.UpdateInput{LegalGivenName: stringPointer("Blocked")}, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.ErrorIs(t, err, guardianrecords.ErrOutOfScope)
	_, err = records.Update(ctx, principal, foreignStudent.ID, guardianrecords.UpdateInput{LegalGivenName: stringPointer("Blocked")}, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)})
	require.ErrorIs(t, err, guardianrecords.ErrOutOfScope)
	require.ErrorIs(t, records.Detach(ctx, principal, foreignStudent.ID, true, audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)}), guardianrecords.ErrOutOfScope)

	historyStudent, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "History", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	hardDeletedStudent, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Hard", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	for _, studentID := range []ids.XID{target.ID, historyStudent.ID, hardDeletedStudent.ID} {
		_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{
			AdultID: tenant.adult.ID, StudentID: studentID, RelationshipType: data.GuardianRelationshipParent,
		})
		require.NoError(t, err)
	}
	programRow, err := tenantFactory.CreateProgram(ctx, tenant.year.ID, "Synthetic Phase 4B History Program")
	require.NoError(t, err)
	_, err = tenantFactory.AddProgramMembership(ctx, tenant.year.ID, programRow.ID, target.ID)
	require.NoError(t, err)

	guardianActor := audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)}
	require.NoError(t, records.Detach(ctx, principal, hardDeletedStudent.ID, true, guardianActor))
	_, err = peopleService.GetStudent(ctx, string(tenant.organizationID), tenant.year.ID, hardDeletedStudent.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)

	require.NoError(t, records.Detach(ctx, principal, target.ID, true, guardianActor))
	includingDeleted, err := peopleService.ListStudents(ctx, string(tenant.organizationID), tenant.year.ID, true)
	require.NoError(t, err)
	deletedTarget := findStudent(includingDeleted, target.ID)
	require.NotNil(t, deletedTarget)
	require.Equal(t, "Deleted student", deletedTarget.LegalGivenName)
	require.Equal(t, "Deleted student", deletedTarget.LegalFamilyName)
	require.Nil(t, deletedTarget.PreferredGivenName)
	require.Nil(t, deletedTarget.ExternalIdentifier)
	require.Equal(t, tenant.gradeID, *deletedTarget.GradeLevelID)
	require.Equal(t, tenant.homeroomID, deletedTarget.HomeroomID)
	require.NotNil(t, deletedTarget.DeletedAt)

	objectType := "student"
	entries, err := harness.Database.ListAuditLog(ctx, string(tenant.organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 200})
	require.NoError(t, err)
	var detachAudit bool
	for _, entry := range entries {
		if entry.Action == string(audit.ActionGuardianDetach) && entry.ObjectID != nil && *entry.ObjectID == target.ID {
			detachAudit = true
			require.Equal(t, guardianActor.Label, entry.ActorLabel)
			require.True(t, entry.OccurredAt.Valid)
			require.NotEmpty(t, entry.ChangeSummary)
		}
	}
	require.True(t, detachAudit, "de-identifying detach must retain actor, time, action, and outcome")
}

type phase4BSessionRevoker struct {
	organizationID ids.XID
	schoolYearID   ids.XID
	adultID        ids.XID
	calls          int
}

func (r *phase4BSessionRevoker) RevokeGuardianSessionsAndOTPs(_ context.Context, organizationID, schoolYearID, adultID ids.XID) error {
	r.organizationID, r.schoolYearID, r.adultID, r.calls = organizationID, schoolYearID, adultID, r.calls+1
	return nil
}

func TestPhase4BGuardianSelfDeleteHardDeletesAndDeidentifies(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "phase 4b self-delete integration"}
	peopleService := people.New(harness.Database)
	tenant := newGuardianFixture(t, harness, peopleService, actor, "Phase4BSelfDelete")
	factory := factories.New(harness.Database, string(tenant.organizationID), actor)
	retained, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Retained", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	hardDeleted, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Removed", LegalFamilyName: "Synthetic", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	for _, studentID := range []ids.XID{retained.ID, hardDeleted.ID} {
		_, err = peopleService.CreateGuardianRelationship(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.GuardianRelationshipCreateInput{
			AdultID: tenant.adult.ID, StudentID: studentID, RelationshipType: data.GuardianRelationshipParent,
		})
		require.NoError(t, err)
	}
	programRow, err := factory.CreateProgram(ctx, tenant.year.ID, "Synthetic Self Delete Program")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, tenant.year.ID, programRow.ID, retained.ID)
	require.NoError(t, err)

	revoker := &phase4BSessionRevoker{}
	records := guardianrecords.New(harness.Database, revoker)
	principal := auth.GuardianPrincipal{AdultID: tenant.adult.ID, OrganizationID: tenant.organizationID, SchoolYearID: tenant.year.ID, Email: "guardian@example.test"}
	guardianActor := audit.Actor{Type: audit.ActorTypeLink, Label: "guardian:" + string(tenant.adult.ID)}
	require.NoError(t, records.DeleteSelf(ctx, principal, true, guardianActor))
	require.Equal(t, 1, revoker.calls)
	require.Equal(t, tenant.organizationID, revoker.organizationID)
	require.Equal(t, tenant.year.ID, revoker.schoolYearID)
	require.Equal(t, tenant.adult.ID, revoker.adultID)

	_, err = peopleService.GetStudent(ctx, string(tenant.organizationID), tenant.year.ID, hardDeleted.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	included, err := peopleService.ListStudents(ctx, string(tenant.organizationID), tenant.year.ID, true)
	require.NoError(t, err)
	deletedRetained := findStudent(included, retained.ID)
	require.NotNil(t, deletedRetained)
	require.Equal(t, "Deleted student", deletedRetained.LegalGivenName)
	require.Equal(t, tenant.gradeID, *deletedRetained.GradeLevelID)
	require.Equal(t, tenant.homeroomID, deletedRetained.HomeroomID)

	relationships, err := peopleService.ListGuardianRelationships(ctx, string(tenant.organizationID), tenant.year.ID, data.GuardianRelationshipFilter{AdultID: tenant.adult.ID})
	require.NoError(t, err)
	require.Empty(t, relationships)
	adults, err := peopleService.List(ctx, string(tenant.organizationID), tenant.year.ID, true)
	require.NoError(t, err)
	deletedAdult := findAdult(adults, tenant.adult.ID)
	require.NotNil(t, deletedAdult)
	require.NotNil(t, deletedAdult.DeletedAt)

	objectType := "adult"
	entries, err := harness.Database.ListAuditLog(ctx, string(tenant.organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	var selfDeleteAudit bool
	for _, entry := range entries {
		if entry.Action == string(audit.ActionPersonalDataDelete) && entry.ObjectID != nil && *entry.ObjectID == tenant.adult.ID {
			selfDeleteAudit = true
			require.Equal(t, guardianActor.Label, entry.ActorLabel)
			require.True(t, entry.OccurredAt.Valid)
		}
	}
	require.True(t, selfDeleteAudit, "self-delete must retain actor, time, and action attribution")
}

func TestPhase4BReviewSignalsJoinPlaceholderDuplicateAndActivityEvidence(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "phase 4b review integration"}
	peopleService := people.New(harness.Database)
	tenant := newGuardianFixture(t, harness, peopleService, actor, "Phase4BReview")

	placeholder, err := peopleService.CreatePlaceholderStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.PlaceholderStudentInput{
		LegalGivenName: "Unknown", LegalFamilyName: "A", GradeLevelID: tenant.gradeID, HomeroomID: tenant.homeroomID, Reason: "synthetic review fixture",
	})
	require.NoError(t, err)
	duplicateOne, err := peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Alex", LegalFamilyName: "Rivera", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	_, err = peopleService.CreateStudent(ctx, string(tenant.organizationID), tenant.year.ID, actor, people.StudentCreateInput{
		LegalGivenName: " alex ", LegalFamilyName: "RIVERA", GradeLevelID: xidPtr(tenant.gradeID), HomeroomID: tenant.homeroomID,
	})
	require.NoError(t, err)
	firstExternal := "review-one"
	secondExternal := "review-two"
	firstExternalPtr := &firstExternal
	_, err = peopleService.UpdateStudentCorrection(ctx, string(tenant.organizationID), tenant.year.ID, duplicateOne.ID, actor, people.StudentUpdateInput{ExternalIdentifier: &firstExternalPtr}, "first synthetic correction")
	require.NoError(t, err)
	secondExternalPtr := &secondExternal
	_, err = peopleService.UpdateStudentCorrection(ctx, string(tenant.organizationID), tenant.year.ID, duplicateOne.ID, actor, people.StudentUpdateInput{ExternalIdentifier: &secondExternalPtr}, "second synthetic correction")
	require.NoError(t, err)

	signals, err := peopleService.ListStudentReviewSignals(ctx, string(tenant.organizationID), tenant.year.ID)
	require.NoError(t, err)
	codes := make(map[string]int)
	for _, signal := range signals {
		codes[signal.Code]++
		require.NotContains(t, signal.Detail, "@", "review signals must not expose contact data")
	}
	require.Equal(t, 1, codes["placeholder-student"])
	require.Equal(t, 2, codes["matching-duplicate"])
	require.GreaterOrEqual(t, codes["registration-unlinked"], 2)
	require.Equal(t, 1, codes["registration-no-email"])
	require.Equal(t, 1, codes["unusual-activity"])

	objectType := "student"
	entries, err := harness.Database.ListAuditLog(ctx, string(tenant.organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	correctionEntries := 0
	for _, entry := range entries {
		if entry.Action == string(audit.ActionStudentAdminCorrection) && entry.ObjectID != nil && *entry.ObjectID == duplicateOne.ID {
			correctionEntries++
			require.Equal(t, actor.Label, entry.ActorLabel)
			require.True(t, entry.OccurredAt.Valid)
			require.True(t, entry.Reason.Valid)
			require.NotEmpty(t, entry.Reason.String)
		}
	}
	require.Equal(t, 2, correctionEntries)
	require.NotEmpty(t, placeholder.ID)
}

func findAdult(rows []data.Adult, id ids.XID) *data.Adult {
	for i := range rows {
		if rows[i].ID == id {
			return &rows[i]
		}
	}
	return nil
}

var _ guardianrecords.GuardianSessionRevoker = (*phase4BSessionRevoker)(nil)
