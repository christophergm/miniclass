package registry

import (
	"context"
	"errors"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/jackc/pgx/v5"
)

func init() {
	Register(Entity{TableName: "placement_comments", YearScoped: true, Factory: createPlacementComment,
		ReadIDs: readPlacementCommentIDs, FetchByID: fetchPlacementComment, UpdateByID: touchPlacementComment,
		DeleteByID: deletePlacementComment, InsertWithForeignParent: insertForeignPlacementComment})
}

func placementCommentAuthor(ctx context.Context, harness *testharness.Harness) (ids.XID, error) {
	var id ids.XID
	err := harness.Migrator.QueryRow(ctx, `insert into users (provider_subject, email) values ('placement-comment-registry-' || public.xid(), 'placement-comment-registry-' || public.xid() || '@example.test') returning id`).Scan(&id)
	return id, err
}

func createPlacementComment(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	assignment, err := assignmentFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	author, err := placementCommentAuthor(ctx, harness)
	if err != nil {
		return "", err
	}
	var result data.PlacementComment
	err = harness.Database.InTenant(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "placement comment fixture"}, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.CreatePlacementComment(ctx, data.CreatePlacementCommentInput{SchoolYearID: assignment.SchoolYearID, ProgramID: assignment.ProgramID, SessionID: assignment.SessionID, HostType: data.CommentHostAssignment, HostID: assignment.ID, AuthorUserID: author, Body: "synthetic placement comment", Sensitivity: "internal"})
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "placement_comment", ObjectID: &result.ID, SchoolYearID: &assignment.SchoolYearID, ChangeSummary: []byte(`{"fixture":true}`)})
	})
	return result.ID, err
}

func readPlacementCommentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllPlacementCommentsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchPlacementComment(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindPlacementCommentForRegistry(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return row.ID != "", err
}
func touchPlacementComment(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchPlacementCommentForRegistry(ctx, id)
}
func deletePlacementComment(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeletePlacementCommentForRegistry(ctx, id)
}

func insertForeignPlacementComment(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	assignment, err := assignmentFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	author, err := placementCommentAuthor(ctx, harness)
	if err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into placement_comments (organization_id, school_year_id, program_id, session_id, host_type, host_id, author_user_id, body) values ($1, $2, $3, $4, 'assignment', $5, $6, 'foreign probe')`, foreignOrganizationID, assignment.SchoolYearID, assignment.ProgramID, assignment.SessionID, assignment.ID, author)
	return err
}
