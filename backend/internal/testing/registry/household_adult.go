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
	Register(Entity{TableName: "household_adults", YearScoped: true, Factory: createHouseholdAdult, ReadIDs: readHouseholdAdultIDs,
		FetchByID: fetchHouseholdAdultByID, UpdateByID: updateHouseholdAdultByID, DeleteByID: deleteHouseholdAdultByID,
		InsertWithForeignParent: insertHouseholdAdultWithForeignParent})
}

func createHouseholdAdult(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create household adult fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 household adult factory"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, fmt.Sprintf("Synthetic adult membership year %s", organizationID))
	if err != nil {
		return "", err
	}
	adult, err := people.New(harness.Database).Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Membership Adult", ParticipationIntent: data.AdultParticipationHelp})
	if err != nil {
		return "", err
	}
	household, err := people.New(harness.Database).CreateHousehold(ctx, string(organizationID), year.ID, actor, people.HouseholdCreateInput{DisplayName: "Synthetic Adult Household"})
	if err != nil {
		return "", err
	}
	membership, err := people.New(harness.Database).AddAdultToHousehold(ctx, string(organizationID), year.ID, household.ID, adult.ID, actor)
	if err != nil {
		return "", err
	}
	return membership.ID, nil
}

func readHouseholdAdultIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllHouseholdAdultsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchHouseholdAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, _, err := tx.FindHouseholdAdultForRegistry(ctx, id)
	return row.ID != "", err
}
func updateHouseholdAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindHouseholdAdultForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.TouchHouseholdAdult(ctx, year, id)
}
func deleteHouseholdAdultByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, _, err := tx.FindHouseholdAdultForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.DeleteHouseholdAdult(ctx, id)
}
func insertHouseholdAdultWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert household adult fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into household_adults (organization_id, school_year_id, household_id, adult_id) values ($1, public.xid(), public.xid(), public.xid())`, string(foreignOrganizationID))
	return err
}
