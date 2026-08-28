package integration

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestStudentCRUDSoftDeleteAndPriorYearLink(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "student integration test"}
	priorYear, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2025–2026")
	require.NoError(t, err)
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "four", "Grade Four")
	require.NoError(t, err)
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Room A")
	require.NoError(t, err)
	service := people.New(harness.Database)
	prior, err := service.CreateStudent(ctx, string(organizationID), priorYear.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Alex", LegalFamilyName: "Rivera", GradeLevelID: grade.ID, HomeroomID: homeroom.ID,
	})
	require.NoError(t, err)
	externalIdentifier := "student-1"
	created, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Alexander", LegalFamilyName: "Rivera", PreferredGivenName: stringPointer("Alex"),
		GradeLevelID: grade.ID, HomeroomID: homeroom.ID, ExternalIdentifier: &externalIdentifier,
		PriorYearStudentID: &prior.ID,
	})
	require.NoError(t, err)
	require.Equal(t, &prior.ID, created.PriorYearStudentID)

	listed, err := service.ListStudents(ctx, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Len(t, listed, 1)

	updated, err := service.UpdateStudent(ctx, string(organizationID), year.ID, created.ID, actor, people.StudentUpdateInput{LegalGivenName: stringPointer("Alec")})
	require.NoError(t, err)
	require.Equal(t, "Alec", updated.LegalGivenName)
	require.Equal(t, "Alex", *updated.PreferredGivenName)

	require.NoError(t, service.DeleteStudent(ctx, string(organizationID), year.ID, created.ID, actor))
	listed, err = service.ListStudents(ctx, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Empty(t, listed)
	_, err = service.GetStudent(ctx, string(organizationID), year.ID, created.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	deleted, err := service.ListStudents(ctx, string(organizationID), year.ID, true)
	require.NoError(t, err)
	require.Len(t, deleted, 1)
	require.NotNil(t, deleted[0].DeletedAt)
	restored, err := service.RestoreStudent(ctx, string(organizationID), year.ID, created.ID, actor, "corrected the deletion")
	require.NoError(t, err)
	require.Nil(t, restored.DeletedAt)
	listed, err = service.ListStudents(ctx, string(organizationID), year.ID, false)
	require.NoError(t, err)
	require.Len(t, listed, 1)
	require.NoError(t, service.DeleteStudent(ctx, string(organizationID), year.ID, created.ID, actor))

	replacement, err := service.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Jordan", LegalFamilyName: "Rivera", GradeLevelID: grade.ID, HomeroomID: homeroom.ID,
		ExternalIdentifier: &externalIdentifier,
	})
	require.NoError(t, err)
	require.NotEqual(t, created.ID, replacement.ID)

	var auditCount int64
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		auditCount, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, auditCount, int64(9))
}

func stringPointer(value string) *string { return &value }
