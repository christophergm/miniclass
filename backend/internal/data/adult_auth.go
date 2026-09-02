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
)

// AdultAccountLink is the explicit, tenant-scoped association between an
// administrative account and the adult record used for guardian mode.
type AdultAccountLink struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	AdultID        ids.XID
	UserID         ids.XID
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// GuardianScope is resolved from live relationships for every authenticated
// request. The session only retains the adult and tenant identifiers.
type GuardianScope struct {
	Adult      Adult
	StudentIDs []ids.XID
}

func (scope GuardianScope) AdultEmail() string {
	if scope.Adult.Email == nil {
		return ""
	}
	return *scope.Adult.Email
}

func (tx *Tx) CreateAdultAccountLink(ctx context.Context, schoolYearID, adultID, userID ids.XID) (AdultAccountLink, error) {
	if tx == nil || tx.queries == nil {
		return AdultAccountLink{}, errors.New("create adult account link: transaction is nil")
	}
	row, err := tx.queries.CreateAdultAccountLink(ctx, db.CreateAdultAccountLinkParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		AdultID:        adultID,
		UserID:         userID,
	})
	if err != nil {
		return AdultAccountLink{}, err
	}
	return adultAccountLink(row), nil
}

func (tx *Tx) GetAdultAccountLink(ctx context.Context, schoolYearID, userID ids.XID) (AdultAccountLink, error) {
	if tx == nil || tx.queries == nil {
		return AdultAccountLink{}, errors.New("get adult account link: transaction is nil")
	}
	row, err := tx.queries.GetAdultAccountLink(ctx, db.GetAdultAccountLinkParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		UserID:         userID,
	})
	if err != nil {
		return AdultAccountLink{}, err
	}
	return adultAccountLink(row), nil
}

func (tx *Tx) GetAdultAccountLinkByAdult(ctx context.Context, schoolYearID, adultID ids.XID) (AdultAccountLink, error) {
	if tx == nil || tx.queries == nil {
		return AdultAccountLink{}, errors.New("get adult account link: transaction is nil")
	}
	row, err := tx.queries.GetAdultAccountLinkByAdult(ctx, db.GetAdultAccountLinkByAdultParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		AdultID:        adultID,
	})
	if err != nil {
		return AdultAccountLink{}, err
	}
	return adultAccountLink(row), nil
}

func (tx *Tx) ListAdultAccountLinks(ctx context.Context, schoolYearID ids.XID) ([]AdultAccountLink, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("list adult account links: transaction is nil")
	}
	rows, err := tx.queries.ListAdultAccountLinks(ctx, db.ListAdultAccountLinksParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]AdultAccountLink, 0, len(rows))
	for _, row := range rows {
		result = append(result, adultAccountLink(row))
	}
	return result, nil
}

func (tx *Tx) DeleteAdultAccountLink(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	if tx == nil || tx.queries == nil {
		return false, errors.New("delete adult account link: transaction is nil")
	}
	rows, err := tx.queries.DeleteAdultAccountLink(ctx, db.DeleteAdultAccountLinkParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID,
	})
	return rows == 1, err
}

// ListAllAdultAccountLinksForRegistry is used by the Layer 2 isolation
// registry and intentionally returns only tenant-local rows.
func (tx *Tx) ListAllAdultAccountLinksForRegistry(ctx context.Context) ([]AdultAccountLink, error) {
	rows, err := tx.queries.ListAllAdultAccountLinksForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, err
	}
	result := make([]AdultAccountLink, 0, len(rows))
	for _, row := range rows {
		result = append(result, adultAccountLink(row))
	}
	return result, nil
}

func (tx *Tx) FindAdultAccountLinkForRegistry(ctx context.Context, id ids.XID) (AdultAccountLink, error) {
	row, err := tx.queries.FindAdultAccountLinkForRegistry(ctx, db.FindAdultAccountLinkForRegistryParams{
		ID: id, OrganizationID: tx.organizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return AdultAccountLink{}, nil
		}
		return AdultAccountLink{}, err
	}
	return adultAccountLink(row), nil
}

func (tx *Tx) TouchAdultAccountLinkForRegistry(ctx context.Context, id ids.XID) (bool, error) {
	rows, err := tx.queries.TouchAdultAccountLinkForRegistry(ctx, db.TouchAdultAccountLinkForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	return rows == 1, err
}

func (tx *Tx) ResolveGuardianScope(ctx context.Context, schoolYearID, adultID ids.XID) (GuardianScope, error) {
	if tx == nil || tx.queries == nil {
		return GuardianScope{}, errors.New("resolve guardian scope: transaction is nil")
	}
	adult, err := tx.GetAdultByID(ctx, schoolYearID, adultID)
	if err != nil {
		return GuardianScope{}, err
	}
	studentIDs, err := tx.queries.ListGuardianStudentIDs(ctx, db.ListGuardianStudentIDsParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		AdultID:        adultID,
	})
	if err != nil {
		return GuardianScope{}, fmt.Errorf("resolve guardian scope: %w", err)
	}
	return GuardianScope{Adult: adult, StudentIDs: studentIDs}, nil
}

func (tx *Tx) FindActiveAdultsByEmail(ctx context.Context, schoolYearID ids.XID, email string) ([]Adult, error) {
	if tx == nil || tx.queries == nil {
		return nil, errors.New("find adults by email: transaction is nil")
	}
	rows, err := tx.queries.FindActiveAdultsByEmail(ctx, db.FindActiveAdultsByEmailParams{
		OrganizationID: tx.organizationID,
		SchoolYearID:   schoolYearID,
		Lower:          strings.TrimSpace(email),
	})
	if err != nil {
		return nil, err
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

func adultAccountLink(row db.AdultAccountLink) AdultAccountLink {
	return AdultAccountLink{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
		AdultID: row.AdultID, UserID: row.UserID,
		CreatedAt: row.CreatedAt.Time, UpdatedAt: row.UpdatedAt.Time,
	}
}
