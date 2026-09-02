package integration

import (
	"context"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestInterestProfileSubmissionsOverlayAndRetainAttribution(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic preference respondent"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "interest")
	adult, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Guardian"})
	require.NoError(t, err)
	service := preference.New(harness.Database)

	first, err := service.SubmitInterestProfile(ctx, string(organizationID), actor, preference.InterestProfileSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelStudentCode,
		Answers: []data.InterestProfileAnswer{
			{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested},
			{InterestAreaID: fixture.secondArea.ID, Rating: data.InterestProfileInterested},
		},
	})
	require.NoError(t, err)
	second, err := service.SubmitInterestProfile(ctx, string(organizationID), actor, preference.InterestProfileSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &adult.ID,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileNotInterested}},
	})
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)

	effective, err := service.EffectiveInterestProfile(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Len(t, effective, 2)
	require.Equal(t, data.InterestProfileNotInterested, effective[0].Rating)
	require.Equal(t, fixture.area.ID, effective[0].InterestAreaID)
	require.Equal(t, data.InterestProfileInterested, effective[1].Rating)
	require.Equal(t, fixture.secondArea.ID, effective[1].InterestAreaID)

	var history []data.InterestProfileSubmission
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		var err error
		history, err = tx.ListInterestProfileSubmissions(ctx, fixture.year.ID, fixture.program.ID, fixture.student.ID)
		return err
	})
	require.NoError(t, err)
	require.Len(t, history, 2)
	require.Equal(t, data.PreferenceChannelStudentCode, history[0].Channel)
	require.Nil(t, history[0].ActorAdultID)
	require.Equal(t, data.PreferenceChannelGuardian, history[1].Channel)
	require.Equal(t, adult.ID, *history[1].ActorAdultID)

	objectType := "interest_profile_submission"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 2)
	for _, entry := range entries {
		require.Equal(t, audit.ActionPreferenceSubmission, audit.Action(entry.Action))
	}
}

func TestInvalidRankedChoiceSubmissionDoesNotReplaceValidResponse(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic student-code respondent"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "ranked")
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Voting Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offeringA, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering A", "Synthetic description A", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	offeringB, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering B", "Synthetic description B", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	service := preference.New(harness.Database)
	rankOne := 1
	valid, err := service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelStudentCode,
		Responses: []data.RankedChoiceResponseInput{
			{OfferingID: offeringA.ID, Answer: data.RankedChoiceRanked, Rank: &rankOne},
			{OfferingID: offeringB.ID, Answer: data.RankedChoiceInterested},
		},
	})
	require.NoError(t, err)

	duplicateRank := 1
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelStudentCode,
		Responses: []data.RankedChoiceResponseInput{
			{OfferingID: offeringA.ID, Answer: data.RankedChoiceRanked, Rank: &duplicateRank},
			{OfferingID: offeringB.ID, Answer: data.RankedChoiceRanked, Rank: &duplicateRank},
		},
	})
	require.ErrorIs(t, err, preference.ErrRankedChoiceInvalid)

	latest, responses, err := service.LatestRankedChoices(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, valid.ID, latest.ID)
	require.Len(t, responses, 2)

	objectType := "ranked_choice_submission"
	entries, err := harness.Database.ListAuditLog(ctx, string(organizationID), data.AuditLogFilter{ObjectType: &objectType, PageSize: 100})
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

type preferenceFixture struct {
	year       data.SchoolYear
	grade      data.GradeLevel
	program    data.Program
	student    data.Student
	area       data.InterestArea
	secondArea data.InterestArea
}

func createPreferenceFixture(t *testing.T, harness *testharness.Harness, factory *factories.Factory, label string) preferenceFixture {
	t.Helper()
	ctx := harness.Context
	year, err := factory.CreateSchoolYear(ctx, "Synthetic "+label+" preference year")
	require.NoError(t, err)
	grade, err := factory.CreateGradeLevel(ctx, year.ID, "synthetic-"+label, "Synthetic Preference Grade")
	require.NoError(t, err)
	homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Preference Room")
	require.NoError(t, err)
	student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Preference", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
	require.NoError(t, err)
	programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic "+label+" preference program")
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
	require.NoError(t, err)
	area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Preference Area")
	require.NoError(t, err)
	secondArea, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Second Preference Area")
	require.NoError(t, err)
	return preferenceFixture{year: year, grade: grade, program: programRow, student: student, area: area, secondArea: secondArea}
}
