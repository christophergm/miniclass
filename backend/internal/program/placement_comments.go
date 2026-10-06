package program

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var (
	ErrPlacementCommentNotFound   = errors.New("placement comment not found")
	ErrPlacementCommentAuthorOnly = errors.New("only the comment author may edit or delete it")
)

type PlacementCommentInput struct {
	SchoolYearID, ProgramID, SessionID ids.XID
	HostType                           string
	HostID                             ids.XID
	Body, Sensitivity                  string
}

func (s *Service) CreatePlacementComment(ctx context.Context, organizationID string, actor audit.Actor, input PlacementCommentInput) (data.PlacementComment, error) {
	if actor.UserID == nil {
		return data.PlacementComment{}, errors.New("create placement comment: administrator identity is required")
	}
	var result data.PlacementComment
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if err := validateCommentHost(ctx, tx, input); err != nil {
			return err
		}
		created, err := tx.CreatePlacementComment(ctx, data.CreatePlacementCommentInput{SchoolYearID: input.SchoolYearID, ProgramID: input.ProgramID, SessionID: input.SessionID, HostType: input.HostType, HostID: input.HostID, AuthorUserID: *actor.UserID, Body: input.Body, Sensitivity: input.Sensitivity})
		if err != nil {
			return err
		}
		result = created
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "placement_comment", ObjectID: &created.ID, SchoolYearID: &input.SchoolYearID, ChangeSummary: mustJSON(map[string]any{"host_type": created.HostType, "host_id": created.HostID, "body": created.Body, "sensitivity": created.Sensitivity})})
	})
	if err != nil {
		return data.PlacementComment{}, fmt.Errorf("create placement comment: %w", err)
	}
	return result, nil
}

func (s *Service) UpdatePlacementComment(ctx context.Context, organizationID string, actor audit.Actor, input PlacementCommentInput, commentID ids.XID) (data.PlacementComment, error) {
	return s.changePlacementComment(ctx, organizationID, actor, input, commentID)
}

func (s *Service) DeletePlacementComment(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, sessionID, commentID ids.XID) (data.PlacementComment, error) {
	if actor.UserID == nil {
		return data.PlacementComment{}, ErrPlacementCommentAuthorOnly
	}
	var result data.PlacementComment
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetPlacementComment(ctx, schoolYearID, programID, sessionID, commentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPlacementCommentNotFound
		}
		if err != nil {
			return err
		}
		if current.AuthorUserID != *actor.UserID || current.DeletedAt != nil {
			return ErrPlacementCommentAuthorOnly
		}
		deleted, err := tx.SoftDeletePlacementComment(ctx, current, *actor.UserID)
		if err != nil {
			return err
		}
		result = deleted
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSoftDelete, ObjectType: "placement_comment", ObjectID: &current.ID, SchoolYearID: &schoolYearID, ChangeSummary: mustJSON(map[string]any{"host_type": current.HostType, "host_id": current.HostID, "body": current.Body, "sensitivity": current.Sensitivity})})
	})
	if err != nil {
		return data.PlacementComment{}, fmt.Errorf("delete placement comment: %w", err)
	}
	return result, nil
}

func (s *Service) changePlacementComment(ctx context.Context, organizationID string, actor audit.Actor, input PlacementCommentInput, commentID ids.XID) (data.PlacementComment, error) {
	if actor.UserID == nil {
		return data.PlacementComment{}, ErrPlacementCommentAuthorOnly
	}
	var result data.PlacementComment
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetPlacementComment(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, commentID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPlacementCommentNotFound
		}
		if err != nil {
			return err
		}
		if current.AuthorUserID != *actor.UserID || current.DeletedAt != nil {
			return ErrPlacementCommentAuthorOnly
		}
		if strings.TrimSpace(input.Sensitivity) == "" {
			input.Sensitivity = current.Sensitivity
		}
		updated, err := tx.UpdatePlacementComment(ctx, current, *actor.UserID, input.Body, input.Sensitivity)
		if err != nil {
			return err
		}
		result = updated
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "placement_comment", ObjectID: &current.ID, SchoolYearID: &input.SchoolYearID, ChangeSummary: mustJSON(map[string]any{"before": map[string]any{"body": current.Body, "sensitivity": current.Sensitivity}, "after": map[string]any{"body": updated.Body, "sensitivity": updated.Sensitivity}})})
	})
	if err != nil {
		return data.PlacementComment{}, fmt.Errorf("update placement comment: %w", err)
	}
	return result, nil
}

func validateCommentHost(ctx context.Context, tx *data.Tx, input PlacementCommentInput) error {
	if _, err := tx.GetSession(ctx, input.SchoolYearID, input.ProgramID, input.SessionID); err != nil {
		return err
	}
	switch input.HostType {
	case data.CommentHostSession:
		if input.HostID != input.SessionID {
			return errors.New("create placement comment: session host must match the workspace session")
		}
	case data.CommentHostOffering:
		if _, err := tx.GetOffering(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, input.HostID); err != nil {
			return err
		}
	case data.CommentHostAssignment:
		assignments, err := tx.ListAssignments(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		for _, assignment := range assignments {
			if assignment.ID == input.HostID {
				return nil
			}
		}
		return errors.New("create placement comment: assignment host not found")
	default:
		return errors.New("create placement comment: host type is invalid")
	}
	return nil
}
