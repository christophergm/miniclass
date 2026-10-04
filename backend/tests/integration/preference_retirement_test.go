package integration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/schoolyear"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
)

func TestStudentCodeMigrationPreservesHistoryAndRejectsNewWrites(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "retirement")
	surveyID := openPreferenceSurvey(t, harness, organizationID, actor, fixture)
	session, offering := openPreferenceSession(t, harness, factory, fixture)

	migration, err := os.ReadFile("../../migrations/20261004083514_retire_student_access_codes.sql")
	require.NoError(t, err)
	up, down, ok := strings.Cut(string(migration), "-- +goose Down")
	require.True(t, ok)
	tx, err := harness.Migrator.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()
	// The harness also holds this lock while replacing shared public functions.
	_, err = tx.Exec(ctx, "select pg_advisory_xact_lock($1)", int64(4_939_441_364_617_518_915))
	require.NoError(t, err)
	_, err = tx.Exec(ctx, down)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
	require.NoError(t, err)

	var legacyProfileID, legacySurveyID, legacyRankedID ids.XID
	require.NoError(t, tx.QueryRow(ctx, `insert into interest_profile_submissions (organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_label, submitted_at)
		values ($1, $2, $3, $4, 'student_code', 'link', 'legacy student respondent', now() - interval '2 hours') returning id`, organizationID, fixture.year.ID, fixture.program.ID, fixture.student.ID).Scan(&legacyProfileID))
	require.NoError(t, tx.QueryRow(ctx, `insert into interest_profile_submissions (organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_label, submitted_at)
		values ($1, $2, $3, $4, $5, 'student_code', 'link', 'legacy survey respondent', now() - interval '1 hour') returning id`, organizationID, fixture.year.ID, fixture.program.ID, surveyID, fixture.student.ID).Scan(&legacySurveyID))
	_, err = tx.Exec(ctx, `insert into interest_profile_responses (organization_id, school_year_id, program_id, submission_id, interest_area_id, response)
		values ($1, $2, $3, $4, $5, 'interested'), ($1, $2, $3, $6, $7, 'very_interested')`, organizationID, fixture.year.ID, fixture.program.ID, legacyProfileID, fixture.secondArea.ID, legacySurveyID, fixture.area.ID)
	require.NoError(t, err)
	require.NoError(t, tx.QueryRow(ctx, `insert into ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_label, submitted_at)
		values ($1, $2, $3, $4, $5, 'student_code', 'link', 'legacy ranked respondent', now() - interval '1 hour') returning id`, organizationID, fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID).Scan(&legacyRankedID))
	_, err = tx.Exec(ctx, `insert into ranked_choice_responses (organization_id, school_year_id, program_id, session_id, submission_id, offering_id, response)
		values ($1, $2, $3, $4, $5, $6, 'interested')`, organizationID, fixture.year.ID, fixture.program.ID, session.ID, legacyRankedID, offering.ID)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `insert into interest_profile_survey_access_codes (organization_id, school_year_id, program_id, survey_id, student_id, code_hash) values ($1, $2, $3, $4, $5, 'legacy-survey-hash')`, organizationID, fixture.year.ID, fixture.program.ID, surveyID, fixture.student.ID)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `insert into ranked_choice_access_codes (organization_id, school_year_id, program_id, session_id, student_id, code_hash) values ($1, $2, $3, $4, $5, 'legacy-ranked-hash')`, organizationID, fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)

	queries := []string{
		`select jsonb_agg(to_jsonb(s) order by id)::text from interest_profile_submissions s where organization_id = $1 and student_id = $2`,
		`select jsonb_agg(to_jsonb(r) order by id)::text from interest_profile_responses r where organization_id = $1 and school_year_id = $2`,
		`select jsonb_agg(to_jsonb(s) order by id)::text from ranked_choice_submissions s where organization_id = $1 and student_id = $2`,
		`select jsonb_agg(to_jsonb(r) order by id)::text from ranked_choice_responses r where organization_id = $1 and school_year_id = $2`,
	}
	before := make([]string, len(queries))
	for i, query := range queries {
		id := fixture.student.ID
		if i%2 == 1 {
			id = fixture.year.ID
		}
		require.NoError(t, tx.QueryRow(ctx, query, organizationID, id).Scan(&before[i]))
	}
	// Existing closed-year history must survive without bypassing mutation guards.
	_, err = tx.Exec(ctx, `update school_years set state = 'closed' where organization_id = $1 and id = $2`, organizationID, fixture.year.ID)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, up)
	require.NoError(t, err)
	for i, query := range queries {
		id := fixture.student.ID
		if i%2 == 1 {
			id = fixture.year.ID
		}
		var after string
		require.NoError(t, tx.QueryRow(ctx, query, organizationID, id).Scan(&after))
		require.JSONEq(t, before[i], after)
	}
	require.NoError(t, tx.Commit(ctx))

	years := schoolyear.New(harness.Database)
	_, err = years.Update(ctx, string(organizationID), fixture.year.ID, auth.RoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearActive), Reason: "synthetic retirement follow-up"})
	require.NoError(t, err)
	service := preference.New(harness.Database)
	form, err := service.GetInterestProfileForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, surveyID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileVeryInterested, *form.InterestAnswers[0].Rating)
	require.Equal(t, data.InterestProfileInterested, *form.InterestAnswers[1].Rating)
	latest, _, err := service.LatestRankedChoices(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, legacyRankedID, latest.ID)
	require.Equal(t, data.PreferenceChannelStudentCode, latest.Channel)
	require.Nil(t, latest.ActorAdultID)
	tracking, err := service.GetInterestProfileResponseTracking(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, surveyID)
	require.NoError(t, err)
	require.Equal(t, 1, tracking.RespondedStudents)

	for _, table := range []string{"interest_profile_survey_access_codes", "ranked_choice_access_codes"} {
		var exists bool
		require.NoError(t, harness.Migrator.QueryRow(ctx, `select exists(select 1 from information_schema.tables where table_schema = current_schema() and table_name = $1)`, table).Scan(&exists))
		require.False(t, exists, table)
	}
	linkActor := audit.Actor{Type: audit.ActorTypeLink, Label: "legacy code respondent"}
	for _, ranked := range []bool{false, true} {
		err = harness.Database.InTenant(ctx, string(organizationID), linkActor, func(ctx context.Context, tx *data.Tx) error {
			if ranked {
				_, _, err := tx.CreateRankedChoiceSubmission(ctx, fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID, data.PreferenceChannelStudentCode, nil, []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceInterested}})
				return err
			}
			_, _, err := tx.CreateInterestProfileSurveySubmission(ctx, fixture.year.ID, fixture.program.ID, surveyID, fixture.student.ID, data.PreferenceChannelStudentCode, nil, []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}})
			return err
		})
		require.ErrorIs(t, err, data.ErrPreferenceChannelRetired)
		appTx, err := harness.App.Begin(ctx)
		require.NoError(t, err)
		_, err = appTx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
		require.NoError(t, err)
		if ranked {
			_, err = appTx.Exec(ctx, `insert into ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_label) values ($1, $2, $3, $4, $5, 'student_code', 'link', 'forged')`, organizationID, fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
		} else {
			_, err = appTx.Exec(ctx, `insert into interest_profile_submissions (organization_id, school_year_id, program_id, student_id, channel, actor_type, actor_label) values ($1, $2, $3, $4, 'student_code', 'link', 'forged')`, organizationID, fixture.year.ID, fixture.program.ID, fixture.student.ID)
		}
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "23514", pgErr.Code)
		require.Contains(t, pgErr.Message, "student_code preference submissions are retired")
		require.NoError(t, appTx.Rollback(ctx))
	}
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), actor, preference.InterestProfileSurveySubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: surveyID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf, Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileNotInterested}}})
	require.NoError(t, err)
	form, err = service.GetInterestProfileForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, surveyID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileNotInterested, *form.InterestAnswers[0].Rating)
	require.Equal(t, data.InterestProfileInterested, *form.InterestAnswers[1].Rating, "omitted historical answers remain effective")
	current, err := service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf, Responses: []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceNotInterested}}})
	require.NoError(t, err)
	latest, _, err = service.LatestRankedChoices(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, current.ID, latest.ID)
	_, err = years.Update(ctx, string(organizationID), fixture.year.ID, auth.RoleOwner, actor, schoolyear.UpdateInput{State: statePtr(data.SchoolYearClosed)})
	require.NoError(t, err)
	_, err = years.Purge(ctx, string(organizationID), fixture.year.ID, auth.RoleOwner, actor, "PURGE "+fixture.year.Label)
	require.NoError(t, err, "purge must remove legacy and current answers without credential-table dependencies")
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		interest, err := tx.ListInterestProfileSubmissions(ctx, fixture.year.ID, fixture.program.ID, fixture.student.ID)
		if err != nil {
			return err
		}
		require.Empty(t, interest)
		ranked, err := tx.ListRankedChoiceSubmissions(ctx, fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
		if err != nil {
			return err
		}
		require.Empty(t, ranked)
		return nil
	})
	require.NoError(t, err)
}

func openPreferenceSession(t *testing.T, harness *testharness.Harness, factory *factories.Factory, fixture preferenceFixture) (data.Session, data.Offering) {
	t.Helper()
	ctx := harness.Context
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Preference Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering", "Synthetic description", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	opened, err := factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	return opened.Session, offering
}

func TestPlaceholderAndStudentDeletionHaveNoCredentialDependencies(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "placeholder retirement")
	surveyID := openPreferenceSurvey(t, harness, organizationID, actor, fixture)
	session, _ := openPreferenceSession(t, harness, factory, fixture)
	unlinked, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Unlinked", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	err = harness.Database.InTenant(ctx, string(organizationID), actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.CountStudentAssociatedData(ctx, fixture.year.ID, unlinked.ID); err != nil {
			return err
		}
		if err := tx.HardDeleteStudent(ctx, fixture.year.ID, unlinked.ID); err != nil {
			return err
		}
		return tx.Record(ctx, audit.Entry{Action: audit.ActionHardDelete, ObjectType: "student", ObjectID: &unlinked.ID, SchoolYearID: &fixture.year.ID})
	})
	require.NoError(t, err)
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		_, err := tx.GetStudentByIDIncludingDeleted(ctx, fixture.year.ID, unlinked.ID)
		return err
	})
	require.Error(t, err)
	candidate, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Placeholder Candidate", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	appTx, err := harness.App.Begin(ctx)
	require.NoError(t, err)
	_, err = appTx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
	require.NoError(t, err)
	_, err = appTx.Exec(ctx, `update students set is_placeholder = true, provenance = 'placeholder' where organization_id = $1 and id = $2`, organizationID, candidate.ID)
	require.NoError(t, err, "the no-dependencies branch must not query dropped tables")
	require.NoError(t, appTx.Commit(ctx))
	for _, query := range []string{
		`insert into ranked_choice_submissions (organization_id, school_year_id, program_id, session_id, student_id, channel, actor_type, actor_user_id, actor_label) values ($1, $2, $3, $4, $5, 'administrator_on_behalf', 'user', $6, 'synthetic admin')`,
		`insert into interest_profile_submissions (organization_id, school_year_id, program_id, survey_id, student_id, channel, actor_type, actor_user_id, actor_label) values ($1, $2, $3, $4, $5, 'administrator_on_behalf', 'user', $6, 'synthetic admin')`,
	} {
		instrumentID := session.ID
		if strings.Contains(query, "interest_profile") {
			instrumentID = surveyID
		}
		appTx, err := harness.App.Begin(ctx)
		require.NoError(t, err)
		_, err = appTx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
		require.NoError(t, err)
		_, err = appTx.Exec(ctx, query, organizationID, fixture.year.ID, fixture.program.ID, instrumentID, candidate.ID, actor.UserID)
		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		require.Equal(t, "23514", pgErr.Code)
		require.Contains(t, pgErr.Message, "placeholder students")
		require.NoError(t, appTx.Rollback(ctx))
	}
	appTx, err = harness.App.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = appTx.Rollback(ctx) }()
	_, err = appTx.Exec(ctx, "select set_config('app.organization_id', $1, true)", organizationID)
	require.NoError(t, err)
	_, err = appTx.Exec(ctx, `update students set is_placeholder = true where organization_id = $1 and id = $2`, organizationID, fixture.student.ID)
	var pgErr *pgconn.PgError
	require.ErrorAs(t, err, &pgErr)
	require.Equal(t, "23514", pgErr.Code)
	require.Contains(t, pgErr.Message, "cannot become a placeholder")
}
