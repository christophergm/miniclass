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

// AdultParticipationIntent is the year-scoped signal used for adult
// participation planning. It is deliberately not a person role.
type AdultParticipationIntent string

const (
	AdultParticipationLead        AdultParticipationIntent = "lead"
	AdultParticipationHelp        AdultParticipationIntent = "help"
	AdultParticipationUnavailable AdultParticipationIntent = "unavailable"
)

// Adult is a tenant-safe, school-year-scoped adult record.
type Adult struct {
	ID                  ids.XID
	OrganizationID      ids.XID
	SchoolYearID        ids.XID
	LegalGivenName      string
	LegalFamilyName     string
	PreferredGivenName  *string
	Email               *string
	Phone               *string
	ExternalIdentifier  *string
	ParticipationIntent AdultParticipationIntent
	DeletedAt           *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// CreateAdult inserts an active adult under the transaction tenant and year.
func (tx *Tx) CreateAdult(ctx context.Context, schoolYearID ids.XID, legalGivenName, legalFamilyName string, preferredGivenName, email, phone, externalIdentifier *string, intent AdultParticipationIntent) (Adult, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	if legalGivenName == "" || legalFamilyName == "" {
		return Adult{}, errors.New("create adult: legal names are required")
	}
	if !validAdultParticipationIntent(intent) {
		return Adult{}, fmt.Errorf("create adult: invalid participation intent %q", intent)
	}
	row, err := tx.queries.CreateAdult(ctx, db.CreateAdultParams{
		OrganizationID:      tx.organizationID,
		SchoolYearID:        schoolYearID,
		LegalGivenName:      legalGivenName,
		LegalFamilyName:     legalFamilyName,
		PreferredGivenName:  nullableAdultText(preferredGivenName),
		Email:               nullableAdultText(email),
		Phone:               nullableAdultText(phone),
		ExternalIdentifier:  nullableAdultText(externalIdentifier),
		ParticipationIntent: db.AdultParticipationIntent(intent),
	})
	if err != nil {
		return Adult{}, wrapAdultMutationError("create adult", err)
	}
	return adult(row)
}

// ListAdults lists adults for one year. Deleted rows are opt-in.
func (tx *Tx) ListAdults(ctx context.Context, schoolYearID ids.XID, includeDeleted bool) ([]Adult, error) {
	rows, err := tx.queries.ListAdults(ctx, db.ListAdultsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Column3: includeDeleted})
	if err != nil {
		return nil, fmt.Errorf("list adults: %w", err)
	}
	result := make([]Adult, 0, len(rows))
	for _, row := range rows {
		value, err := adult(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// GetAdultByID fetches one active adult in the transaction tenant and year.
func (tx *Tx) GetAdultByID(ctx context.Context, schoolYearID, id ids.XID) (Adult, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Adult{}, errors.New("get adult: ids are required")
	}
	row, err := tx.queries.GetAdultByID(ctx, db.GetAdultByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Adult{}, fmt.Errorf("get adult: %w", err)
	}
	return adult(row)
}

func (tx *Tx) GetAdultByIDIncludingDeleted(ctx context.Context, schoolYearID, id ids.XID) (Adult, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Adult{}, errors.New("get adult: ids are required")
	}
	row, err := tx.queries.GetAdultByIDIncludingDeleted(ctx, db.GetAdultByIDIncludingDeletedParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Adult{}, fmt.Errorf("get adult including deleted: %w", err)
	}
	return adult(row)
}

// UpdateAdult replaces the editable fields of one active adult.
func (tx *Tx) UpdateAdult(ctx context.Context, schoolYearID, id ids.XID, legalGivenName, legalFamilyName string, preferredGivenName, email, phone, externalIdentifier *string, intent AdultParticipationIntent) (Adult, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	if legalGivenName == "" || legalFamilyName == "" {
		return Adult{}, errors.New("update adult: legal names are required")
	}
	if !validAdultParticipationIntent(intent) {
		return Adult{}, fmt.Errorf("update adult: invalid participation intent %q", intent)
	}
	row, err := tx.queries.UpdateAdult(ctx, db.UpdateAdultParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID,
		LegalGivenName: legalGivenName, LegalFamilyName: legalFamilyName,
		PreferredGivenName: nullableAdultText(preferredGivenName), Email: nullableAdultText(email),
		Phone: nullableAdultText(phone), ExternalIdentifier: nullableAdultText(externalIdentifier),
		ParticipationIntent: db.AdultParticipationIntent(intent),
	})
	if err != nil {
		return Adult{}, wrapAdultMutationError("update adult", err)
	}
	return adult(row)
}

// SoftDeleteAdult hides one active adult while preserving its audit/history
// row and allowing a future import to reuse its external identifier.
func (tx *Tx) SoftDeleteAdult(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.SoftDeleteAdult(ctx, db.SoftDeleteAdultParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return false, wrapAdultMutationError("delete adult", err)
	}
	return rows == 1, nil
}

func (tx *Tx) RestoreAdult(ctx context.Context, schoolYearID, id ids.XID) (Adult, error) {
	row, err := tx.queries.RestoreAdult(ctx, db.RestoreAdultParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Adult{}, wrapAdultMutationError("restore adult", err)
	}
	return adult(row)
}

// ListAllActiveAdultsForRegistry is only used by the isolation registry. It
// keeps the generated query behind internal/data while allowing the generic
// registry to exercise every year-scoped adult row.
func (tx *Tx) ListAllActiveAdultsForRegistry(ctx context.Context) ([]Adult, error) {
	rows, err := tx.queries.ListAllActiveAdultsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list adults for registry: %w", err)
	}
	result := make([]Adult, 0, len(rows))
	for _, row := range rows {
		value, err := adult(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// FindAdultForRegistry returns one active adult and its year for a generic
// isolation operation.
func (tx *Tx) FindAdultForRegistry(ctx context.Context, id ids.XID) (Adult, ids.XID, error) {
	row, err := tx.queries.FindAdultForRegistry(ctx, db.FindAdultForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Adult{}, "", nil
		}
		return Adult{}, "", fmt.Errorf("find adult for registry: %w", err)
	}
	value, err := adult(row)
	return value, value.SchoolYearID, err
}

func validAdultParticipationIntent(intent AdultParticipationIntent) bool {
	switch intent {
	case AdultParticipationLead, AdultParticipationHelp, AdultParticipationUnavailable:
		return true
	default:
		return false
	}
}

func nullableAdultText(value *string) pgtype.Text {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(*value), Valid: true}
}

func adult(row db.Adult) (Adult, error) {
	createdAt, err := adultTime(row.CreatedAt, "created_at")
	if err != nil {
		return Adult{}, err
	}
	updatedAt, err := adultTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Adult{}, err
	}
	return Adult{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
		LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName,
		PreferredGivenName: nullableAdultString(row.PreferredGivenName), Email: nullableAdultString(row.Email),
		Phone: nullableAdultString(row.Phone), ExternalIdentifier: nullableAdultString(row.ExternalIdentifier),
		ParticipationIntent: AdultParticipationIntent(row.ParticipationIntent),
		DeletedAt:           nullableAdultTime(row.DeletedAt), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func nullableAdultString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableAdultTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func adultTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("adult row: %s is null", name)
	}
	return value.Time, nil
}

func wrapAdultMutationError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
