package integration

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// SPEC §§13.2–13.5 and §17.4.1: the sidebar's student, membership, guardian,
// and session scope are all resolved inside the tenant transaction.
func TestStudentPreferenceContextScopesStudentAndTenant(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "student-context")
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic student context session", []time.Time{time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	_, err = factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic student context offering", "", nil, 3, fixture.grade.ID, fixture.grade.ID, "", "", "", &fixture.area.ID)
	require.NoError(t, err)
	guardian, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Guardian", Email: studentContextStringPointer("guardian@example.invalid")})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: guardian.ID, StudentID: fixture.student.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)

	service := program.New(harness.Database)
	context, err := service.GetStudentPreferenceContext(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, string(fixture.student.ID), context.Student.ID)
	require.False(t, context.Rankings.RankedChoiceEnabled)
	require.False(t, context.Rankings.Submitted)
	require.NotNil(t, context.Rankings.Rows)
	require.Equal(t, []program.StudentLinkedAdult{{DisplayName: "Synthetic Guardian", Relationship: "parent"}}, context.Student.LinkedAdults)

	unlinked, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Unlinked", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	_, err = service.GetStudentPreferenceContext(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, unlinked.ID)
	require.ErrorIs(t, err, program.ErrStudentNotProgramMember)

	foreignOrganizationID := harness.MintOrganization(t)
	_, err = service.GetStudentPreferenceContext(ctx, string(foreignOrganizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.Error(t, err, "another organization cannot read this student context")
}

func studentContextStringPointer(value string) *string { return &value }
