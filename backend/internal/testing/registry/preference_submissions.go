package registry

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
)

func init() {
	Register(Entity{TableName: "interest_profile_submissions", YearScoped: true, Immutable: true,
		Factory: createInterestProfileSubmission, ReadIDs: readInterestProfileSubmissionIDs,
		FetchByID: fetchInterestProfileSubmissionByID, UpdateByID: immutableUpdate, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertInterestProfileSubmissionWithForeignParent})
	Register(Entity{TableName: "interest_profile_responses", YearScoped: true, Immutable: true,
		Factory: createInterestProfileResponse, ReadIDs: readInterestProfileResponseIDs,
		FetchByID: fetchInterestProfileResponseByID, UpdateByID: immutableUpdate, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertInterestProfileResponseWithForeignParent})
	Register(Entity{TableName: "ranked_choice_submissions", YearScoped: true, Immutable: true,
		Factory: createRankedChoiceSubmission, ReadIDs: readRankedChoiceSubmissionIDs,
		FetchByID: fetchRankedChoiceSubmissionByID, UpdateByID: immutableUpdate, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertRankedChoiceSubmissionWithForeignParent})
	Register(Entity{TableName: "ranked_choice_responses", YearScoped: true, Immutable: true,
		Factory: createRankedChoiceResponse, ReadIDs: readRankedChoiceResponseIDs,
		FetchByID: fetchRankedChoiceResponseByID, UpdateByID: immutableUpdate, DeleteByID: immutableDelete,
		InsertWithForeignParent: insertRankedChoiceResponseWithForeignParent})
	Register(Entity{TableName: "ranked_choice_access_codes", YearScoped: true,
		Factory: createRankedChoiceAccessCode, ReadIDs: readRankedChoiceAccessCodeIDs,
		FetchByID: fetchRankedChoiceAccessCodeByID, UpdateByID: revokeRankedChoiceAccessCode, DeleteByID: revokeRankedChoiceAccessCode,
		InsertWithForeignParent: insertRankedChoiceAccessCodeWithForeignParent})
}

func createInterestProfileSubmission(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createInterestFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	submission, err := fixture.factory.SubmitInterestProfile(ctx, preference.InterestProfileSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelStudentCode,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}},
	})
	return submission.ID, err
}

func createInterestProfileResponse(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createInterestFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	submission, err := fixture.factory.SubmitInterestProfile(ctx, preference.InterestProfileSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelStudentCode,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}},
	})
	if err != nil {
		return "", err
	}
	var result []data.InterestProfileResponse
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		result, err = tx.ListInterestProfileResponses(ctx, fixture.year.ID, fixture.program.ID, submission.ID)
		return err
	})
	if err != nil {
		return "", err
	}
	if len(result) == 0 {
		return "", errors.New("interest profile response fixture: response not found")
	}
	return result[0].ID, nil
}

func createRankedChoiceSubmission(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createRankedFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	submission, err := fixture.factory.SubmitRankedChoices(ctx, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: fixture.session.ID, StudentID: fixture.student.ID,
		Code: fixture.accessCode, Channel: data.PreferenceChannelStudentCode,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: fixture.offering.ID, Answer: data.RankedChoiceInterested}},
	})
	return submission.ID, err
}

func createRankedChoiceResponse(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createRankedFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	_, err = fixture.factory.SubmitRankedChoices(ctx, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: fixture.session.ID, StudentID: fixture.student.ID,
		Code: fixture.accessCode, Channel: data.PreferenceChannelStudentCode,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: fixture.offering.ID, Answer: data.RankedChoiceInterested}},
	})
	if err != nil {
		return "", err
	}
	var result []data.RankedChoiceResponse
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		submission, err := tx.GetLatestRankedChoiceSubmission(ctx, fixture.year.ID, fixture.program.ID, fixture.session.ID, fixture.student.ID)
		if err != nil {
			return err
		}
		result, err = tx.ListRankedChoiceResponses(ctx, fixture.year.ID, fixture.program.ID, fixture.session.ID, submission.ID)
		return err
	})
	if err != nil {
		return "", err
	}
	if len(result) == 0 {
		return "", errors.New("ranked choice response fixture: response not found")
	}
	return result[0].ID, nil
}

type interestFixture struct {
	factory *factories.Factory
	year    data.SchoolYear
	grade   data.GradeLevel
	program data.Program
	student data.Student
	area    data.InterestArea
}

func createInterestFixture(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (interestFixture, error) {
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: "layer 2 student-code fixture"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic preference year %s", organizationID))
	if err != nil {
		return interestFixture{}, err
	}
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-preference", "Synthetic Preference Grade")
	if err != nil {
		return interestFixture{}, err
	}
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Preference Room")
	if err != nil {
		return interestFixture{}, err
	}
	student, err := factory.CreateStudent(ctx, year.ID, structStudent(grade.ID, homeroom.ID))
	if err != nil {
		return interestFixture{}, err
	}
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Preference Program")
	if err != nil {
		return interestFixture{}, err
	}
	if _, err := factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID); err != nil {
		return interestFixture{}, err
	}
	area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Preference Area")
	if err != nil {
		return interestFixture{}, err
	}
	return interestFixture{factory: factory, year: year, grade: grade, program: programRow, student: student, area: area}, nil
}

type rankedFixture struct {
	factory    *factories.Factory
	year       data.SchoolYear
	program    data.Program
	student    data.Student
	grade      data.GradeLevel
	session    data.Session
	offering   data.Offering
	accessCode string
}

func createRankedFixture(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (rankedFixture, error) {
	interest, err := createInterestFixture(ctx, harness, organizationID)
	if err != nil {
		return rankedFixture{}, err
	}
	session, err := interest.factory.CreateSession(ctx, interest.year.ID, interest.program.ID, "Synthetic Preference Session", []time.Time{time.Date(2026, 10, 23, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		return rankedFixture{}, err
	}
	offering, err := interest.factory.CreateOffering(ctx, interest.year.ID, interest.program.ID, session.ID, "Synthetic Preference Offering", "Synthetic description", nil, 10, interest.grade.ID, interest.grade.ID, "", "", "", nil)
	if err != nil {
		return rankedFixture{}, err
	}
	if _, err := interest.factory.ConfigureRankedChoice(ctx, interest.year.ID, interest.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour)); err != nil {
		return rankedFixture{}, err
	}
	if _, err := interest.factory.TransitionSession(ctx, interest.year.ID, interest.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil); err != nil {
		return rankedFixture{}, err
	}
	opened, err := interest.factory.TransitionSession(ctx, interest.year.ID, interest.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	if err != nil {
		return rankedFixture{}, err
	}
	if len(opened.AccessCodes) != 1 {
		return rankedFixture{}, fmt.Errorf("ranked choice fixture: expected one access code, got %d", len(opened.AccessCodes))
	}
	return rankedFixture{factory: interest.factory, year: interest.year, grade: interest.grade, program: interest.program, student: interest.student, session: opened.Session, offering: offering, accessCode: opened.AccessCodes[0].Code}, nil
}

func readInterestProfileSubmissionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileSubmissionsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileSubmissionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileSubmissionForRegistry(ctx, id)
	return row.ID != "", err
}

func readInterestProfileResponseIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllInterestProfileResponsesForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchInterestProfileResponseByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindInterestProfileResponseForRegistry(ctx, id)
	return row.ID != "", err
}

func readRankedChoiceSubmissionIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllRankedChoiceSubmissionsForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchRankedChoiceSubmissionByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindRankedChoiceSubmissionForRegistry(ctx, id)
	return row.ID != "", err
}

func readRankedChoiceResponseIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllRankedChoiceResponsesForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchRankedChoiceResponseByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindRankedChoiceResponseForRegistry(ctx, id)
	return row.ID != "", err
}

func createRankedChoiceAccessCode(ctx context.Context, harness *testharness.Harness, organizationID ids.XID) (ids.XID, error) {
	fixture, err := createRankedFixture(ctx, harness, organizationID)
	if err != nil {
		return "", err
	}
	var rows []data.RankedChoiceAccessCode
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		rows, err = tx.ListActiveRankedChoiceAccessCodes(ctx, fixture.year.ID, fixture.program.ID, fixture.session.ID)
		return err
	})
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", errors.New("ranked choice access code fixture: code not found")
	}
	return rows[0].ID, nil
}

func readRankedChoiceAccessCodeIDs(ctx context.Context, tx *data.Tx) ([]ids.XID, error) {
	rows, err := tx.ListAllRankedChoiceAccessCodesForRegistry(ctx)
	result := make([]ids.XID, 0, len(rows))
	for _, row := range rows {
		result = append(result, row.ID)
	}
	return result, err
}

func fetchRankedChoiceAccessCodeByID(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	row, err := tx.FindRankedChoiceAccessCodeForRegistry(ctx, id)
	return row.ID != "", err
}

func revokeRankedChoiceAccessCode(ctx context.Context, tx *data.Tx, id ids.XID) (bool, error) {
	return tx.RevokeRankedChoiceAccessCodeForRegistry(ctx, id)
}

func immutableUpdate(context.Context, *data.Tx, ids.XID) (bool, error) { return false, nil }
func immutableDelete(context.Context, *data.Tx, ids.XID) (bool, error) { return false, nil }

func insertInterestProfileSubmissionWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createInterestFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_submissions (organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_label) values ($1, $2, $3, $4, 'student_code', 'link', 'foreign')`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.student.ID)
}

func insertInterestProfileResponseWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createInterestFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	submission, err := fixture.factory.SubmitInterestProfile(ctx, preference.InterestProfileSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelStudentCode, Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}}})
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into interest_profile_responses (organization_id, school_year_id, program_id, submission_id, interest_area_id, response) values ($1, $2, $3, $4, $5, 'interested')`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, submission.ID, fixture.area.ID)
}

func insertRankedChoiceSubmissionWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createRankedFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_label) values ($1, $2, $3, $4, $5, 'student_code', 'link', 'foreign')`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.session.ID, fixture.student.ID)
}

func insertRankedChoiceResponseWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createRankedFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	submission, err := fixture.factory.SubmitRankedChoices(ctx, preference.RankedChoiceSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: fixture.session.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelStudentCode, Responses: []data.RankedChoiceResponseInput{{OfferingID: fixture.offering.ID, Answer: data.RankedChoiceInterested}}})
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into ranked_choice_responses (organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response) values ($1, $2, $3, $4, $5, $6, 'interested')`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.session.ID, submission.ID, fixture.offering.ID)
}

func insertRankedChoiceAccessCodeWithForeignParent(ctx context.Context, harness *testharness.Harness, tenantID, foreignOrganizationID ids.XID) error {
	fixture, err := createRankedFixture(ctx, harness, foreignOrganizationID)
	if err != nil {
		return err
	}
	return insertForeign(ctx, harness, tenantID, `insert into ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, student_id, code_hash) values ($1, $2, $3, $4, $5, $6)`, foreignOrganizationID, fixture.year.ID, fixture.program.ID, fixture.session.ID, fixture.student.ID, "foreign-code-hash")
}

func insertForeign(ctx context.Context, harness *testharness.Harness, tenantID ids.XID, query string, args ...any) error {
	if harness == nil || harness.App == nil {
		return errors.New("preference foreign-parent fixture: app pool is nil")
	}
	tx, err := harness.App.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", string(tenantID)); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, query, args...)
	return err
}
