package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// GradeLevel is a school year's ordered, concrete grade vocabulary entry.
type GradeLevel struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	Code           string
	Label          string
	Ordinal        int
	RetiredAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Homeroom is a school year's categorical homeroom vocabulary entry.
type Homeroom struct {
	ID                 ids.XID
	OrganizationID     ids.XID
	SchoolYearID       ids.XID
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

func (tx *Tx) CreateGradeLevel(ctx context.Context, schoolYearID ids.XID, code, label string, ordinal int) (GradeLevel, error) {
	code, label, err := vocabularyText(code, label)
	if err != nil {
		return GradeLevel{}, wrapSchoolYearError("create grade level", err)
	}
	if ordinal < 1 {
		return GradeLevel{}, errors.New("create grade level: ordinal must be positive")
	}
	row, err := tx.queries.CreateGradeLevel(ctx, db.CreateGradeLevelParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Code: code, Label: label, Ordinal: int32(ordinal),
	})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("create grade level: %w", err)
	}
	return gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

// ListGradeLevels returns picker entries unless includeRetired is requested.
func (tx *Tx) ListGradeLevels(ctx context.Context, schoolYearID ids.XID, includeRetired bool) ([]GradeLevel, error) {
	var result []GradeLevel
	if includeRetired {
		rows, err := tx.queries.ListAllGradeLevels(ctx, db.ListAllGradeLevelsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
		if err != nil {
			return nil, fmt.Errorf("list grade levels: %w", err)
		}
		result = make([]GradeLevel, 0, len(rows))
		for _, row := range rows {
			value, err := gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
	} else {
		rows, err := tx.queries.ListGradeLevels(ctx, db.ListGradeLevelsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
		if err != nil {
			return nil, fmt.Errorf("list grade levels: %w", err)
		}
		result = make([]GradeLevel, 0, len(rows))
		for _, row := range rows {
			value, err := gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
			if err != nil {
				return nil, err
			}
			result = append(result, value)
		}
	}
	return result, nil
}

func (tx *Tx) GetGradeLevelByID(ctx context.Context, schoolYearID, id ids.XID) (GradeLevel, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return GradeLevel{}, errors.New("get grade level: ids are required")
	}
	row, err := tx.queries.GetGradeLevelByID(ctx, db.GetGradeLevelByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("get grade level: %w", err)
	}
	return gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateGradeLevel(ctx context.Context, schoolYearID, id ids.XID, code, label string) (GradeLevel, error) {
	code, label, err := vocabularyText(code, label)
	if err != nil {
		return GradeLevel{}, wrapSchoolYearError("update grade level", err)
	}
	row, err := tx.queries.UpdateGradeLevel(ctx, db.UpdateGradeLevelParams{ID: id, Code: code, Label: label, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GradeLevel{}, fmt.Errorf("update grade level: %w", err)
	}
	return gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) SetGradeLevelRetired(ctx context.Context, schoolYearID, id ids.XID, retired bool) (GradeLevel, error) {
	row, err := tx.queries.SetGradeLevelRetired(ctx, db.SetGradeLevelRetiredParams{ID: id, Column2: retired, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return GradeLevel{}, wrapSchoolYearError("set grade level retirement", err)
	}
	return gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

// ReorderGradeLevels assigns ordinals in the supplied order. The caller must
// provide every grade ID in the tenant exactly once.
func (tx *Tx) ReorderGradeLevels(ctx context.Context, schoolYearID ids.XID, orderedIDs []ids.XID) ([]GradeLevel, error) {
	if len(orderedIDs) == 0 {
		return nil, errors.New("reorder grade levels: at least one grade level is required")
	}
	current, err := tx.ListGradeLevels(ctx, schoolYearID, true)
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
			return nil, fmt.Errorf("reorder grade levels: grade level %q is not in this school year", id)
		}
		if _, ok := seen[id]; ok {
			return nil, fmt.Errorf("reorder grade levels: grade level %q is repeated", id)
		}
		seen[id] = struct{}{}
	}
	if err := tx.queries.ShiftGradeLevelOrdinals(ctx, db.ShiftGradeLevelOrdinalsParams{
		OrganizationID: tx.organizationID, Ordinal: int32(len(current) + 1), SchoolYearID: schoolYearID,
	}); err != nil {
		return nil, wrapSchoolYearError("reorder grade levels: clear existing ordinals", err)
	}
	for index, id := range orderedIDs {
		if err := tx.queries.UpdateGradeLevelOrdinal(ctx, db.UpdateGradeLevelOrdinalParams{ID: id, Ordinal: int32(index + 1), OrganizationID: tx.organizationID, SchoolYearID: schoolYearID}); err != nil {
			return nil, wrapSchoolYearError("reorder grade levels: assign ordinal", err)
		}
	}
	return tx.ListGradeLevels(ctx, schoolYearID, true)
}

func (tx *Tx) CreateHomeroom(ctx context.Context, schoolYearID ids.XID, name string, externalIdentifier *string) (Homeroom, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Homeroom{}, errors.New("create homeroom: name is empty")
	}
	row, err := tx.queries.CreateHomeroom(ctx, db.CreateHomeroomParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Name: name, ExternalIdentifier: nullableVocabularyText(externalIdentifier)})
	if err != nil {
		return Homeroom{}, wrapSchoolYearError("create homeroom", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

// ListHomerooms returns picker entries unless includeRetired is requested.
func (tx *Tx) ListHomerooms(ctx context.Context, schoolYearID ids.XID, includeRetired bool) ([]Homeroom, error) {
	var result []Homeroom
	var err error
	if includeRetired {
		rows, queryErr := tx.queries.ListAllHomerooms(ctx, db.ListAllHomeroomsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
		err = queryErr
		if err == nil {
			result = make([]Homeroom, 0, len(rows))
			for _, row := range rows {
				value, conversionErr := homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
				if conversionErr != nil {
					return nil, conversionErr
				}
				result = append(result, value)
			}
		}
	} else {
		rows, queryErr := tx.queries.ListHomerooms(ctx, db.ListHomeroomsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
		err = queryErr
		if err == nil {
			result = make([]Homeroom, 0, len(rows))
			for _, row := range rows {
				value, conversionErr := homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
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

func (tx *Tx) GetHomeroomByID(ctx context.Context, schoolYearID, id ids.XID) (Homeroom, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Homeroom{}, errors.New("get homeroom: ids are required")
	}
	row, err := tx.queries.GetHomeroomByID(ctx, db.GetHomeroomByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Homeroom{}, fmt.Errorf("get homeroom: %w", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) UpdateHomeroom(ctx context.Context, schoolYearID, id ids.XID, name string, externalIdentifier *string) (Homeroom, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Homeroom{}, errors.New("update homeroom: name is empty")
	}
	row, err := tx.queries.UpdateHomeroom(ctx, db.UpdateHomeroomParams{ID: id, Name: name, ExternalIdentifier: nullableVocabularyText(externalIdentifier), OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Homeroom{}, wrapSchoolYearError("update homeroom", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) SetHomeroomRetired(ctx context.Context, schoolYearID, id ids.XID, retired bool) (Homeroom, error) {
	row, err := tx.queries.SetHomeroomRetired(ctx, db.SetHomeroomRetiredParams{ID: id, Column2: retired, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Homeroom{}, wrapSchoolYearError("set homeroom retirement", err)
	}
	return homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func (tx *Tx) GetVocabularySettings(ctx context.Context) (VocabularySettings, error) {
	row, err := tx.queries.GetOrganizationVocabularySettings(ctx, tx.organizationID)
	if err != nil {
		return VocabularySettings{}, fmt.Errorf("get vocabulary settings: %w", err)
	}
	return VocabularySettings{OrganizationID: row.ID, Organization: row.Name, HomeroomLabel: row.HomeroomLabel}, nil
}

// FindGradeLevelForRegistry returns a grade and its year for a generic
// isolation operation. It intentionally scopes only by tenant so the registry
// can recover the year that must be supplied to the normal accessor.
func (tx *Tx) FindGradeLevelForRegistry(ctx context.Context, id ids.XID) (GradeLevel, ids.XID, error) {
	row, err := tx.queries.FindGradeLevelForRegistry(ctx, db.FindGradeLevelForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return GradeLevel{}, "", nil
		}
		return GradeLevel{}, "", fmt.Errorf("find grade level for registry: %w", err)
	}
	value, err := gradeLevel(db.GradeLevel{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
		Code: row.Code, Label: row.Label, Ordinal: row.Ordinal, RetiredAt: row.RetiredAt,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	})
	return value, value.SchoolYearID, err
}

// FindHomeroomForRegistry returns a homeroom and its year for a generic
// isolation operation.
func (tx *Tx) FindHomeroomForRegistry(ctx context.Context, id ids.XID) (Homeroom, ids.XID, error) {
	row, err := tx.queries.FindHomeroomForRegistry(ctx, db.FindHomeroomForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Homeroom{}, "", nil
		}
		return Homeroom{}, "", fmt.Errorf("find homeroom for registry: %w", err)
	}
	value, err := homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
	return value, value.SchoolYearID, err
}

// ListAllGradeLevelsForRegistry returns every active tenant-local grade for
// the generic isolation registry, across all school years.
func (tx *Tx) ListAllGradeLevelsForRegistry(ctx context.Context) ([]GradeLevel, error) {
	rows, err := tx.queries.ListAllGradeLevelsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list grade levels for registry: %w", err)
	}
	result := make([]GradeLevel, 0, len(rows))
	for _, row := range rows {
		value, err := gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// ListAllHomeroomsForRegistry returns every active tenant-local homeroom for
// the generic isolation registry, across all school years.
func (tx *Tx) ListAllHomeroomsForRegistry(ctx context.Context) ([]Homeroom, error) {
	rows, err := tx.queries.ListAllHomeroomsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list homerooms for registry: %w", err)
	}
	result := make([]Homeroom, 0, len(rows))
	for _, row := range rows {
		value, err := homeroomValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Name, row.ExternalIdentifier, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
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
	return gradeLevelValues(row.ID, row.OrganizationID, row.SchoolYearID, row.Code, row.Label, row.Ordinal, row.RetiredAt, row.CreatedAt, row.UpdatedAt)
}

func gradeLevelValues(id, organizationID, schoolYearID ids.XID, code, label string, ordinal int32, retiredAt, createdAt, updatedAt pgtype.Timestamptz) (GradeLevel, error) {
	created, err := vocabularyTime(createdAt, "created_at")
	if err != nil {
		return GradeLevel{}, err
	}
	updated, err := vocabularyTime(updatedAt, "updated_at")
	if err != nil {
		return GradeLevel{}, err
	}
	return GradeLevel{ID: id, OrganizationID: organizationID, SchoolYearID: schoolYearID, Code: code, Label: label,
		Ordinal: int(ordinal), RetiredAt: nullableTime(retiredAt), CreatedAt: created, UpdatedAt: updated}, nil
}

func homeroomValues(id, organizationID, schoolYearID ids.XID, name string, externalIdentifier pgtype.Text, retiredAt, createdAt, updatedAt pgtype.Timestamptz) (Homeroom, error) {
	created, err := vocabularyTime(createdAt, "created_at")
	if err != nil {
		return Homeroom{}, err
	}
	updated, err := vocabularyTime(updatedAt, "updated_at")
	if err != nil {
		return Homeroom{}, err
	}
	return Homeroom{ID: id, OrganizationID: organizationID, SchoolYearID: schoolYearID, Name: name, ExternalIdentifier: nullableVocabularyString(externalIdentifier),
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
