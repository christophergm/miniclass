package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestVocabularyOrderingRetirementAndSettings(t *testing.T) {
	harness := testharness.Open(t)
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "vocabulary integration test"}
	service := vocabulary.New(harness.Database)
	year, err := schoolyear.New(harness.Database).Create(harness.Context, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	nextYear, err := schoolyear.New(harness.Database).Create(harness.Context, string(organizationID), actor, "2027–2028")
	require.NoError(t, err)

	firstGrade, err := service.CreateGrade(harness.Context, string(organizationID), year.ID, actor, "z", "Zed")
	require.NoError(t, err)
	secondGrade, err := service.CreateGrade(harness.Context, string(organizationID), year.ID, actor, "a", "Alpha")
	require.NoError(t, err)
	externalIdentifier := "blue-room"
	firstHomeroom, err := service.CreateHomeroom(harness.Context, string(organizationID), year.ID, actor, "Blue", &externalIdentifier)
	require.NoError(t, err)
	require.Equal(t, &externalIdentifier, firstHomeroom.ExternalIdentifier)
	nextYearGrade, err := service.CreateGrade(harness.Context, string(organizationID), nextYear.ID, actor, "z", "Zed next year")
	require.NoError(t, err)
	nextYearHomeroom, err := service.CreateHomeroom(harness.Context, string(organizationID), nextYear.ID, actor, "Blue", &externalIdentifier)
	require.NoError(t, err)
	require.NotEqual(t, firstGrade.ID, nextYearGrade.ID)
	require.NotEqual(t, firstHomeroom.ID, nextYearHomeroom.ID)
	_, err = service.CreateHomeroom(harness.Context, string(organizationID), year.ID, actor, "Amber", nil)
	require.NoError(t, err)
	_, err = service.CreateHomeroom(harness.Context, string(organizationID), year.ID, actor, "Duplicate identifier", &externalIdentifier)
	var duplicate *pgconn.PgError
	require.ErrorAs(t, err, &duplicate)
	require.Equal(t, "23505", duplicate.Code)

	snapshot, err := service.List(harness.Context, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Equal(t, []string{"z", "a"}, []string{snapshot.Grades[0].Code, snapshot.Grades[1].Code}, "grade picker order uses ordinal, not code")
	require.Equal(t, []string{"Amber", "Blue"}, []string{snapshot.Homerooms[0].Name, snapshot.Homerooms[1].Name})
	require.Equal(t, "homeroom", snapshot.Settings.HomeroomLabel)
	nextYearSnapshot, err := service.List(harness.Context, string(organizationID), nextYear.ID, false)
	require.NoError(t, err)
	require.Equal(t, nextYear.ID, nextYearSnapshot.SchoolYearID)
	require.Equal(t, []string{"z"}, []string{nextYearSnapshot.Grades[0].Code})
	require.Equal(t, []string{"Blue"}, []string{nextYearSnapshot.Homerooms[0].Name})

	retired := true
	_, err = service.UpdateGrade(harness.Context, string(organizationID), year.ID, firstGrade.ID, actor, vocabulary.GradeLevelUpdate{Retired: &retired})
	require.NoError(t, err)
	_, err = service.UpdateHomeroom(harness.Context, string(organizationID), year.ID, firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{Retired: &retired})
	require.NoError(t, err)
	updatedIdentifier := "blue-room-updated"
	updatedIdentifierValue := &updatedIdentifier
	updatedHomeroom, err := service.UpdateHomeroom(harness.Context, string(organizationID), year.ID, firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{ExternalIdentifier: &updatedIdentifierValue})
	require.NoError(t, err)
	require.Equal(t, &updatedIdentifier, updatedHomeroom.ExternalIdentifier)
	nextYearSnapshot, err = service.List(harness.Context, string(organizationID), nextYear.ID, false)
	require.NoError(t, err)
	require.Equal(t, &externalIdentifier, nextYearSnapshot.Homerooms[0].ExternalIdentifier, "editing one school year's homeroom must not affect another")

	snapshot, err = service.List(harness.Context, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Len(t, snapshot.Grades, 1)
	require.Len(t, snapshot.Homerooms, 1)
	retiredGrade, err := service.GetGrade(harness.Context, string(organizationID), year.ID, firstGrade.ID)
	require.NoError(t, err)
	require.NotNil(t, retiredGrade.RetiredAt, "retired grade remains a valid reference")
	retiredHomeroom, err := service.GetHomeroom(harness.Context, string(organizationID), year.ID, firstHomeroom.ID)
	require.NoError(t, err)
	require.NotNil(t, retiredHomeroom.RetiredAt, "retired homeroom remains a valid reference")

	retired = false
	_, err = service.UpdateGrade(harness.Context, string(organizationID), year.ID, firstGrade.ID, actor, vocabulary.GradeLevelUpdate{Retired: &retired})
	require.NoError(t, err)
	_, err = service.UpdateHomeroom(harness.Context, string(organizationID), year.ID, firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{Retired: &retired})
	require.NoError(t, err)

	_, err = service.ReorderGrades(harness.Context, string(organizationID), year.ID, actor, []ids.XID{secondGrade.ID, firstGrade.ID})
	require.NoError(t, err)
	snapshot, err = service.List(harness.Context, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Equal(t, secondGrade.ID, snapshot.Grades[0].ID)
	require.Equal(t, 1, snapshot.Grades[0].Ordinal)
	require.Equal(t, firstGrade.ID, snapshot.Grades[1].ID)
	require.Equal(t, 2, snapshot.Grades[1].Ordinal)
	nextYearSnapshot, err = service.List(harness.Context, string(organizationID), nextYear.ID, false)
	require.NoError(t, err)
	require.Equal(t, 1, nextYearSnapshot.Grades[0].Ordinal, "reordering one school year must not affect another")

	settings, err := service.UpdateHomeroomLabel(harness.Context, string(organizationID), actor, "advisory")
	require.NoError(t, err)
	require.Equal(t, "advisory", settings.HomeroomLabel)

	err = harness.Database.InTenantRead(context.Background(), string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		count, err := tx.Queries().CountAuditLog(ctx)
		require.NoError(t, err)
		require.Greater(t, count, int64(0), "vocabulary definition changes and retirement are audited")
		return nil
	})
	require.NoError(t, err)
}

func TestNullableRosterFieldsAreAccepted(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "nullable roster integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), year.ID, actor, "Nullable Room", nil)
	require.NoError(t, err)
	service := people.New(harness.Database)
	student, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Ungraded", LegalFamilyName: "Synthetic", HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	require.Nil(t, student.GradeLevelID)
	adult, err := service.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Undeclared", LegalFamilyName: "Synthetic",
	})
	require.NoError(t, err)
	require.Nil(t, adult.ParticipationIntent)
}

func TestClosedYearVocabularyMutationIsRejectedByDatabaseTrigger(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed vocabulary integration test"}
	yearService := schoolyear.New(harness.Database)
	year, err := yearService.Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	year, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive)})
	require.NoError(t, err)
	_, err = yearService.Update(ctx, string(organizationID), year.ID, authRoleAdministrator, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)

	vocabularyService := vocabulary.New(harness.Database)
	_, err = vocabularyService.CreateGrade(ctx, string(organizationID), year.ID, actor, "closed", "Closed")
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year grade create = %v", err)
	_, err = vocabularyService.CreateHomeroom(ctx, string(organizationID), year.ID, actor, "Closed Room", nil)
	require.Error(t, err)
	require.True(t, data.IsSchoolYearClosed(err), "closed-year homeroom create = %v", err)
}
