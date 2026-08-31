package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "sessions", YearScoped: true, Factory: createSession,
		ReadIDs: readSessionIDs, FetchByID: fetchSessionByID, UpdateByID: updateSessionByID,
		DeleteByID: deleteSessionByID, InsertWithForeignParent: insertSessionWithForeignParent})
	Register(Entity{TableName: "meeting_dates", YearScoped: true, Factory: createMeetingDate,
		ReadIDs: readMeetingDateIDs, FetchByID: fetchMeetingDateByID, UpdateByID: updateMeetingDateByID,
		DeleteByID: deleteMeetingDateByID, InsertWithForeignParent: insertMeetingDateWithForeignParent})
}

func createSession(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create session fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 session factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic session year %s", organizationID))
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Session Program")
	if err != nil {
		return "", err
	}
	row, err := factory.CreateSession(ctx, year.ID, program.ID, "Synthetic Session", 1, []time.Time{time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func createMeetingDate(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create meeting date fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 meeting-date factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic meeting-date year %s", organizationID))
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Meeting Date Program")
	if err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, program.ID, "Synthetic Meeting Date Session", 1, []time.Time{time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	row, err := factory.CreateMeetingDate(ctx, year.ID, program.ID, session.ID, time.Date(2026, 9, 18, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readSessionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllSessionsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchSessionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSessionForRegistry(ctx, id)
	return row.ID != "", err
}

func updateSessionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSessionForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.UpdateSessionForRegistry(ctx, id, row.Name+" updated")
}

func deleteSessionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteSessionForRegistry(ctx, id)
}

func readMeetingDateIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllMeetingDatesForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchMeetingDateByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindMeetingDateForRegistry(ctx, id)
	return row.ID != "", err
}

func updateMeetingDateByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindMeetingDateForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.UpdateMeetingDateForRegistry(ctx, id, row.Date.AddDate(0, 0, 1))
}

func deleteMeetingDateByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteMeetingDateForRegistry(ctx, id)
}

func insertSessionWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert session fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign session fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Session Year")
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Session Program")
	if err != nil {
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
	_, err = tx.Exec(ctx, "insert into sessions (organization_id, school_year_id, program_id, name, ordinal) values ($1, $2, $3, $4, $5)", string(foreignOrganizationID), string(year.ID), string(program.ID), "Foreign Session", 1)
	return err
}

func insertMeetingDateWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert meeting date fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign meeting-date fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Meeting Date Year")
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Meeting Date Program")
	if err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, program.ID, "Synthetic Foreign Meeting Date Session", 1, []time.Time{time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
	if err != nil {
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
	_, err = tx.Exec(ctx, "insert into meeting_dates (organization_id, school_year_id, program_id, session_id, meeting_date) values ($1, $2, $3, $4, $5)", string(foreignOrganizationID), string(year.ID), string(program.ID), string(session.ID), "2026-10-09")
	return err
}
