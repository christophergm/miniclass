// Package people owns roster person services. Adult identity is independent
// of household membership so the later access decision remains open.
package people

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

var ErrNoChanges = errors.New("adult update has no changes")

type AdultCreateInput struct {
	LegalGivenName      string
	LegalFamilyName     string
	PreferredGivenName  *string
	Email               *string
	Phone               *string
	ExternalIdentifier  *string
	ParticipationIntent data.AdultParticipationIntent
}

type AdultUpdateInput struct {
	LegalGivenName      *string
	LegalFamilyName     *string
	PreferredGivenName  **string
	Email               **string
	Phone               **string
	ExternalIdentifier  **string
	ParticipationIntent *data.AdultParticipationIntent
}

type Service struct {
	database *data.DB
}

func New(database *data.DB) *Service {
	return &Service{database: database}
}

func (s *Service) Create(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, input AdultCreateInput) (data.Adult, error) {
	if s == nil || s.database == nil {
		return data.Adult{}, errors.New("create adult: data service is nil")
	}
	var result data.Adult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		created, err := tx.CreateAdult(ctx, schoolYearID, input.LegalGivenName, input.LegalFamilyName, input.PreferredGivenName, input.Email, input.Phone, input.ExternalIdentifier, input.ParticipationIntent)
		if err != nil {
			return err
		}
		result = created
		id := created.ID
		year := created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionCreate, ObjectType: "adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: adultSummary(nil, &created)})
	})
	if err != nil {
		return data.Adult{}, fmt.Errorf("create adult: %w", err)
	}
	return result, nil
}

func (s *Service) List(ctx context.Context, organizationID string, schoolYearID ids.XID, includeDeleted bool) ([]data.Adult, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("list adults: data service is nil")
	}
	var result []data.Adult
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		var err error
		result, err = tx.ListAdults(ctx, schoolYearID, includeDeleted)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list adults: %w", err)
	}
	return result, nil
}

func (s *Service) Restore(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, reason string) (data.Adult, error) {
	if s == nil || s.database == nil {
		return data.Adult{}, errors.New("restore adult: data service is nil")
	}
	reason, err := restoreReason(reason)
	if err != nil {
		return data.Adult{}, err
	}
	var result data.Adult
	err = s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetAdultByIDIncludingDeleted(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if current.DeletedAt == nil {
			return ErrRestoreNotDeleted
		}
		restored, err := tx.RestoreAdult(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		result = restored
		year := restored.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionRestore, ObjectType: "adult", ObjectID: &id, SchoolYearID: &year, Reason: reason, ChangeSummary: adultSummary(&current, &restored)})
	})
	if err != nil {
		return data.Adult{}, fmt.Errorf("restore adult: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.Adult, error) {
	if s == nil || s.database == nil {
		return data.Adult{}, errors.New("get adult: data service is nil")
	}
	var result data.Adult
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetAdultByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.Adult{}, fmt.Errorf("get adult: %w", err)
	}
	return result, nil
}

func (s *Service) Update(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input AdultUpdateInput) (data.Adult, error) {
	if s == nil || s.database == nil {
		return data.Adult{}, errors.New("update adult: data service is nil")
	}
	var result data.Adult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetAdultByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		updatedInput, changed := applyAdultUpdate(current, input)
		if !changed {
			return ErrNoChanges
		}
		updated, err := tx.UpdateAdult(ctx, schoolYearID, id, updatedInput.LegalGivenName, updatedInput.LegalFamilyName, updatedInput.PreferredGivenName, updatedInput.Email, updatedInput.Phone, updatedInput.ExternalIdentifier, updatedInput.ParticipationIntent)
		if err != nil {
			return err
		}
		result = updated
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionEdit, ObjectType: "adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: adultSummary(&current, &updated)})
	})
	if err != nil {
		return data.Adult{}, fmt.Errorf("update adult: %w", err)
	}
	return result, nil
}

func (s *Service) Delete(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor) error {
	if s == nil || s.database == nil {
		return errors.New("delete adult: data service is nil")
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetAdultByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		deleted, err := tx.SoftDeleteAdult(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		if !deleted {
			return pgx.ErrNoRows
		}
		year := current.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSoftDelete, ObjectType: "adult", ObjectID: &id, SchoolYearID: &year, ChangeSummary: adultSummary(&current, nil)})
	})
	if err != nil {
		return fmt.Errorf("delete adult: %w", err)
	}
	return nil
}

func applyAdultUpdate(current data.Adult, input AdultUpdateInput) (AdultCreateInput, bool) {
	result := AdultCreateInput{
		LegalGivenName: current.LegalGivenName, LegalFamilyName: current.LegalFamilyName,
		PreferredGivenName: current.PreferredGivenName, Email: current.Email, Phone: current.Phone,
		ExternalIdentifier: current.ExternalIdentifier, ParticipationIntent: current.ParticipationIntent,
	}
	changed := false
	if input.LegalGivenName != nil && strings.TrimSpace(*input.LegalGivenName) != current.LegalGivenName {
		result.LegalGivenName, changed = *input.LegalGivenName, true
	}
	if input.LegalFamilyName != nil && strings.TrimSpace(*input.LegalFamilyName) != current.LegalFamilyName {
		result.LegalFamilyName, changed = *input.LegalFamilyName, true
	}
	if input.PreferredGivenName != nil && !sameAdultOptional(result.PreferredGivenName, *input.PreferredGivenName) {
		result.PreferredGivenName, changed = *input.PreferredGivenName, true
	}
	if input.Email != nil && !sameAdultOptional(result.Email, *input.Email) {
		result.Email, changed = *input.Email, true
	}
	if input.Phone != nil && !sameAdultOptional(result.Phone, *input.Phone) {
		result.Phone, changed = *input.Phone, true
	}
	if input.ExternalIdentifier != nil && !sameAdultOptional(result.ExternalIdentifier, *input.ExternalIdentifier) {
		result.ExternalIdentifier, changed = *input.ExternalIdentifier, true
	}
	if input.ParticipationIntent != nil && *input.ParticipationIntent != result.ParticipationIntent {
		result.ParticipationIntent, changed = *input.ParticipationIntent, true
	}
	return result, changed
}

func sameAdultOptional(current, next *string) bool {
	currentValue, nextValue := "", ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	if next != nil {
		nextValue = strings.TrimSpace(*next)
	}
	return currentValue == nextValue
}

func adultSummary(before, after *data.Adult) json.RawMessage {
	value := map[string]any{}
	if before != nil {
		value["before"] = map[string]any{"legal_given_name": before.LegalGivenName, "legal_family_name": before.LegalFamilyName, "participation_intent": before.ParticipationIntent}
	}
	if after != nil {
		value["after"] = map[string]any{"legal_given_name": after.LegalGivenName, "legal_family_name": after.LegalFamilyName, "participation_intent": after.ParticipationIntent}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return json.RawMessage(`{"error":"could not encode adult audit summary"}`)
	}
	return encoded
}
