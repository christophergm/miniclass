package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
)

const (
	CommentHostAssignment = "assignment"
	CommentHostOffering   = "offering"
	CommentHostSession    = "session"
)

// PlacementComment is an administrator-only note attached to a placement,
// offering, or session (SPEC §§10.5, 20.3–20.4).
type PlacementComment struct {
	ID, OrganizationID, SchoolYearID, ProgramID, SessionID ids.XID
	HostType                                               string
	HostID, AuthorUserID                                   ids.XID
	Body, Sensitivity                                      string
	DeletedAt                                              *time.Time
	DeletedByUserID                                        *ids.XID
	CreatedAt, UpdatedAt                                   time.Time
}

type CreatePlacementCommentInput struct {
	SchoolYearID, ProgramID, SessionID ids.XID
	HostType                           string
	HostID, AuthorUserID               ids.XID
	Body, Sensitivity                  string
}

func (tx *Tx) CreatePlacementComment(ctx context.Context, input CreatePlacementCommentInput) (PlacementComment, error) {
	if tx == nil || tx.queries == nil {
		return PlacementComment{}, errors.New("create placement comment: transaction is nil")
	}
	input.Body, input.Sensitivity = strings.TrimSpace(input.Body), strings.TrimSpace(input.Sensitivity)
	if input.Sensitivity == "" {
		input.Sensitivity = "internal"
	}
	if input.SchoolYearID == "" || input.ProgramID == "" || input.SessionID == "" || input.HostID == "" || input.AuthorUserID == "" || input.Body == "" {
		return PlacementComment{}, errors.New("create placement comment: required fields are missing")
	}
	if !validCommentHost(input.HostType) || !validCommentSensitivity(input.Sensitivity) {
		return PlacementComment{}, errors.New("create placement comment: host type or sensitivity is invalid")
	}
	row, err := tx.queries.CreatePlacementComment(ctx, db.CreatePlacementCommentParams{OrganizationID: tx.organizationID, SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID, HostType: input.HostType, HostID: input.HostID, AuthorUserID: input.AuthorUserID, Body: input.Body, Sensitivity: input.Sensitivity})
	if err != nil {
		return PlacementComment{}, fmt.Errorf("create placement comment: %w", err)
	}
	return placementComment(row)
}

func (tx *Tx) GetPlacementComment(ctx context.Context, schoolYearID, programID, sessionID, id ids.XID) (PlacementComment, error) {
	row, err := tx.queries.GetPlacementComment(ctx, db.GetPlacementCommentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return PlacementComment{}, fmt.Errorf("get placement comment: %w", err)
	}
	return placementComment(row)
}

func (tx *Tx) ListPlacementComments(ctx context.Context, schoolYearID, programID, sessionID ids.XID) ([]PlacementComment, error) {
	rows, err := tx.queries.ListPlacementComments(ctx, db.ListPlacementCommentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID})
	if err != nil {
		return nil, fmt.Errorf("list placement comments: %w", err)
	}
	result := make([]PlacementComment, 0, len(rows))
	for _, row := range rows {
		value, err := placementComment(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) UpdatePlacementComment(ctx context.Context, value PlacementComment, authorUserID ids.XID, body, sensitivity string) (PlacementComment, error) {
	body, sensitivity = strings.TrimSpace(body), strings.TrimSpace(sensitivity)
	if body == "" || !validCommentSensitivity(sensitivity) {
		return PlacementComment{}, errors.New("update placement comment: body or sensitivity is invalid")
	}
	row, err := tx.queries.UpdatePlacementComment(ctx, db.UpdatePlacementCommentParams{Body: body, Sensitivity: sensitivity, ID: value.ID, OrganizationID: tx.organizationID, SchoolYearID: value.SchoolYearID, ProgramID: value.ProgramID, SessionID: value.SessionID, AuthorUserID: authorUserID})
	if err != nil {
		return PlacementComment{}, fmt.Errorf("update placement comment: %w", err)
	}
	return placementComment(row)
}

func (tx *Tx) SoftDeletePlacementComment(ctx context.Context, value PlacementComment, authorUserID ids.XID) (PlacementComment, error) {
	row, err := tx.queries.SoftDeletePlacementComment(ctx, db.SoftDeletePlacementCommentParams{DeletedByUserID: &authorUserID, ID: value.ID, OrganizationID: tx.organizationID, SchoolYearID: value.SchoolYearID, ProgramID: value.ProgramID, SessionID: value.SessionID})
	if err != nil {
		return PlacementComment{}, fmt.Errorf("soft delete placement comment: %w", err)
	}
	return placementComment(row)
}

func validCommentHost(value string) bool {
	return value == CommentHostAssignment || value == CommentHostOffering || value == CommentHostSession
}
func validCommentSensitivity(value string) bool {
	return value == "public" || value == "internal" || value == "sensitive"
}

func placementComment(row db.PlacementComment) (PlacementComment, error) {
	created, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return PlacementComment{}, err
	}
	updated, err := programTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return PlacementComment{}, err
	}
	var deleted *time.Time
	if row.DeletedAt.Valid {
		value := row.DeletedAt.Time
		deleted = &value
	}
	return PlacementComment{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, HostType: row.HostType, HostID: row.HostID, AuthorUserID: row.AuthorUserID, Body: row.Body, Sensitivity: row.Sensitivity, DeletedAt: deleted, DeletedByUserID: row.DeletedByUserID, CreatedAt: created, UpdatedAt: updated}, nil
}

// The following methods are restricted to the Layer 2 isolation registry.
func (tx *Tx) ListAllPlacementCommentsForRegistry(ctx context.Context) ([]PlacementComment, error) {
	rows, err := tx.queries.ListAllPlacementCommentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]PlacementComment, 0, len(rows))
	for _, row := range rows {
		value, err := placementComment(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}
func (tx *Tx) FindPlacementCommentForRegistry(ctx context.Context, id ids.XID) (PlacementComment, error) {
	row, err := tx.queries.FindPlacementCommentForRegistry(ctx, db.FindPlacementCommentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		return PlacementComment{}, err
	}
	return placementComment(row)
}
func (tx *Tx) TouchPlacementCommentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.TouchPlacementCommentForRegistry(ctx, db.TouchPlacementCommentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
func (tx *Tx) DeletePlacementCommentForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	n, err := tx.queries.DeletePlacementCommentForRegistry(ctx, db.DeletePlacementCommentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return n > 0, err
}
