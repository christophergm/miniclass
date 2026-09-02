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
	"github.com/chrismott/miniclass/internal/preference"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "interest_profile_surveys", YearScoped: true, Factory: createInterestProfileSurvey,
		ReadIDs: readInterestProfileSurveyIDs, FetchByID: fetchInterestProfileSurveyByID,
		UpdateByID: updateInterestProfileSurveyByID, DeleteByID: deleteInterestProfileSurveyByID,
		InsertWithForeignParent: insertInterestProfileSurveyWithForeignParent})
	Register(Entity{TableName: "interest_profile_survey_audience_students", YearScoped: true, Factory: createInterestProfileSurveyDefinitionStudent,
		ReadIDs: readInterestProfileSurveyDefinitionStudentIDs, FetchByID: fetchInterestProfileSurveyDefinitionStudentByID,
		UpdateByID: updateInterestProfileSurveyDefinitionStudentByID, DeleteByID: deleteInterestProfileSurveyDefinitionStudentByID,
		InsertWithForeignParent: insertInterestProfileSurveyDefinitionStudentWithForeignParent})
	Register(Entity{TableName: "interest_profile_survey_questions", YearScoped: true, Factory: createInterestProfileSurveyQuestion,
		ReadIDs: readInterestProfileSurveyQuestionIDs, FetchByID: fetchInterestProfileSurveyQuestionByID,
		UpdateByID: updateInterestProfileSurveyQuestionByID, DeleteByID: deleteInterestProfileSurveyQuestionByID,
		InsertWithForeignParent: insertInterestProfileSurveyQuestionWithForeignParent})
	Register(Entity{TableName: "interest_profile_survey_scale_options", YearScoped: true, Factory: createInterestProfileSurveyScaleOption,
		ReadIDs: readInterestProfileSurveyScaleOptionIDs, FetchByID: fetchInterestProfileSurveyScaleOptionByID,
		UpdateByID: updateInterestProfileSurveyScaleOptionByID, DeleteByID: deleteInterestProfileSurveyScaleOptionByID,
		InsertWithForeignParent: insertInterestProfileSurveyScaleOptionWithForeignParent})
	Register(Entity{TableName: "interest_profile_survey_audience_snapshots", YearScoped: true, Immutable: true, Factory: createInterestProfileSurveyAudienceSnapshot,
		ReadIDs: readInterestProfileSurveyAudienceSnapshotIDs, FetchByID: fetchInterestProfileSurveyAudienceSnapshotByID,
		UpdateByID: immutableUpdate, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertInterestProfileSurveyAudienceSnapshotWithForeignParent})
	Register(Entity{TableName: "interest_profile_survey_access_codes", YearScoped: true, Factory: createInterestProfileSurveyAccessCode,
		ReadIDs: readInterestProfileSurveyAccessCodeIDs, FetchByID: fetchInterestProfileSurveyAccessCodeByID,
		UpdateByID: updateInterestProfileSurveyAccessCodeByID, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertInterestProfileSurveyAccessCodeWithForeignParent})
}

type surveyFixture struct {
	factory *factories.Factory
	year    data.SchoolYear
	grade   data.GradeLevel
	student data.Student
	program data.Program
	area    data.InterestArea
	survey  preference.InterestProfileSurveyView
}

func createSurveyFixture(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (surveyFixture, error) {
	if harness == nil || harness.Database == nil {
		return surveyFixture{}, errors.New("survey fixture: database is nil")
	}
	factory := factories.New(harness.Database, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 survey fixture"})
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic survey year %s", organizationID))
	if err != nil {
		return surveyFixture{}, err
	}
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-survey", "Synthetic Survey Grade")
	if err != nil {
		return surveyFixture{}, err
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Survey Room")
	if err != nil {
		return surveyFixture{}, err
	}
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Survey", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	if err != nil {
		return surveyFixture{}, err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Survey Program")
	if err != nil {
		return surveyFixture{}, err
	}
	if _, err := factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID); err != nil {
		return surveyFixture{}, err
	}
	area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Survey Area")
	if err != nil {
		return surveyFixture{}, err
	}
	survey, err := preference.New(harness.Database).CreateInterestProfileSurvey(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 survey fixture"}, year.ID, programRow.ID, preference.InterestProfileSurveyInput{
		Name:      fmt.Sprintf("Synthetic Survey %s", organizationID),
		Audience:  preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceExplicitStudents, StudentIDs: []ids.XID{student.ID}},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: area.ID}},
	})
	if err != nil {
		return surveyFixture{}, err
	}
	return surveyFixture{factory: factory, year: year, grade: grade, student: student, program: programRow, area: area, survey: survey}, nil
}

func createOpenSurveyFixture(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (surveyFixture, error) {
	fixture, err := createSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return surveyFixture{}, err
	}
	closingAt := time.Now().UTC().Add(time.Hour)
	transition, err := preference.New(harness.Database).TransitionInterestProfileSurvey(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeSystem, Label: "layer 2 survey fixture"}, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	if err != nil {
		return surveyFixture{}, err
	}
	fixture.survey = transition.Survey
	return fixture, nil
}

func createInterestProfileSurvey(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createSurveyFixture(ctx, harness, organizationID)
	return fixture.survey.Survey.ID, err
}

func createInterestProfileSurveyDefinitionStudent(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	return fixture.survey.DefinitionStudents[0].ID, nil
}

func createInterestProfileSurveyQuestion(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	return fixture.survey.Questions[0].ID, nil
}

func createInterestProfileSurveyScaleOption(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	return fixture.survey.ScaleOptions[0].ID, nil
}

func createInterestProfileSurveyAudienceSnapshot(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createOpenSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	return fixture.survey.AudienceSnapshot[0].ID, nil
}

func createInterestProfileSurveyAccessCode(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createOpenSurveyFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	return fixture.survey.ActiveCodes[0].ID, nil
}

func readInterestProfileSurveyIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveysForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestProfileSurveyByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateInterestProfileSurveyForRegistry(ctx, id)
}

func deleteInterestProfileSurveyByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteInterestProfileSurveyForRegistry(ctx, id)
}

func readInterestProfileSurveyDefinitionStudentIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveyDefinitionStudentsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyDefinitionStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyDefinitionStudentForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestProfileSurveyDefinitionStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateInterestProfileSurveyDefinitionStudentForRegistry(ctx, id)
}

func deleteInterestProfileSurveyDefinitionStudentByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteInterestProfileSurveyDefinitionStudentForRegistry(ctx, id)
}

func readInterestProfileSurveyQuestionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveyQuestionsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyQuestionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyQuestionForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestProfileSurveyQuestionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateInterestProfileSurveyQuestionForRegistry(ctx, id)
}

func deleteInterestProfileSurveyQuestionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteInterestProfileSurveyQuestionForRegistry(ctx, id)
}

func readInterestProfileSurveyScaleOptionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveyScaleOptionsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyScaleOptionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyScaleOptionForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestProfileSurveyScaleOptionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.UpdateInterestProfileSurveyScaleOptionForRegistry(ctx, id)
}

func deleteInterestProfileSurveyScaleOptionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.DeleteInterestProfileSurveyScaleOptionForRegistry(ctx, id)
}

func readInterestProfileSurveyAudienceSnapshotIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveyAudienceSnapshotsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyAudienceSnapshotByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyAudienceSnapshotForRegistry(ctx, id)
	return row.ID != "", err
}

func readInterestProfileSurveyAccessCodeIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSurveyAccessCodesForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSurveyAccessCodeByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSurveyAccessCodeForRegistry(ctx, id)
	return row.ID != "", err
}

func updateInterestProfileSurveyAccessCodeByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.RevokeInterestProfileSurveyAccessCodeForRegistry(ctx, id)
}

func insertInterestProfileSurveyWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_surveys (organization_id, school_year_id, program_id, name) values ($1, $2, $3, $4)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, "Foreign Survey")
}

func insertInterestProfileSurveyDefinitionStudentWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_survey_audience_students (organization_id, school_year_id, program_id, survey_id, student_id) values ($1, $2, $3, $4, $5)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, fixture.student.ID)
}

func insertInterestProfileSurveyQuestionWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_survey_questions (organization_id, school_year_id, program_id, survey_id, interest_area_id, ordinal, label) values ($1, $2, $3, $4, $5, $6, $7)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, fixture.area.ID, 2, "Foreign question")
}

func insertInterestProfileSurveyScaleOptionWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_survey_scale_options (organization_id, school_year_id, program_id, survey_id, value, label, ordinal) values ($1, $2, $3, $4, $5, $6, $7)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, "foreign", "Foreign option", 4)
}

func insertInterestProfileSurveyAudienceSnapshotWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createOpenSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_survey_audience_snapshots (organization_id, school_year_id, program_id, survey_id, student_id) values ($1, $2, $3, $4, $5)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, fixture.student.ID)
}

func insertInterestProfileSurveyAccessCodeWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createOpenSurveyFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_survey_access_codes (organization_id, school_year_id, program_id, survey_id, student_id, code_hash) values ($1, $2, $3, $4, $5, $6)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.survey.Survey.ID, fixture.student.ID, "foreign-code-hash")
}
