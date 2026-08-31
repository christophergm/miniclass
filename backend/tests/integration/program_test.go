package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestProgramMembershipRequiresGradeAndFlagsLaterRemoval(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "program integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic program year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-program", "Synthetic Program Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Program Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Enrichment")
	require.NoError(t, err)
	ungraded, err := factory.CreateUngradedStudent(ctx, year.ID, homeroom.ID, "Ungraded", "Synthetic")
	require.NoError(t, err)

	service := program.New(harness.Database)
	_, err = service.AddMembership(ctx, string(organizationID), actor, year.ID, programRow.ID, ungraded.ID)
	require.Error(t, err)
	require.True(t, errors.Is(err, program.ErrStudentGradeRequired))
	require.Contains(t, err.Error(), "Ungraded Synthetic")

	graded, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{
		LegalGivenName: "Graded", LegalFamilyName: "Synthetic", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	membership, err := service.AddMembership(ctx, string(organizationID), actor, year.ID, programRow.ID, graded.ID)
	require.NoError(t, err)
	require.False(t, membership.GradeMissing)
	require.Equal(t, "Graded", membership.LegalGivenName)

	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetStudentByID(ctx, year.ID, graded.ID)
		if err != nil {
			return err
		}
		updated, err := tx.UpdateStudent(ctx, year.ID, graded.ID, current.LegalGivenName, current.LegalFamilyName, current.PreferredGivenName, nil, current.HomeroomID, current.ExternalIdentifier, current.PriorYearStudentID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "student", ObjectID: &updated.ID, SchoolYearID: &updated.SchoolYearID, Reason: "synthetic grade removal", ChangeSummary: []byte(`{"grade_level_id":null}`)})
	})
	require.NoError(t, err)

	memberships, err := service.ListMemberships(ctx, string(organizationID), year.ID, programRow.ID)
	require.NoError(t, err)
	require.Len(t, memberships, 1)
	require.True(t, memberships[0].GradeMissing)
	require.Equal(t, graded.ID, memberships[0].StudentID)

	count, err := service.CountStudentsWithoutGrade(ctx, string(organizationID), year.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
}

func TestInterestAreaVocabularyPreservesIdentityAndAuditsChanges(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "interest-area integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic interest area year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic enrichment")
	require.NoError(t, err)
	service := program.New(harness.Database)

	first, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Arts and crafts")
	require.NoError(t, err)
	second, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Gardening")
	require.NoError(t, err)
	require.Equal(t, 1, first.Ordinal)
	require.Equal(t, 2, second.Ordinal)

	listed, err := service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	newLabel := "Arts & crafts"
	updated, err := service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Label: &newLabel})
	require.NoError(t, err)
	require.Equal(t, first.ID, updated.ID)
	require.Equal(t, newLabel, updated.Label)
	require.Equal(t, first.Ordinal, updated.Ordinal)

	retired := true
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.NotNil(t, updated.RetiredAt)

	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{second.ID}, []ids.XID{listed[0].ID}, "retired areas are excluded from selection lists")
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, true)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	retired = false
	updated, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, first.ID, program.InterestAreaUpdate{Retired: &retired})
	require.NoError(t, err)
	require.Nil(t, updated.RetiredAt)
	listed, err = service.ListInterestAreas(ctx, string(organizationID), year.ID, programRow.ID, false)
	require.NoError(t, err)
	require.Equal(t, []ids.XID{first.ID, second.ID}, []ids.XID{listed[0].ID, listed[1].ID})

	objectType := "interest_area"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 5, "each vocabulary create, edit, retirement, and reactivation is audited")
	firstChanges, secondChanges := 0, 0
	for _, entry := range entries {
		require.Equal(t, string(audit.ActionVocabularyChange), entry.Action)
		require.NotNil(t, entry.ObjectID)
		switch string(*entry.ObjectID) {
		case string(first.ID):
			firstChanges++
		case string(second.ID):
			secondChanges++
		default:
			t.Errorf("unexpected interest-area audit object %q", string(*entry.ObjectID))
		}
	}
	require.Equal(t, 4, firstChanges)
	require.Equal(t, 1, secondChanges)
}

func TestClosedYearInterestAreaMutationIsRejected(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed interest-area integration test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic closed interest area year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic closed enrichment")
	require.NoError(t, err)
	service := program.New(harness.Database)
	area, err := service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Synthetic area")
	require.NoError(t, err)

	yearService := schoolyear.New(harness.Database)
	year, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	newLabel := "Closed edit"
	_, err = service.UpdateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, area.ID, program.InterestAreaUpdate{Label: &newLabel})
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year label edit = %v", err)

	_, err = service.CreateInterestArea(ctx, string(organizationID), actor, year.ID, programRow.ID, "Closed create")
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year area create = %v", err)
}
