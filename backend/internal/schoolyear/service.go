// Package schoolyear owns school-year lifecycle rules and audited mutations.
package schoolyear

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var (
	// ErrInvalidTransition means that the requested state edge is not part of
	// the school-year lifecycle in SPEC §11.1.
	ErrInvalidTransition = errors.New("invalid school year state transition")
	// ErrRoleRequired identifies the Owner/Administrator transition boundary.
	ErrRoleRequired = errors.New("owner or administrator role is required")
	// ErrOwnerRequired identifies the exceptional closed-year reopen boundary.
	ErrOwnerRequired = errors.New("owner role is required to reopen a school year")
	// ErrReasonRequired identifies an unaudited or unexplained reopen attempt.
	ErrReasonRequired = errors.New("reason is required to reopen a school year")
	// ErrNoChanges avoids opening a write transaction that cannot produce an
	// audit entry.
	ErrNoChanges = errors.New("school year update has no changes")
)

// UpdateInput contains the independently optional CRUD and lifecycle fields.
// A non-nil State is a lifecycle request; Reason is used only for reopen.
type UpdateInput struct {
	Label  *string
	State  *data.SchoolYearState
	Reason string
}

// Service is the school-year application service over the tenant data boundary.
type Service struct {
	database *data.DB
}

// New creates a school-year service. A nil database is retained so API
// contract generation can register operations without opening a connection.
func New(database *data.DB) *Service {
	return &Service{database: database}
}

// Create creates a setup year and records its creation in the same unit of
// work. The caller has already passed the manage-school-year capability gate.
func (s *Service) Create(ctx context.Context, organizationID string, actor audit.Actor, label string) (data.SchoolYear, error) {
	if s == nil || s.database == nil {
		return data.SchoolYear{}, errors.New("create school year: data service is nil")
	}
	var result data.SchoolYear
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		created, err := tx.CreateSchoolYear(ctx, label)
		if err != nil {
			return err
		}
		result = created
		id := created.ID
		return tx.Record(ctx, audit.Entry{
			Action: audit.ActionSchoolYearCreate, ObjectType: "school_year", ObjectID: &id,
			SchoolYearID: &id, ChangeSummary: summary(map[string]any{
				"label": created.Label, "state": string(created.State),
			}),
		})
	})
	if err != nil {
		return data.SchoolYear{}, fmt.Errorf("create school year: %w", err)
	}
	return result, nil
}

// List returns the current tenant's years. The read path is transaction-local
// and does not require an actor or audit entry.
func (s *Service) List(ctx context.Context, organizationID string) ([]data.SchoolYear, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list school years: data service is nil")
	}
	var result []data.SchoolYear
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.ListSchoolYears(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list school years: %w", err)
	}
	return result, nil
}

// Get fetches one year in the current tenant.
func (s *Service) Get(ctx context.Context, organizationID string, id ids.XID) (data.SchoolYear, error) {
	if s == nil || s.database == nil {
		return data.SchoolYear{}, errors.New("get school year: data service is nil")
	}
	var result data.SchoolYear
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetSchoolYearByID(ctx, id)
		return err
	})
	if err != nil {
		return data.SchoolYear{}, fmt.Errorf("get school year: %w", err)
	}
	return result, nil
}

// Update applies a label edit and/or one validated lifecycle transition in a
// single audited transaction.
func (s *Service) Update(ctx context.Context, organizationID string, id ids.XID, role auth.OrganizationRole, actor audit.Actor, input UpdateInput) (data.SchoolYear, error) {
	if s == nil || s.database == nil {
		return data.SchoolYear{}, errors.New("update school year: data service is nil")
	}
	if input.Label == nil && input.State == nil {
		return data.SchoolYear{}, ErrNoChanges
	}
	if input.State != nil && !validState(*input.State) {
		return data.SchoolYear{}, fmt.Errorf("%w: unknown target state %q", ErrInvalidTransition, *input.State)
	}

	var result data.SchoolYear
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSchoolYearByID(ctx, id)
		if err != nil {
			return err
		}
		if input.State != nil && *input.State != current.State {
			if err := validateTransition(current.State, *input.State, role, input.Reason); err != nil {
				return err
			}
			if current.State == data.SchoolYearClosed {
				if err := tx.PrepareSchoolYearReopen(ctx, id, input.Reason); err != nil {
					return err
				}
			}
			changed, err := tx.UpdateSchoolYearState(ctx, id, *input.State)
			if err != nil {
				return err
			}
			result = changed
			before, after := string(current.State), string(changed.State)
			if err := tx.Record(ctx, audit.Entry{
				Action: audit.ActionSchoolYearStateTransition, ObjectType: "school_year", ObjectID: &id,
				SchoolYearID: &id, Reason: strings.TrimSpace(input.Reason), ChangeSummary: summary(map[string]any{
					"before": map[string]string{"state": before},
					"after":  map[string]string{"state": after},
				}),
			}); err != nil {
				return err
			}
			current = changed
		}

		if input.Label != nil && strings.TrimSpace(*input.Label) != current.Label {
			changed, err := tx.UpdateSchoolYearLabel(ctx, id, *input.Label)
			if err != nil {
				return err
			}
			result = changed
			if err := tx.Record(ctx, audit.Entry{
				Action: audit.ActionEdit, ObjectType: "school_year", ObjectID: &id,
				SchoolYearID: &id, ChangeSummary: summary(map[string]any{
					"before": map[string]string{"label": current.Label},
					"after":  map[string]string{"label": changed.Label},
				}),
			}); err != nil {
				return err
			}
			current = changed
		}
		if result.ID == "" {
			return ErrNoChanges
		}
		return nil
	})
	if err != nil {
		return data.SchoolYear{}, fmt.Errorf("update school year: %w", err)
	}
	return result, nil
}

// Delete hard-deletes an open year and records the action. A closed year is
// rejected by the shared database trigger and mapped by the API as a 409.
func (s *Service) Delete(ctx context.Context, organizationID string, id ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("delete school year: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetSchoolYearByID(ctx, id)
		if err != nil {
			return err
		}
		rows, err := tx.DeleteSchoolYear(ctx, id)
		if err != nil {
			return err
		}
		if rows != 1 {
			return pgx.ErrNoRows
		}
		return tx.Record(ctx, audit.Entry{
			Action: audit.ActionHardDelete, ObjectType: "school_year", ObjectID: &id,
			SchoolYearID: &id, ChangeSummary: summary(map[string]any{
				"label": current.Label, "state": string(current.State),
			}),
		})
	})
	if err != nil {
		return fmt.Errorf("delete school year: %w", err)
	}
	return nil
}

func validateTransition(from, to data.SchoolYearState, role auth.OrganizationRole, reason string) error {
	if from == to {
		return fmt.Errorf("%w: state is already %q", ErrInvalidTransition, to)
	}
	if role != auth.RoleOwner && role != auth.RoleAdministrator {
		return ErrRoleRequired
	}
	switch {
	case from == data.SchoolYearSetup && to == data.SchoolYearActive:
		return nil
	case from == data.SchoolYearActive && to == data.SchoolYearClosed:
		return nil
	case from == data.SchoolYearActive && to == data.SchoolYearSetup:
		return fmt.Errorf("%w: active years cannot return to setup", ErrInvalidTransition)
	case from == data.SchoolYearClosed && to == data.SchoolYearActive:
		if role != auth.RoleOwner {
			return ErrOwnerRequired
		}
		if strings.TrimSpace(reason) == "" {
			return ErrReasonRequired
		}
		return nil
	default:
		return fmt.Errorf("%w: %s to %s", ErrInvalidTransition, from, to)
	}
}

func validState(state data.SchoolYearState) bool {
	switch state {
	case data.SchoolYearSetup, data.SchoolYearActive, data.SchoolYearClosed:
		return true
	default:
		return false
	}
}

func summary(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode audit summary"}`)
	}
	return encoded
}
