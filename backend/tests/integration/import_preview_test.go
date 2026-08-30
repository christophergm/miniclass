package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/ingest"
	"github.com/chrismott/miniclass/internal/ingest/roster"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

// TestImportPreviewUsesTenantReadAndClassifiesRepeatAndEdit exercises the
// database-backed preview boundary from SPEC §§9.2, 9.4, 11.1, and 11.5–11.7.
// The preview must not create an audit entry, and a school year from another
// organization must remain indistinguishable from a missing year.
func TestImportPreviewUsesTenantReadAndClassifiesRepeatAndEdit(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	otherOrganizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "import preview integration test"}

	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	roomExternalIdentifier := "room-1"
	room, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Synthetic Room", &roomExternalIdentifier)
	require.NoError(t, err)

	adultExternalIdentifier := "adult-1"
	studentExternalIdentifier := "student-1"
	service := ingest.NewPreviewService(harness.Database)
	document := []byte(`[{"id":"adult-1","given_name":"Alex","family_name":"Rivera","email":"alex@example.test","relationships":[{"relationship":"MOM","child":{"id":"student-1","given_name":"Sam","family_name":"Rivera","groups":[{"class":"classroom","id":"room-1","label":"Synthetic Room","band":"K"}]}}]}]`)

	auditBefore := countAuditEntries(t, harness.Database, ctx, organizationID)
	fresh, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, document)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Create: 3}, fresh.Counts)
	require.Equal(t, ingest.ContentHash(document), fresh.ContentHash)
	require.Len(t, fresh.Rows, 1)
	require.Len(t, fresh.Rows[0].Records, 3)
	require.Equal(t, auditBefore, countAuditEntries(t, harness.Database, ctx, organizationID), "preview is read-only")
	repeatedFresh, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, document)
	require.NoError(t, err)
	require.Equal(t, fresh, repeatedFresh, "a repeated preview is stateless")

	student, err := people.New(harness.Database).CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
		LegalGivenName: "Sam", LegalFamilyName: "Rivera", HomeroomID: room.ID, ExternalIdentifier: &studentExternalIdentifier,
	})
	require.NoError(t, err)
	adult, err := people.New(harness.Database).Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
		LegalGivenName: "Alex", LegalFamilyName: "Rivera", Email: stringPointer("alex@example.test"), ExternalIdentifier: &adultExternalIdentifier,
	})
	require.NoError(t, err)
	_, err = people.New(harness.Database).CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
		AdultID: adult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent,
	})
	require.NoError(t, err)

	repeated, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, document)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Unchanged: 3}, repeated.Counts)

	editedEmail := "edited@example.test"
	_, err = people.New(harness.Database).Update(ctx, string(organizationID), year.ID, adult.ID, actor, people.AdultUpdateInput{
		Email: editedEmailPatch(editedEmail),
	})
	require.NoError(t, err)
	edited, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, document)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Update: 1, Unchanged: 2}, edited.Counts)
	require.Equal(t, "email", edited.Rows[0].Records[0].Changes[0].Field)
	require.Equal(t, "edited@example.test", edited.Rows[0].Records[0].Changes[0].Before)
	require.Equal(t, "alex@example.test", edited.Rows[0].Records[0].Changes[0].After)

	_, err = service.Preview(ctx, string(otherOrganizationID), year.ID, ingest.KindRosterJSON, document)
	require.Error(t, err)
	require.True(t, errors.Is(err, pgx.ErrNoRows), "foreign year = %v", err)
}

// TestImportPreviewGoldenCorpusAgainstDatabase verifies the complete source
// distribution through the real tenant read path, including a simulated
// committed corpus and one subsequent hand edit. The commit itself belongs to
// P2-5; these normal domain-service writes only establish the state that its
// preview would later inspect.
func TestImportPreviewGoldenCorpusAgainstDatabase(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "golden import preview integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)

	encoded, err := os.ReadFile(filepath.Join("..", "..", "internal", "ingest", "roster", "testdata", "synthetic_roster.json"))
	require.NoError(t, err)
	document, err := roster.ParseDocument(encoded)
	require.NoError(t, err)

	vocabularyService := vocabulary.New(harness.Database)
	homerooms := make(map[string]ids.XID, roster.SyntheticClassroomCnt)
	for _, source := range document.Result.Students {
		if _, exists := homerooms[source.ClassroomExternalIdentifier]; exists {
			continue
		}
		externalIdentifier := source.ClassroomExternalIdentifier
		room, createErr := vocabularyService.CreateHomeroom(ctx, string(organizationID), actor, source.ClassroomLabel, &externalIdentifier)
		require.NoError(t, createErr)
		homerooms[externalIdentifier] = room.ID
	}
	require.Len(t, homerooms, roster.SyntheticClassroomCnt)

	service := ingest.NewPreviewService(harness.Database)
	auditBefore := countAuditEntries(t, harness.Database, ctx, organizationID)
	fresh, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, encoded)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Create: 714}, fresh.Counts)
	require.Len(t, fresh.Rows, 226)
	require.Len(t, fresh.Exclusions, 160)
	require.Equal(t, auditBefore, countAuditEntries(t, harness.Database, ctx, organizationID), "golden preview is read-only")

	repeatedFresh, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, encoded)
	require.NoError(t, err)
	require.Equal(t, fresh, repeatedFresh, "repeated golden preview is stateless")

	peopleService := people.New(harness.Database)
	adultIDs := make(map[string]ids.XID, len(document.Result.Adults))
	for _, source := range document.Result.Adults {
		externalIdentifier, email := source.SourceExternalIdentifier, source.Email
		adult, createErr := peopleService.Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{
			LegalGivenName: source.GivenName, LegalFamilyName: source.FamilyName,
			Email: &email, ExternalIdentifier: &externalIdentifier,
		})
		require.NoError(t, createErr)
		adultIDs[externalIdentifier] = adult.ID
	}
	studentIDs := make(map[string]ids.XID, len(document.Result.Students))
	for _, source := range document.Result.Students {
		externalIdentifier := source.SourceExternalIdentifier
		student, createErr := peopleService.CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{
			LegalGivenName: source.GivenName, LegalFamilyName: source.FamilyName,
			HomeroomID: homerooms[source.ClassroomExternalIdentifier], ExternalIdentifier: &externalIdentifier,
		})
		require.NoError(t, createErr)
		studentIDs[externalIdentifier] = student.ID
	}
	for _, source := range document.Result.GuardianRelationships {
		_, createErr := peopleService.CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{
			AdultID: adultIDs[source.AdultExternalIdentifier], StudentID: studentIDs[source.StudentExternalIdentifier],
			RelationshipType: data.GuardianRelationshipType(source.RelationshipType),
		})
		require.NoError(t, createErr)
	}

	repeated, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, encoded)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Unchanged: 714}, repeated.Counts)

	handEditedEmail := "hand-edited-golden@example.test"
	handEditedEmailPatch := &handEditedEmail
	_, err = peopleService.Update(ctx, string(organizationID), year.ID, adultIDs[document.Result.Adults[0].SourceExternalIdentifier], actor, people.AdultUpdateInput{Email: &handEditedEmailPatch})
	require.NoError(t, err)
	handEdited, err := service.Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, encoded)
	require.NoError(t, err)
	require.Equal(t, ingest.OutcomeCounts{Update: 1, Unchanged: 713}, handEdited.Counts)
	require.Equal(t, []ingest.FieldChange{{Field: "email", Before: "hand-edited-golden@example.test", After: document.Result.Adults[0].Email}}, handEdited.Rows[0].Records[0].Changes)
}

func TestImportPreviewRejectsClosedSchoolYear(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "closed import preview integration test"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, "2026–2027")
	require.NoError(t, err)
	closed := data.SchoolYearClosed
	_, err = schoolyear.New(harness.Database).Update(ctx, string(organizationID), year.ID, auth.RoleAdministrator, actor, schoolyear.UpdateInput{State: &closed})
	require.NoError(t, err)

	_, err = ingest.NewPreviewService(harness.Database).Preview(ctx, string(organizationID), year.ID, ingest.KindRosterJSON, []byte(`[{"id":"adult-1","given_name":"Alex","family_name":"Rivera","relationships":[]}]`))
	require.Error(t, err)
	require.True(t, errors.Is(err, ingest.ErrSchoolYearClosed), "closed year = %v", err)
}

func countAuditEntries(t *testing.T, database *data.DB, ctx context.Context, organizationID ids.XID) int64 {
	t.Helper()
	var count int64
	err := database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		count, err = tx.Queries().CountAuditLog(ctx)
		return err
	})
	require.NoError(t, err)
	return count
}

func editedEmailPatch(value string) **string {
	result := &value
	return &result
}
