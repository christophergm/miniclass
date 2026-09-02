package registry

import (
	"context"
	"errors"
	"fmt"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "adult_account_links", YearScoped: true, Factory: createAdultAccountLink,
		ReadIDs: readAdultAccountLinkIDs, FetchByID: fetchAdultAccountLinkByID,
		UpdateByID: updateAdultAccountLinkByID, DeleteByID: deleteAdultAccountLinkByID,
		InsertWithForeignParent: insertAdultAccountLinkWithForeignParent})
}

func createAdultAccountLink(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil || harness.Migrator == nil {
		return "", errors.New("create adult account link fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 adult account link factory"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic adult link year %s", organizationID))
	if err != nil {
		return "", err
	}
	adult, err := factory.CreateAdult(ctx, year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Linked Adult"})
	if err != nil {
		return "", err
	}
	var userID ids.XID
	err = harness.Migrator.QueryRow(ctx, `
		insert into users (provider_subject, email)
		values ($1, $2)
		returning id`, "adult-link-"+string(organizationID), "adult-link-"+string(organizationID)+"@example.test").Scan(&userID)
	if err != nil {
		return "", err
	}
	var link data.AdultAccountLink
	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		var err error
		link, err = tx.CreateAdultAccountLink(ctx, year.ID, adult.ID, userID)
		if err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionAdultAccountLink, ObjectType: "adult_account_link", ObjectID: &link.ID, SchoolYearID: &year.ID, ChangeSummary: []byte(`{"linked":true}`)})
	})
	return link.ID, err
}

func readAdultAccountLinkIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllAdultAccountLinksForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchAdultAccountLinkByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindAdultAccountLinkForRegistry(ctx, id)
	return row.ID != "", err
}

func updateAdultAccountLinkByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.TouchAdultAccountLinkForRegistry(ctx, id)
}

func deleteAdultAccountLinkByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindAdultAccountLinkForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.DeleteAdultAccountLink(ctx, row.SchoolYearID, id)
}

func insertAdultAccountLinkWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil || harness.Migrator == nil {
		return errors.New("insert adult account link fixture: harness is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign adult link fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Adult Link Year")
	if err != nil {
		return err
	}
	adult, err := foreignFactory.CreateAdult(ctx, year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Foreign Linked Adult"})
	if err != nil {
		return err
	}
	var userID ids.XID
	if err := harness.Migrator.QueryRow(ctx, `insert into users (provider_subject, email) values ($1, $2) returning id`, "foreign-adult-link-"+string(foreignOrganizationID), "foreign-adult-link-"+string(foreignOrganizationID)+"@example.test").Scan(&userID); err != nil {
		return err
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into adult_account_links (organization_id, school_year_id, adult_id, user_id) values ($1, $2, $3, $4)`, string(foreignOrganizationID), string(year.ID), string(adult.ID), string(userID))
	return err
}
