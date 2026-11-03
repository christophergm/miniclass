package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/api"
	"github.com/chrismott/miniclass/internal/api/handlers"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/identity"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestPlacementCommentsAreAuthorOnlyAndStayWithTheirOriginalAssignment(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	authorID := placementCommentTestUser(t, ctx, harness, "author")
	otherID := placementCommentTestUser(t, ctx, harness, "other")
	author := audit.Actor{Type: audit.ActorTypeUser, UserID: &authorID, Label: "author@example.test"}
	factory := factories.New(harness.Database, string(organizationID), author)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic placement comment year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "comments", "Synthetic Comments Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Comments Room")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic comments program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic comments session", []time.Time{time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Comment", LegalFamilyName: "Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
	require.NoError(t, err)
	first, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic first offering", "", nil, 2, grade.ID, grade.ID, "", "", "", nil)
	require.NoError(t, err)
	second, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic second offering", "", nil, 2, grade.ID, grade.ID, "", "", "", nil)
	require.NoError(t, err)

	service := program.New(harness.Database)
	moved, err := service.MoveAssignment(ctx, string(organizationID), author, program.AssignmentOperationInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID}, student.ID, first.ID)
	require.NoError(t, err)
	require.Len(t, moved.Assignments, 1)
	comment, err := service.CreatePlacementComment(ctx, string(organizationID), author, program.PlacementCommentInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, HostType: data.CommentHostAssignment, HostID: moved.Assignments[0].ID, Body: "Accepted after organizer review", Sensitivity: "sensitive"})
	require.NoError(t, err)
	_, err = service.UpdatePlacementComment(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeUser, UserID: &otherID, Label: "other@example.test"}, program.PlacementCommentInput{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, Body: "not permitted", Sensitivity: "internal"}, comment.ID)
	require.ErrorIs(t, err, program.ErrPlacementCommentAuthorOnly)

	// A same-offering replacement preserves the assignment identity and comment.
	require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), author, func(ctx context.Context, tx *data.Tx) error {
		rows, err := tx.ReplaceDraftAssignments(ctx, year.ID, programRow.ID, session.ID, []data.CreateAssignmentInput{{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, StudentID: student.ID, OfferingID: first.ID, Origin: "manual", Pinned: true, RealizedQuality: "neutral"}})
		if err != nil {
			return err
		}
		require.Equal(t, moved.Assignments[0].ID, rows[0].ID)
		return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", SchoolYearID: &year.ID, ChangeSummary: []byte(`{"synthetic":true}`)})
	}))
	// A changed placement receives a new assignment identity. The comment remains
	// recorded against the prior one and is never transferred.
	require.NoError(t, harness.Database.InTenant(ctx, string(organizationID), author, func(ctx context.Context, tx *data.Tx) error {
		rows, err := tx.ReplaceDraftAssignments(ctx, year.ID, programRow.ID, session.ID, []data.CreateAssignmentInput{{SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, StudentID: student.ID, OfferingID: second.ID, Origin: "manual", Pinned: true, RealizedQuality: "neutral"}})
		if err != nil {
			return err
		}
		require.NotEqual(t, moved.Assignments[0].ID, rows[0].ID)
		comments, err := tx.ListPlacementComments(ctx, year.ID, programRow.ID, session.ID)
		if err != nil {
			return err
		}
		require.Len(t, comments, 1)
		require.Equal(t, moved.Assignments[0].ID, comments[0].HostID)
		return tx.Record(ctx, audit.Entry{Action: audit.ActionManualOperation, ObjectType: "assignment", SchoolYearID: &year.ID, ChangeSummary: []byte(`{"synthetic":true}`)})
	}))
	_, err = service.DeletePlacementComment(ctx, string(organizationID), author, year.ID, programRow.ID, session.ID, comment.ID)
	require.NoError(t, err)
	require.NoError(t, harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		comments, err := tx.ListPlacementComments(ctx, year.ID, programRow.ID, session.ID)
		require.NoError(t, err)
		require.Empty(t, comments)
		return nil
	}))
}

func TestAssignmentWorkspaceOmitsSensitiveCommentsForNonAdministrators(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	adminSubject := "sensitive-comments-admin-" + string(organizationID)
	coordinatorSubject := "sensitive-comments-coordinator-" + string(organizationID)
	adminEmail := adminSubject + "@example.test"
	coordinatorEmail := coordinatorSubject + "@example.test"
	var adminID ids.XID
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, adminSubject, adminEmail).Scan(&adminID))
	_, err := harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'administrator')`, organizationID, adminID)
	require.NoError(t, err)
	var coordinatorID ids.XID
	require.NoError(t, harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, coordinatorSubject, coordinatorEmail).Scan(&coordinatorID))
	_, err = harness.Migrator.Exec(ctx, `
		insert into organization_members (organization_id, user_id, role)
		values ($1, $2, 'coordinator')`, organizationID, coordinatorID)
	require.NoError(t, err)

	actor := audit.Actor{Type: audit.ActorTypeUser, UserID: &adminID, Label: adminEmail}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, "Synthetic sensitive-comment year")
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic sensitive-comment program")
	require.NoError(t, err)
	session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic sensitive-comment session", []time.Time{time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	service := program.New(harness.Database)
	comment, err := service.CreatePlacementComment(ctx, string(organizationID), actor, program.PlacementCommentInput{
		SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID,
		HostType: data.CommentHostSession, HostID: session.ID,
		Body: "Synthetic restricted placement reasoning", Sensitivity: "sensitive",
	})
	require.NoError(t, err)

	verifier, tokens := localTestVerifierTokens(t,
		localTestIdentity{Subject: adminSubject, Email: adminEmail},
		localTestIdentity{Subject: coordinatorSubject, Email: coordinatorEmail},
	)
	server := api.NewServer(
		api.WithDatabase(harness.Database),
		api.WithIdentity(identity.NewStore(harness.Database)),
		api.WithPrograms(service),
		api.WithVerifier(verifier),
	)
	path := "/api/school-years/" + string(year.ID) + "/programs/" + string(programRow.ID) + "/sessions/" + string(session.ID) + "/assignment-workspace"
	for _, test := range []struct {
		name      string
		subject   string
		wantCount int
	}{
		{name: "coordinator payload omits sensitive comment", subject: coordinatorSubject, wantCount: 0},
		{name: "administrator payload includes sensitive comment", subject: adminSubject, wantCount: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("Authorization", "Bearer "+tokens[test.subject])
			recording := httptest.NewRecorder()
			server.Handler().ServeHTTP(recording, request)

			require.Equal(t, http.StatusOK, recording.Code, recording.Body.String())
			var payload handlers.AssignmentWorkspaceResponse
			require.NoError(t, json.NewDecoder(recording.Body).Decode(&payload))
			require.Len(t, payload.Comments, test.wantCount)
			if test.wantCount == 0 {
				require.NotContains(t, recording.Body.String(), comment.Body)
				return
			}
			require.Equal(t, comment.ID, ids.XID(payload.Comments[0].ID))
			require.Equal(t, comment.Body, payload.Comments[0].Body)
		})
	}
}

func placementCommentTestUser(t *testing.T, ctx context.Context, harness *testharness.Harness, suffix string) ids.XID {
	t.Helper()
	var id ids.XID
	require.NoError(t, harness.Migrator.QueryRow(ctx, `insert into users (provider_subject, email) values ($1, $2) returning id`, "placement-comment-"+suffix+"-"+string(ids.XID("x")), "placement-comment-"+suffix+"@example.test").Scan(&id))
	return id
}
