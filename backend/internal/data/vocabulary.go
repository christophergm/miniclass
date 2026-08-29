package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5/pgtype"
)

// GradeLevel is an organization's ordered, concrete grade vocabulary entry.
type GradeLevel struct {
	ID             ids.XID
	OrganizationID ids.XID
	Code           string
	Label          string
	Ordinal        int
	RetiredAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Homeroom is an organization's categorical homeroom vocabulary entry.
type Homeroom struct {
	ID                 ids.XID
	OrganizationID     ids.XID
	Name               string
	ExternalIdentifier *string
	RetiredAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// VocabularySettings contains the configurable label used for homerooms.
type VocabularySettings struct {
	OrganizationID ids.XID
	Organization   string
	HomeroomLabel  string
}

func (tx *Tx) CreateGradeLevel(ctx context.Context, code, label string, ordinal int) (GradeLevel, error) {
	code, label, err := vocabularyText(code, label)
	if err != nil {
		return GradeLevel{}, fmt.Errorf("create grade level: %w", err)
	}
	if ordinal < 1 {
		return GradeLevel{}, errors.New("create grade level: ordinal must be positive")
	}
	row, err := tx.queries.CreateGradeLevel(ctx, db.CreateGradeLevelParams{
		OrganizationID: tx.organizationID, Code: code, Label: label, Ordinal: int32(ordinal),
	})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("create grade level: %w", err)
	}
	return gradeLevel(row)
}

// ListGradeLevels returns picker entries unless includeRetired is requested.
func (tx *Tx) ListGradeLevels(ctx context.Context, includeRetired bool) ([]GradeLevel, error) {
	var rows []db.GradeLevel
	var err error
	if includeRetired {
		rows, err = tx.queries.ListAllGradeLevels(ctx)
	} else {
		rows, err = tx.queries.ListGradeLevels(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("list grade levels: %w", err)
	}
	result := make([]GradeLevel, 0, len(rows))
	for _, row := range rows {
		value, err := gradeLevel(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetGradeLevelByID(ctx context.Context, id ids.XID) (GradeLevel, error) {
	if strings.TrimSpace(string(id)) == "" {
		return GradeLevel{}, errors.New("get grade level: id is empty")
	}
	row, err := tx.queries.GetGradeLevelByID(ctx, id)
	if err != nil {
		return GradeLevel{}, fmt.Errorf("get grade level: %w", err)
	}
	return gradeLevel(row)
}

func (tx *Tx) UpdateGradeLevel(ctx context.Context, id ids.XID, code, label string) (GradeLevel, error) {
	code, label, err := vocabularyText(code, label)
	if err != nil {
		return GradeLevel{}, fmt.Errorf("update grade level: %w", err)
	}
	row, err := tx.queries.UpdateGradeLevel(ctx, db.UpdateGradeLevelParams{ID: id, Code: code, Label: label})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("update grade level: %w", err)
	}
	return gradeLevel(row)
}

func (tx *Tx) SetGradeLevelRetired(ctx context.Context, id ids.XID, retired bool) (GradeLevel, error) {
	row, err := tx.queries.SetGradeLevelRetired(ctx, db.SetGradeLevelRetiredParams{ID: id, Column2: retired})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("set grade level retirement: %w", err)
	}
	return gradeLevel(row)
}

// ReorderGradeLevels assigns ordinals in the supplied order. The caller must
// provide every grade ID in the tenant exactly once.
func (tx *Tx) ReorderGradeLevels(ctx context.Context, orderedIDs []ids.XID) ([]GradeLevel, error) {
	if len(orderedIDs) == 0 {
		return nil, errors.New("reorder grade levels: at least one grade level is required")
	}
	current, err := tx.ListGradeLevels(ctx, true)
	if err != nil {
		return nil, err
	}
	if len(current) != len(orderedIDs) {
		return nil, errors.New("reorder grade levels: all grade levels must be included")
	}
	known := make(map[ids.XID]struct{}, len(current))
	for _, row := range current {
		known[row.ID] = struct{}{}
	}
	seen := make(map[ids.XID]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("reorder grade levels: grade level %q is not in this organization", id)
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("reorder grade levels: grade level %q is repeated", id)
		}
		seen[id] = struct{}{}
	}
	if err := tx.queries.ShiftGradeLevelOrdinals(ctx, db.ShiftGradeLevelOrdinalsParams{
		OrganizationID: tx.organizationID, Ordinal: int32(len(current) + 1),
	}); err != nil {
		return nil, fmt.Errorf("reorder grade levels: clear existing ordinals: %w", err)
	}
	for index, id := range orderedIDs {
		if err := tx.queries.UpdateGradeLevelOrdinal(ctx, db.UpdateGradeLevelOrdinalParams{ID: id, Ordinal: int32(index + 1)}); err != nil {
			return nil, fmt.Errorf("reorder grade levels: assign ordinal: %w", err)
		}
	}
	return tx.ListGradeLevels(ctx, true)
}

func (tx *Tx) CreateHomeroom(ctx context.Context, name string, externalIdentifier *string) (Homeroom, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Homeroom{}, errors.New("create homeroom: name is empty")
	}
	row, err := tx.queries.CreateHomeroom(ctx, db.CreateHomeroomParams{OrganizationID: tx.organizationID, Name: name, ExternalIdentifier: nullableVocabularyText(externalIdentifier)})
	if err != nil {
		return Homeroom{}, fmt.Errorf("create homeroom: %w", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

// ListHomerooms returns picker entries unless includeRetired is requested.
func (tx *Tx) ListHomerooms(ctx context.Context, includeRetired bool) ([]Homeroom, error) {
	var result []Homeroom
	var err error
	if includeRetired {
		rows, queryErr := tx.queries.ListAllHomerooms(ctx)
		err = queryErr
		if err == nil {
			result = make([]Homeroom, 0, len(rows))
			for _, row := range rows {
				value, conversionErr := homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
				if conversionErr != nil {
					return nil, conversionErr
				}
				result = append(result, value)
			}
		}
	} else {
		rows, queryErr := tx.queries.ListHomerooms(ctx)
		err = queryErr
		if err == nil {
			result = make([]Homeroom, 0, len(rows))
			for _, row := range rows {
				value, conversionErr := homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
				if conversionErr != nil {
					return nil, conversionErr
				}
				result = append(result, value)
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list homerooms: %w", err)
	}
	return result, nil
}

func (tx *Tx) GetHomeroomByID(ctx context.Context, id ids.XID) (Homeroom, error) {
	if strings.TrimSpace(string(id)) == "" {
		return Homeroom{}, errors.New("get homeroom: id is empty")
	}
	row, err := tx.queries.GetHomeroomByID(ctx, id)
	if err != nil {
		return Homeroom{}, fmt.Errorf("get homeroom: %w", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateHomeroom(ctx context.Context, id ids.XID, name string, externalIdentifier *string) (Homeroom, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Homeroom{}, errors.New("update homeroom: name is empty")
	}
	row, err := tx.queries.UpdateHomeroom(ctx, db.UpdateHomeroomParams{ID: id, Name: name, ExternalIdentifier: nullableVocabularyText(externalIdentifier)})
	if err != nil {
		return Homeroom{}, fmt.Errorf("update homeroom: %w", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) SetHomeroomRetired(ctx context.Context, id ids.XID, retired bool) (Homeroom, error) {
	row, err := tx.queries.SetHomeroomRetired(ctx, db.SetHomeroomRetiredParams{ID: id, Column2: retired})
	if err != nil {
		return Homeroom{}, fmt.Errorf("set homeroom retirement: %w", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) GetVocabularySettings(ctx context.Context) (VocabularySettings, error) {
	row, err := tx.queries.GetOrganizationVocabularySettings(ctx, tx.organizationID)
	if err != nil {
		return VocabularySettings{}, fmt.Errorf("get vocabulary settings: %w", err)
	}
	return VocabularySettings{OrganizationID: row.ID, Organization: row.Name, HomeroomLabel: row.HomeroomLabel}, nil
}

func (tx *Tx) UpdateHomeroomLabel(ctx context.Context, label string) (VocabularySettings, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return VocabularySettings{}, errors.New("update homeroom label: label is empty")
	}
	row, err := tx.queries.UpdateOrganizationHomeroomLabel(ctx, db.UpdateOrganizationHomeroomLabelParams{
		ID: tx.organizationID, HomeroomLabel: label,
	})
	if err != nil {
		return VocabularySettings{}, fmt.Errorf("update homeroom label: %w", err)
	}
	return VocabularySettings{OrganizationID: row.ID, Organization: row.Name, HomeroomLabel: row.HomeroomLabel}, nil
}

func vocabularyText(first, second string) (string, string, error) {
	first, second = strings.TrimSpace(first), strings.TrimSpace(second)
	if first == "" {
		return "", "", errors.New("code is empty")
	}
	if second == "" {
		return "", "", errors.New("label is empty")
	}
	return first, second, nil
}

func gradeLevel(row db.GradeLevel) (GradeLevel, error) {
	createdAt, err := vocabularyTime(row.CreatedAt, "created_at")
	if err != nil {
		return GradeLevel{}, err
	}
	updatedAt, err := vocabularyTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return GradeLevel{}, err
	}
	return GradeLevel{ID: row.ID, OrganizationID: row.OrganizationID, Code: row.Code, Label: row.Label,
		Ordinal: int(row.Ordinal), RetiredAt: nullableTime(row.RetiredAt), CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func homeroomValues(id, organizationID ids.XID, name string, externalIdentifier pgtype.Text, retiredAt, createdAt, updatedAt pgtype.Timestamptz) (Homeroom, error) {
	created, err := vocabularyTime(createdAt, "created_at")
	if err != nil {
		return Homeroom{}, err
	}
	updated, err := vocabularyTime(updatedAt, "updated_at")
	if err != nil {
		return Homeroom{}, err
	}
	return Homeroom{ID: id, OrganizationID: organizationID, Name: name, ExternalIdentifier: nullableVocabularyString(externalIdentifier),
		RetiredAt: nullableTime(retiredAt), CreatedAt: created, UpdatedAt: updated}, nil
}

func nullableVocabularyText(value *string) pgtype.Text {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(*value), Valid: true}
}

func nullableVocabularyString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func vocabularyTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("vocabulary row: %s is null", name)
	}
	return value.Time, nil
}

func nullableTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
