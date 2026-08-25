package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
)

func init() {
	Register(Entity{
		TableName: "adults", YearScoped: true,
		Factory: createAdult, ReadIDs: readAdultIDs, FetchByID: fetchAdultByID,
		UpdateByID: updateAdultByID, DeleteByID: deleteAdultByID,
		InsertWithForeignParent: insertAdultWithForeignParent,
	})
}

func createAdult(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create adult fixture: harness is nil")
	}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 school-year fixture"}, fmt.Sprintf("Synthetic year %s", organizationID))
	if err != nil {
		return "", err
	}
	row, err := people.New(harness.Database).Create(ctx, string(organizationID), year.ID, audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 adult factory"}, people.AdultCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: fmt.Sprintf("Adult %s", organizationID),
		ParticipationIntent: data.AdultParticipationHelp,
	})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func fetchAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	// The registry callback does not carry the year ID; use the row's opaque
	// parent through a small tenant-local lookup so the generated accessor stays
	// behind internal/data.
	return findAdultByID(ctx, tx, id)
}

func findAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	rows, err := tx.ListAllActiveAdultsForRegistry(ctx)
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.ID == id {
			return true, nil
		}
	}
	return false, nil
}

func readAdultIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllActiveAdultsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func updateAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindAdultForRegistry(ctx, id)
	if err != nil {
		return false, err
	}
	if row.ID == "" {
		return false, nil
	}
	updatedName := row.LegalGivenName + " updated"
	_, err = tx.UpdateAdult(ctx, yearID, id, updatedName, row.LegalFamilyName, row.PreferredGivenName, row.Email, row.Phone, row.ExternalIdentifier, row.ParticipationIntent)
	if err != nil {
		return false, err
	}
	return true, nil
}

func deleteAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, yearID, err := tx.FindAdultForRegistry(ctx, id)
	if err != nil {
		return false, err
	}
	if row.ID == "" {
		return false, nil
	}
	return tx.SoftDeleteAdult(ctx, yearID, id)
}

func insertAdultWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert adult fixture: app pool is nil")
	}
	var schoolYearID ids.XID
	if err := harness.Migrator.QueryRow(ctx, `
		select public.xid()`).Scan(&schoolYearID); err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "set local app.organization_id = '"+string(tenantID)+"'"); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		insert into adults (organization_id, school_year_id, legal_given_name, legal_family_name, participation_intent)
		values ($1, $2, 'Cross', 'Parent', 'help')`, string(foreignOrganizationID), string(schoolYearID))
	return err
}
