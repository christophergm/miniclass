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
	"github.com/chrismott/miniclass/internal/vocabulary"
)

func init() {
	Register(Entity{TableName: "guardian_relationships", YearScoped: true, Factory: createGuardianRelationship, ReadIDs: readGuardianRelationshipIDs,
		FetchByID: fetchGuardianRelationshipByID, UpdateByID: updateGuardianRelationshipByID, DeleteByID: deleteGuardianRelationshipByID,
		InsertWithForeignParent: insertGuardianRelationshipWithForeignParent})
}

func createGuardianRelationship(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	if harness == nil {
		return "", errors.New("create guardian relationship fixture: harness is nil")
	}
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 guardian relationship factory"}
	year, err := schoolyear.New(harness.Database).Create(ctx, string(organizationID), actor, fmt.Sprintf("Synthetic relationship year %s", organizationID))
	if err != nil {
		return "", err
	}
	grade, err := vocabulary.New(harness.Database).CreateGrade(ctx, string(organizationID), actor, "synthetic-relationship", "Synthetic Grade")
	if err != nil {
		return "", err
	}
	homeroom, err := vocabulary.New(harness.Database).CreateHomeroom(ctx, string(organizationID), actor, "Synthetic Relationship Room")
	if err != nil {
		return "", err
	}
	student, err := people.New(harness.Database).CreateStudent(ctx, string(organizationID), year.ID, actor, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Relationship Student", GradeLevelID: grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return "", err
	}
	adult, err := people.New(harness.Database).Create(ctx, string(organizationID), year.ID, actor, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Relationship Adult", ParticipationIntent: data.AdultParticipationHelp})
	if err != nil {
		return "", err
	}
	relationship, err := people.New(harness.Database).CreateGuardianRelationship(ctx, string(organizationID), year.ID, actor, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: student.ID, RelationshipType: data.GuardianRelationshipParent})
	if err != nil {
		return "", err
	}
	return relationship.ID, nil
}

func readGuardianRelationshipIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllGuardianRelationshipsForRegistry(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, nil
}
func fetchGuardianRelationshipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, _, err := tx.FindGuardianRelationshipForRegistry(ctx, id)
	return row.ID != "", err
}
func updateGuardianRelationshipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindGuardianRelationshipForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	next := data.GuardianRelationshipOther
	if row.RelationshipType == next {
		next = data.GuardianRelationshipParent
	}
	_, err = tx.UpdateGuardianRelationship(ctx, year, id, next)
	return err == nil, err
}
func deleteGuardianRelationshipByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, year, err := tx.FindGuardianRelationshipForRegistry(ctx, id)
	if err != nil || row.ID == "" {
		return false, err
	}
	return tx.DeleteGuardianRelationship(ctx, year, id)
}
func insertGuardianRelationshipWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	if harness == nil || harness.App == nil {
		return errors.New("insert guardian relationship fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `insert into guardian_relationships (organization_id, school_year_id, adult_id, student_id, relationship_type) values ($1, public.xid(), public.xid(), public.xid(), 'parent')`, string(foreignOrganizationID))
	return err
}
