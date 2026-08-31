package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "session_non_participations", YearScoped: true, Factory: createSessionNonParticipation,
		ReadIDs: readSessionNonParticipationIDs, FetchByID: fetchSessionNonParticipationByID,
		UpdateByID: updateSessionNonParticipationByID, DeleteByID: deleteSessionNonParticipationByID,
		InsertWithForeignParent: insertSessionNonParticipationWithForeignParent})
}

func createSessionNonParticipation(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create session non-participation fixture: harness is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 non-participation factory"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic non-participation year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-non-participation", "Synthetic Non-Participation Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Non-Participation Room")
	if err != nil {
		return "", err
	}
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Non-Participant", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return "", err
	}
	program, err := factory.CreateProgram(ctx, year.ID, "Synthetic Non-Participation Program")
	if err != nil {
		return "", err
	}
	if _, err := factory.AddProgramMembership(ctx, year.ID, program.ID, student.ID); err != nil {
		return "", err
	}
	session, err := factory.CreateSession(ctx, year.ID, program.ID, "Synthetic Non-Participation Session", []time.Time{time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return "", err
	}
	row, err := factory.CreateSessionNonParticipation(ctx, year.ID, program.ID, session.ID, student.ID, "Synthetic fixture reason")
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func readSessionNonParticipationIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllSessionNonParticipationsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}

func fetchSessionNonParticipationByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSessionNonParticipationForRegistry(ctx, id)
	return row.ID != "", err
}

func updateSessionNonParticipationByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindSessionNonParticipationForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.UpdateSessionNonParticipationForRegistry(ctx, id, row.Reason+" updated")
}

func deleteSessionNonParticipationByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteSessionNonParticipationForRegistry(ctx, id)
}

func insertSessionNonParticipationWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert session non-participation fixture: app pool is nil")
	}
	foreignFactory := factories.New(harness.Database, string(foreignOrganizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "foreign non-participation fixture"})
	year, err := foreignFactory.CreateSchoolYear(ctx, "Synthetic Foreign Non-Participation Year")
	if err != nil {
		return err
	}
	grade, err := foreignFactory.CreateGradeLevel(ctx, year.ID, "synthetic-foreign-non-participation", "Synthetic Foreign Grade")
	if err != nil {
		return err
	}
	homeroom, err := foreignFactory.CreateHomeroom(ctx, year.ID, "Synthetic Foreign Room")
	if err != nil {
		return err
	}
	student, err := foreignFactory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Foreign Non-Participant", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return err
	}
	program, err := foreignFactory.CreateProgram(ctx, year.ID, "Synthetic Foreign Non-Participation Program")
	if err != nil {
		return err
	}
	if _, err := foreignFactory.AddProgramMembership(ctx, year.ID, program.ID, student.ID); err != nil {
		return err
	}
	session, err := foreignFactory.CreateSession(ctx, year.ID, program.ID, "Synthetic Foreign Non-Participation Session", []time.Time{time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)})
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
	_, err = tx.Exec(ctx, "insert into session_non_participations (organization_id, school_year_id, program_id, session_id, student_id, reason) values ($1, $2, $3, $4, $5, $6)", string(foreignOrganizationID), string(year.ID), string(program.ID), string(session.ID), string(student.ID), "Foreign reason")
	return err
}
