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

	firstGrade, err := service.CreateGrade(harness.Context, string(organizationID), actor, "z", "Zed")
	require.NoError(t, err)
	secondGrade, err := service.CreateGrade(harness.Context, string(organizationID), actor, "a", "Alpha")
	require.NoError(t, err)
	externalIdentifier := "blue-room"
	firstHomeroom, err := service.CreateHomeroom(harness.Context, string(organizationID), actor, "Blue", &externalIdentifier)
	require.NoError(t, err)
	require.Equal(t, &externalIdentifier, firstHomeroom.ExternalIdentifier)
	_, err = service.CreateHomeroom(harness.Context, string(organizationID), actor, "Amber", nil)
	require.NoError(t, err)
	_, err = service.CreateHomeroom(harness.Context, string(organizationID), actor, "Duplicate identifier", &externalIdentifier)
	var duplicate *pgconn.PgError
	require.ErrorAs(t, err, &duplicate)
	require.Equal(t, "23505", duplicate.Code)

	snapshot, err := service.List(harness.Context, string(organizationID), false)
	require.NoError(t, err)
	require.Equal(t, []string{"z", "a"}, []string{snapshot.Grades[0].Code, snapshot.Grades[1].Code}, "grade picker order uses ordinal, not code")
	require.Equal(t, []string{"Amber", "Blue"}, []string{snapshot.Homerooms[0].Name, snapshot.Homerooms[1].Name})
	require.Equal(t, "homeroom", snapshot.Settings.HomeroomLabel)

	retired := true
	_, err = service.UpdateGrade(harness.Context, string(organizationID), firstGrade.ID, actor, vocabulary.GradeLevelUpdate{Retired: &retired})
	require.NoError(t, err)
	_, err = service.UpdateHomeroom(harness.Context, string(organizationID), firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{Retired: &retired})
	require.NoError(t, err)
	updatedIdentifier := "blue-room-updated"
	updatedIdentifierValue := &updatedIdentifier
	updatedHomeroom, err := service.UpdateHomeroom(harness.Context, string(organizationID), firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{ExternalIdentifier: &updatedIdentifierValue})
	require.NoError(t, err)
	require.Equal(t, &updatedIdentifier, updatedHomeroom.ExternalIdentifier)

	snapshot, err = service.List(harness.Context, string(organizationID), false)
	require.NoError(t, err)
	require.Len(t, snapshot.Grades, 1)
	require.Len(t, snapshot.Homerooms, 1)
	retiredGrade, err := service.GetGrade(harness.Context, string(organizationID), firstGrade.ID)
	require.NoError(t, err)
	require.NotNil(t, retiredGrade.RetiredAt, "retired grade remains a valid reference")
	retiredHomeroom, err := service.GetHomeroom(harness.Context, string(organizationID), firstHomeroom.ID)
	require.NoError(t, err)
	require.NotNil(t, retiredHomeroom.RetiredAt, "retired homeroom remains a valid reference")

	retired = false
	_, err = service.UpdateGrade(harness.Context, string(organizationID), firstGrade.ID, actor, vocabulary.GradeLevelUpdate{Retired: &retired})
	require.NoError(t, err)
	_, err = service.UpdateHomeroom(harness.Context, string(organizationID), firstHomeroom.ID, actor, vocabulary.HomeroomUpdate{Retired: &retired})
	require.NoError(t, err)

	_, err = service.ReorderGrades(harness.Context, string(organizationID), actor, []ids.XID{secondGrade.ID, firstGrade.ID})
	require.NoError(t, err)
	snapshot, err = service.List(harness.Context, string(organizationID), false)
	require.NoError(t, err)
	require.Equal(t, secondGrade.ID, snapshot.Grades[0].ID)
	require.Equal(t, 1, snapshot.Grades[0].Ordinal)
	require.Equal(t, firstGrade.ID, snapshot.Grades[1].ID)
	require.Equal(t, 2, snapshot.Grades[1].Ordinal)

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
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Nullable Room", nil)
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
