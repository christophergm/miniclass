package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
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
	grade, err := factory.CreateGradeLevel(ctx, "synthetic-program", "Synthetic Program Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, "Synthetic Program Room")
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
