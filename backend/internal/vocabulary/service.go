// Package vocabulary owns each school year's grade and homeroom definitions.
package vocabulary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

var (
	ErrNoChanges = errors.New("vocabulary update has no changes")
	ErrInvalid   = errors.New("invalid vocabulary input")
)

type GradeLevelUpdate struct {
	Code    *string
	Label   *string
	Retired *bool
}

type HomeroomUpdate struct {
	Name               *string
	ExternalIdentifier **string
	Retired            *bool
}

type Snapshot struct {
	SchoolYearID ids.XID
	Settings     data.VocabularySettings
	Grades       []data.GradeLevel
	Homerooms    []data.Homeroom
}

type Service struct {
	database *data.DB
}

func New(database *data.DB) *Service {
	return &Service{database: database}
}

func (s *Service) List(ctx context.Context, organizationID string, schoolYearID ids.XID, includeRetired bool) (Snapshot, error) {
	if s == nil || s.database == nil {
		return Snapshot{}, errors.New("list vocabulary: data service is nil")
	}
	var result Snapshot
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetSchoolYearByID(ctx, schoolYearID); err != nil {
			return err
		}
		result.SchoolYearID = schoolYearID
		var err error
		result.Settings, err = tx.GetVocabularySettings(ctx)
		if err != nil {
			return err
		}
		result.Grades, err = tx.ListGradeLevels(ctx, schoolYearID, includeRetired)
		if err != nil {
			return err
		}
		result.Homerooms, err = tx.ListHomerooms(ctx, schoolYearID, includeRetired)
		return err
	})
	if err != nil {
		return Snapshot{}, fmt.Errorf("list vocabulary: %w", err)
	}
	return result, nil
}

func (s *Service) GetGrade(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.GradeLevel, error) {
	if s == nil || s.database == nil {
		return data.GradeLevel{}, errors.New("get grade level: data service is nil")
	}
	var result data.GradeLevel
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetGradeLevelByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.GradeLevel{}, fmt.Errorf("get grade level: %w", err)
	}
	return result, nil
}

func (s *Service) CreateGrade(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, code, label string) (data.GradeLevel, error) {
	if s == nil || s.database == nil {
		return data.GradeLevel{}, errors.New("create grade level: data service is nil")
	}
	var result data.GradeLevel
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		levels, err := tx.ListGradeLevels(ctx, schoolYearID, true)
		if err != nil {
			return err
		}
		ordinal := 1
		for _, level := range levels {
			if level.Ordinal >= ordinal {
				ordinal = level.Ordinal + 1
			}
		}
		result, err = tx.CreateGradeLevel(ctx, schoolYearID, code, label, ordinal)
		if err != nil {
			return err
		}
		id := result.ID
		return recordChange(ctx, tx, schoolYearID, &id, "grade_level", map[string]any{
			"code": result.Code, "label": result.Label, "ordinal": result.Ordinal,
		})
	})
	if err != nil {
		return data.GradeLevel{}, fmt.Errorf("create grade level: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateGrade(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input GradeLevelUpdate) (data.GradeLevel, error) {
	if s == nil || s.database == nil {
		return data.GradeLevel{}, errors.New("update grade level: data service is nil")
	}
	var result data.GradeLevel
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetGradeLevelByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		changed := false
		before := map[string]any{"code": current.Code, "label": current.Label, "retired": current.RetiredAt != nil}
		if input.Code != nil || input.Label != nil {
			code, label := current.Code, current.Label
			if input.Code != nil {
				code = *input.Code
			}
			if input.Label != nil {
				label = *input.Label
			}
			if strings.TrimSpace(code) != current.Code || strings.TrimSpace(label) != current.Label {
				result, err = tx.UpdateGradeLevel(ctx, schoolYearID, id, code, label)
				if err != nil {
					return err
				}
				changed = true
			}
		}
		if input.Retired != nil && *input.Retired != (current.RetiredAt != nil) {
			result, err = tx.SetGradeLevelRetired(ctx, schoolYearID, id, *input.Retired)
			if err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			return ErrNoChanges
		}
		if result.ID == "" {
			result = current
		}
		after := map[string]any{"code": result.Code, "label": result.Label, "retired": result.RetiredAt != nil}
		return recordChange(ctx, tx, schoolYearID, &id, "grade_level", map[string]any{"before": before, "after": after})
	})
	if err != nil {
		return data.GradeLevel{}, fmt.Errorf("update grade level: %w", err)
	}
	return result, nil
}

func (s *Service) ReorderGrades(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, orderedIDs []ids.XID) ([]data.GradeLevel, error) {
	if s == nil || s.database == nil {
		return nil, errors.New("reorder grade levels: data service is nil")
	}
	var result []data.GradeLevel
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.ReorderGradeLevels(ctx, schoolYearID, orderedIDs)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalid, err)
		}
		return recordChange(ctx, tx, schoolYearID, nil, "grade_level", map[string]any{"reordered_ids": orderedIDs})
	})
	if err != nil {
		return nil, fmt.Errorf("reorder grade levels: %w", err)
	}
	return result, nil
}

func (s *Service) GetHomeroom(ctx context.Context, organizationID string, schoolYearID, id ids.XID) (data.Homeroom, error) {
	if s == nil || s.database == nil {
		return data.Homeroom{}, errors.New("get homeroom: data service is nil")
	}
	var result data.Homeroom
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.GetHomeroomByID(ctx, schoolYearID, id)
		return err
	})
	if err != nil {
		return data.Homeroom{}, fmt.Errorf("get homeroom: %w", err)
	}
	return result, nil
}

func (s *Service) CreateHomeroom(ctx context.Context, organizationID string, schoolYearID ids.XID, actor audit.Actor, name string, externalIdentifier *string) (data.Homeroom, error) {
	if s == nil || s.database == nil {
		return data.Homeroom{}, errors.New("create homeroom: data service is nil")
	}
	var result data.Homeroom
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.CreateHomeroom(ctx, schoolYearID, name, externalIdentifier)
		if err != nil {
			return err
		}
		id := result.ID
		return recordChange(ctx, tx, schoolYearID, &id, "homeroom", map[string]any{"name": result.Name, "external_identifier": result.ExternalIdentifier})
	})
	if err != nil {
		return data.Homeroom{}, fmt.Errorf("create homeroom: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateHomeroom(ctx context.Context, organizationID string, schoolYearID, id ids.XID, actor audit.Actor, input HomeroomUpdate) (data.Homeroom, error) {
	if s == nil || s.database == nil {
		return data.Homeroom{}, errors.New("update homeroom: data service is nil")
	}
	var result data.Homeroom
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetHomeroomByID(ctx, schoolYearID, id)
		if err != nil {
			return err
		}
		changed := false
		before := map[string]any{"name": current.Name, "external_identifier": current.ExternalIdentifier, "retired": current.RetiredAt != nil}
		if input.ExternalIdentifier != nil {
			value := *input.ExternalIdentifier
			if !sameOptionalString(current.ExternalIdentifier, value) {
				result, err = tx.UpdateHomeroom(ctx, schoolYearID, id, current.Name, value)
				if err != nil {
					return err
				}
				changed = true
			}
		}
		if input.Name != nil && strings.TrimSpace(*input.Name) != current.Name {
			externalIdentifier := current.ExternalIdentifier
			if input.ExternalIdentifier != nil {
				externalIdentifier = *input.ExternalIdentifier
			}
			result, err = tx.UpdateHomeroom(ctx, schoolYearID, id, *input.Name, externalIdentifier)
			if err != nil {
				return err
			}
			changed = true
		}
		if input.Retired != nil && *input.Retired != (current.RetiredAt != nil) {
			result, err = tx.SetHomeroomRetired(ctx, schoolYearID, id, *input.Retired)
			if err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			return ErrNoChanges
		}
		if result.ID == "" {
			result = current
		}
		return recordChange(ctx, tx, schoolYearID, &id, "homeroom", map[string]any{
			"before": before, "after": map[string]any{"name": result.Name, "external_identifier": result.ExternalIdentifier, "retired": result.RetiredAt != nil},
		})
	})
	if err != nil {
		return data.Homeroom{}, fmt.Errorf("update homeroom: %w", err)
	}
	return result, nil
}

func sameOptionalString(current, next *string) bool {
	currentValue, nextValue := "", ""
	if current != nil {
		currentValue = strings.TrimSpace(*current)
	}
	if next != nil {
		nextValue = strings.TrimSpace(*next)
	}
	return currentValue == nextValue
}

func (s *Service) UpdateHomeroomLabel(ctx context.Context, organizationID string, actor audit.Actor, label string) (data.VocabularySettings, error) {
	if s == nil || s.database == nil {
		return data.VocabularySettings{}, errors.New("update homeroom label: data service is nil")
	}
	var result data.VocabularySettings
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetVocabularySettings(ctx)
		if err != nil {
			return err
		}
		label = strings.TrimSpace(label)
		if label == current.HomeroomLabel {
			return ErrNoChanges
		}
		result, err = tx.UpdateHomeroomLabel(ctx, label)
		if err != nil {
			return err
		}
		id := result.OrganizationID
		return recordChange(ctx, tx, "", &id, "organization", map[string]any{
			"before": map[string]string{"homeroom_label": current.HomeroomLabel},
			"after":  map[string]string{"homeroom_label": result.HomeroomLabel},
		})
	})
	if err != nil {
		return data.VocabularySettings{}, fmt.Errorf("update homeroom label: %w", err)
	}
	return result, nil
}

func recordChange(ctx context.Context, tx *data.Tx, schoolYearID ids.XID, objectID *ids.XID, objectType string, summaryValue any) error {
	encoded, err := json.Marshal(summaryValue)
	if err != nil {
		return fmt.Errorf("record vocabulary change: encode summary: %w", err)
	}
	var year *ids.XID
	if schoolYearID != "" {
		year = &schoolYearID
	}
	return tx.Record(ctx, audit.Entry{
		Action: audit.ActionVocabularyChange, ObjectType: objectType, ObjectID: objectID, SchoolYearID: year, ChangeSummary: encoded,
	})
}
