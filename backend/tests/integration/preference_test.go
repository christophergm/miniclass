package integration

import (
	"context"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestInterestProfileSubmissionsOverlayAndRetainAttribution(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "interest")
	adult, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Guardian"})
	require.NoError(t, err)
	service := preference.New(harness.Database)
	surveyID := openPreferenceSurvey(t, harness, organizationID, actor, fixture)
	_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: fixture.student.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)

	first, err := service.SubmitInterestProfileSurvey(ctx, string(organizationID), actor, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: surveyID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelAdministratorOnBehalf,
		Answers: []data.InterestProfileAnswer{
			{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested},
			{InterestAreaID: fixture.secondArea.ID, Rating: data.InterestProfileInterested},
		},
	})
	require.NoError(t, err)
	second, err := service.SubmitInterestProfileSurvey(ctx, string(organizationID), audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic guardian"}, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: surveyID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &adult.ID, GuardianAdultID: &adult.ID,
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
	require.Equal(t, data.PreferenceChannelAdministratorOnBehalf, history[0].Channel)
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

// A membership is retained for history when a student is soft-deleted, but the
// deleted student must not have a respondent form.
func TestRankedChoiceOpeningExcludesSoftDeletedMembers(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "ranked-choice deleted member test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "deleted-member")

	deleted, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{
		LegalGivenName: "Synthetic", LegalFamilyName: "Deleted Member", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID,
	})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, deleted.ID)
	require.NoError(t, err)
	require.NoError(t, people.New(harness.Database).DeleteStudent(ctx, string(organizationID), fixture.year.ID, deleted.ID, actor))

	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Voting Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	_, err = factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering", "Synthetic description", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)

	opened, err := factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	require.Equal(t, data.SessionVotingOpen, opened.Session.State)
	service := preference.New(harness.Database)
	_, err = service.GetRankedChoiceForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, deleted.ID)
	require.Error(t, err, "deleted students must not have a respondent form")
	_, err = service.GetRankedChoiceForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
}

func TestInvalidRankedChoiceSubmissionDoesNotReplaceValidResponse(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "ranked")
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Voting Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offeringA, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering A", "Synthetic description A", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	offeringB, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Offering B", "Synthetic description B", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	service := preference.New(harness.Database)
	rankOne := 1
	valid, err := service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{
			{OfferingID: offeringA.ID, Answer: data.RankedChoiceRanked, Rank: &rankOne},
			{OfferingID: offeringB.ID, Answer: data.RankedChoiceInterested},
		},
	})
	require.NoError(t, err)

	duplicateRank := 1
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelAdministratorOnBehalf,
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

func TestRankedChoiceWindowFollowsSessionLifecycle(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "ranked-window")
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Voting Window", []time.Time{time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Voting Offering", "Synthetic description", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	deadline := time.Now().UTC().Add(time.Hour)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, deadline)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)

	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionAssigning, false, "", nil)
	require.ErrorIs(t, err, program.ErrSessionTransitionInvalid)

	service := preference.New(harness.Database)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID,
		StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceInterested}},
	})
	require.NoError(t, err)

	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingClosed, false, "", nil)
	require.NoError(t, err)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID,
		StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceInterested}},
	})
	require.ErrorIs(t, err, preference.ErrRankedChoiceNotAccepting)

	preview, err := factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	require.False(t, preview.Applied)
	require.Contains(t, transitionWarningCodes(preview.Warnings), "ranked-choice-reopened")
	newDeadline := time.Now().UTC().Add(2 * time.Hour)
	reopened, err := factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, true, "reopening for late response", &newDeadline)
	require.NoError(t, err)
	require.True(t, reopened.Applied)
	require.NotNil(t, reopened.Session.RankedChoice)
	require.WithinDuration(t, newDeadline, *reopened.Session.RankedChoice.Deadline, time.Second)
}

// SPEC §11.7 permits placeholder membership but prohibits preferences; §13.6.4
// snapshots only eligible respondents when a survey opens.
func TestInterestProfileSurveyExcludesPlaceholderMembers(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "placeholder survey audience test"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "placeholder audience")
	peopleService := people.New(harness.Database)
	placeholder, err := peopleService.CreatePlaceholderStudent(ctx, string(organizationID), fixture.year.ID, actor, people.PlaceholderStudentInput{
		LegalGivenName: "Unknown", LegalFamilyName: "Synthetic", GradeLevelID: fixture.grade.ID, HomeroomID: fixture.student.HomeroomID, Reason: "synthetic unregistered member",
	})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, placeholder.ID)
	require.NoError(t, err)
	service := preference.New(harness.Database)
	prior, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic prior survey", Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
	})
	require.NoError(t, err)
	notResponded := data.SurveyNotResponded
	for _, audience := range []preference.InterestProfileSurveyAudienceInput{
		{Type: data.SurveyAudienceAllMembers},
		{Type: data.SurveyAudienceGradeLevel, GradeLevelID: &fixture.grade.ID},
		{Type: data.SurveyAudienceResponseState, PriorSurveyID: &prior.Survey.ID, ResponseState: &notResponded},
	} {
		t.Run(string(audience.Type), func(t *testing.T) {
			survey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
				Name: "Synthetic audience survey", Audience: audience, Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
			})
			require.NoError(t, err)
			closingAt := time.Now().UTC().Add(time.Hour)
			opened, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
			require.NoError(t, err)
			require.Equal(t, data.InterestProfileSurveyOpen, opened.Survey.Survey.State)
			require.Len(t, opened.Survey.AudienceSnapshot, 1)
			require.Equal(t, fixture.student.ID, opened.Survey.AudienceSnapshot[0].StudentID)
		})
	}
	latePlaceholder, err := peopleService.CreatePlaceholderStudent(ctx, string(organizationID), fixture.year.ID, actor, people.PlaceholderStudentInput{
		LegalGivenName: "Unknown", LegalFamilyName: "Late Synthetic", GradeLevelID: fixture.grade.ID, HomeroomID: fixture.student.HomeroomID, Reason: "synthetic late member",
	})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, latePlaceholder.ID)
	require.NoError(t, err, "placeholder membership must not attempt late survey audience insertion")
	surveys, err := service.ListInterestProfileSurveys(ctx, string(organizationID), fixture.year.ID, fixture.program.ID)
	require.NoError(t, err)
	for _, survey := range surveys {
		if survey.Survey.State == data.InterestProfileSurveyOpen {
			require.Len(t, survey.AudienceSnapshot, 1)
			require.Equal(t, fixture.student.ID, survey.AudienceSnapshot[0].StudentID)
		}
	}
}

func TestInterestProfileSurveyLifecycleFreezesAudienceAndRetainsScale(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "synthetic survey organizer"}
	respondentActor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "survey")
	secondStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Second Survey", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, secondStudent.ID)
	require.NoError(t, err)
	service := preference.New(harness.Database)

	survey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name:      "Synthetic Student Interest Survey",
		Audience:  preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
	})
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileSurveyDraft, survey.Survey.State)
	require.Len(t, survey.Questions, 1)
	require.Equal(t, fixture.area.ID, survey.Questions[0].InterestAreaID)
	require.Len(t, survey.ScaleOptions, 3)
	require.Equal(t, "Very interested", survey.ScaleOptions[0].Label)

	closingAt := time.Now().UTC().Add(time.Hour)
	opened, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileSurveyOpen, opened.Survey.Survey.State)
	require.Len(t, opened.Survey.AudienceSnapshot, 2)
	require.Empty(t, opened.Warnings)

	thirdStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Third Survey", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, thirdStudent.ID)
	require.NoError(t, err)
	current, err := service.GetInterestProfileSurvey(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID)
	require.NoError(t, err)
	require.Len(t, current.AudienceSnapshot, 3, "a member added while the survey is open is added to its audience")
	require.Contains(t, []ids.XID{
		current.AudienceSnapshot[0].StudentID,
		current.AudienceSnapshot[1].StudentID,
		current.AudienceSnapshot[2].StudentID,
	}, thirdStudent.ID)

	_, err = service.UpdateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyUpdate{InterestProfileSurveyInput: preference.InterestProfileSurveyInput{
		Name: "Changed after opening", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers}, Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.secondArea.ID}},
	}})
	require.ErrorIs(t, err, preference.ErrSurveyDefinitionLocked)

	closed, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyClosed, Reason: "synthetic close"})
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileSurveyClosed, closed.Survey.Survey.State)
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondentActor, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}},
	})
	require.ErrorIs(t, err, preference.ErrSurveyNotAcceptingSubmissions)

	reopenedClosingAt := time.Now().UTC().Add(2 * time.Hour)
	reopened, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &reopenedClosingAt, Reason: "synthetic reopen"})
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileSurveyOpen, reopened.Survey.Survey.State)
	require.Contains(t, reopened.Warnings, preference.SurveyWarningReopened)

	firstSubmission, err := service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondentActor, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}},
	})
	require.NoError(t, err)
	require.NotNil(t, firstSubmission.SurveyID)
	require.Equal(t, survey.Survey.ID, *firstSubmission.SurveyID)

	err = service.DeleteInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID)
	require.ErrorIs(t, err, preference.ErrSurveyHasSubmissions)

	priorSurveyID := survey.Survey.ID
	notResponded := data.SurveyNotResponded
	secondSurvey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name:      "Synthetic Follow-up Survey",
		Audience:  preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceResponseState, PriorSurveyID: &priorSurveyID, ResponseState: &notResponded},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.secondArea.ID}},
	})
	require.NoError(t, err)
	secondOpenedClosingAt := time.Now().UTC().Add(time.Hour)
	secondOpened, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, secondSurvey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &secondOpenedClosingAt})
	require.NoError(t, err)
	require.Len(t, secondOpened.Survey.AudienceSnapshot, 2)
	require.ElementsMatch(t, []ids.XID{secondStudent.ID, thirdStudent.ID}, []ids.XID{
		secondOpened.Survey.AudienceSnapshot[0].StudentID,
		secondOpened.Survey.AudienceSnapshot[1].StudentID,
	})
}

func TestInterestProfileSurveyAllowsEmptyAudienceAndStopsAtDeadline(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := audit.Actor{Type: audit.ActorTypeSystem, Label: "synthetic empty survey organizer"}
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "empty survey")
	service := preference.New(harness.Database)
	survey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name:      "Synthetic Empty Audience Survey",
		Audience:  preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceExplicitStudents},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
	})
	require.NoError(t, err)
	closingAt := time.Now().UTC().Add(10 * time.Millisecond)
	opened, err := service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)
	require.Contains(t, opened.Warnings, preference.SurveyWarningEmptyAudience)
	require.Empty(t, opened.Survey.AudienceSnapshot)

	time.Sleep(20 * time.Millisecond)
	current, err := service.GetInterestProfileSurvey(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID)
	require.NoError(t, err)
	require.Equal(t, data.InterestProfileSurveyClosed, current.Survey.State)
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), actor, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}},
	})
	require.ErrorIs(t, err, preference.ErrSurveyNotAcceptingSubmissions)
}

func TestPreferenceFormsRespectGuardianScopeAndSupportEverySubmissionMode(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	organizer := audit.Actor{Type: audit.ActorTypeSystem, Label: "synthetic preference form organizer"}
	respondent := audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic preference form respondent"}
	factory := factories.New(harness.Database, string(organizationID), organizer)
	fixture := createPreferenceFixture(t, harness, factory, "forms")

	otherStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Other Guardian", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	noGuardianStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "No Guardian", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	for _, student := range []data.Student{otherStudent, noGuardianStudent} {
		_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, student.ID)
		require.NoError(t, err)
	}
	firstAdult, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "First Guardian"})
	require.NoError(t, err)
	secondAdult, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Second Guardian"})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: firstAdult.ID, StudentID: fixture.student.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: secondAdult.ID, StudentID: otherStudent.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)

	unrelatedProgram, err := factory.CreateProgram(ctx, fixture.year.ID, "Synthetic Unrelated Program")
	require.NoError(t, err)
	unrelatedSession, err := factory.CreateSession(ctx, fixture.year.ID, unrelatedProgram.ID, "Synthetic Unrelated Voting Session", []time.Time{time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	_, err = factory.CreateOffering(ctx, fixture.year.ID, unrelatedProgram.ID, unrelatedSession.ID, "Synthetic Unrelated Course", "Synthetic unrelated course", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, unrelatedProgram.ID, unrelatedSession.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, unrelatedProgram.ID, unrelatedSession.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, unrelatedProgram.ID, unrelatedSession.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)

	service := preference.New(harness.Database)
	survey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), organizer, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name:      "Synthetic Respondent Interest Form",
		Audience:  preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}, {InterestAreaID: fixture.secondArea.ID}},
	})
	require.NoError(t, err)
	closingAt := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(ctx, string(organizationID), organizer, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)

	guardianForms, err := service.ListGuardianPreferenceForms(ctx, string(organizationID), fixture.year.ID, firstAdult.ID)
	require.NoError(t, err)
	require.Len(t, guardianForms.Students, 1)
	require.Equal(t, fixture.student.ID, guardianForms.Students[0].StudentID)
	require.Len(t, guardianForms.Students[0].Forms, 1)
	require.Equal(t, preference.FormTypeInterestProfile, guardianForms.Students[0].Forms[0].Type)

	otherGuardianForms, err := service.ListGuardianPreferenceForms(ctx, string(organizationID), fixture.year.ID, secondAdult.ID)
	require.NoError(t, err)
	require.Len(t, otherGuardianForms.Students, 1)
	require.Equal(t, otherStudent.ID, otherGuardianForms.Students[0].StudentID)

	form, err := service.GetInterestProfileForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, fixture.student.ID, form.StudentID)
	require.NotEmpty(t, form.StudentName)

	firstRating := data.InterestProfileVeryInterested
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondent, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &firstAdult.ID, GuardianAdultID: &firstAdult.ID,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: firstRating}},
	})
	require.NoError(t, err)
	secondRating := data.InterestProfileInterested
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondent, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &firstAdult.ID, GuardianAdultID: &firstAdult.ID,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.secondArea.ID, Rating: secondRating}},
	})
	require.NoError(t, err)
	updatedForm, err := service.GetInterestProfileForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, firstRating, *updatedForm.InterestAnswers[0].Rating)
	require.Equal(t, secondRating, *updatedForm.InterestAnswers[1].Rating)

	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondent, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &secondAdult.ID, GuardianAdultID: &secondAdult.ID,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileNotInterested}},
	})
	require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)

	var adminID ids.XID
	err = harness.Migrator.QueryRow(ctx, `insert into users (provider_subject, email) values ($1, $2) returning id`, "synthetic-preference-admin", "synthetic-preference-admin@example.test").Scan(&adminID)
	require.NoError(t, err)
	adminActor := audit.Actor{Type: audit.ActorTypeUser, UserID: &adminID, Label: "synthetic-admin@example.test"}
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), adminActor, preference.InterestProfileSurveySubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: noGuardianStudent.ID,
		Channel: data.PreferenceChannelAdministratorOnBehalf,
		Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileNotInterested}},
	})
	require.NoError(t, err)

	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Preference Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offeringA, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Course A", "Synthetic course A", nil, 10, fixture.grade.ID, fixture.grade.ID, "Room A", "", "", nil)
	require.NoError(t, err)
	offeringB, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Course B", "Synthetic course B", nil, 10, fixture.grade.ID, fixture.grade.ID, "Room B", "", "", nil)
	require.NoError(t, err)
	olderGrade, err := factory.CreateGradeLevel(ctx, fixture.year.ID, "older", "Older grade")
	require.NoError(t, err)
	hiddenOffering, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Older Course", "Older students only", nil, 10, olderGrade.ID, olderGrade.ID, "Room C", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	rankedForm, err := service.GetRankedChoiceForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Len(t, rankedForm.Offerings, 2)
	require.Equal(t, offeringA.ID, rankedForm.Offerings[0].ID)
	require.Equal(t, offeringB.ID, rankedForm.Offerings[1].ID)
	position := 1
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), respondent, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel: data.PreferenceChannelGuardian, ActorAdultID: &firstAdult.ID, GuardianAdultID: &firstAdult.ID,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: offeringA.ID, Answer: data.RankedChoiceRanked, Rank: &position}, {OfferingID: offeringB.ID, Answer: data.RankedChoiceInterested}},
	})
	require.NoError(t, err)
	updatedRankedForm, err := service.GetRankedChoiceForm(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Len(t, updatedRankedForm.Offerings, 2)
	require.Len(t, updatedRankedForm.RankedAnswers, 3)
	for _, answer := range updatedRankedForm.RankedAnswers {
		if answer.OfferingID == hiddenOffering.ID {
			require.Equal(t, data.RankedChoiceNoResponse, answer.Answer)
			require.Nil(t, answer.Rank)
		}
	}
}

func TestResponseTrackingUsesStudentDenominatorAndGuardianFollowUp(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	organizer := audit.Actor{Type: audit.ActorTypeSystem, Label: "synthetic response tracking organizer"}
	respondent := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), organizer)
	fixture := createPreferenceFixture(t, harness, factory, "response tracking")
	secondStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Tracking Second", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	thirdStudent, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Tracking Third", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	for _, student := range []data.Student{secondStudent, thirdStudent} {
		_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, student.ID)
		require.NoError(t, err)
	}
	guardianEmail := "synthetic-tracking-guardian@example.test"
	firstGuardian, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Tracking Guardian One", Email: &guardianEmail})
	require.NoError(t, err)
	secondGuardian, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Tracking Guardian Two"})
	require.NoError(t, err)
	for _, adult := range []data.Adult{firstGuardian, secondGuardian} {
		_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: secondStudent.ID, RelationshipType: data.GuardianRelationshipParent})
		require.NoError(t, err)
	}

	service := preference.New(harness.Database)
	survey, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), organizer, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic Response Tracking Survey", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
	})
	require.NoError(t, err)
	closingAt := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(ctx, string(organizationID), organizer, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)
	for _, student := range []data.Student{fixture.student, secondStudent} {
		_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), respondent, preference.InterestProfileSurveySubmissionInput{
			SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: student.ID,
			Channel: data.PreferenceChannelAdministratorOnBehalf,
			Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}},
		})
		require.NoError(t, err)
	}

	tracking, err := service.GetInterestProfileResponseTracking(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID)
	require.NoError(t, err)
	require.Equal(t, 3, tracking.TotalStudents)
	require.Equal(t, 2, tracking.RespondedStudents)
	require.InDelta(t, 66.666666, tracking.CompletionPercentage, 0.000001)
	require.Len(t, tracking.NonResponders, 1)
	require.Equal(t, thirdStudent.ID, tracking.NonResponders[0].StudentID)
	require.Equal(t, preference.ResponseTrackingUnreachable, tracking.NonResponders[0].ContactStatus)
	require.Empty(t, tracking.GuardianFollowUp)
	require.Equal(t, 3, tracking.GradeBreakdown[0].TotalStudents)

	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Response Tracking Session", []time.Time{time.Date(2026, 11, 27, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	offering, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Tracking Offering", "Synthetic tracking offering", nil, 10, fixture.grade.ID, fixture.grade.ID, "", "", "", nil)
	require.NoError(t, err)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), respondent, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID,
		Channel:   data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceInterested}},
	})
	require.NoError(t, err)

	rankedTracking, err := service.GetRankedChoiceResponseTracking(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, 3, rankedTracking.TotalStudents)
	require.Equal(t, 1, rankedTracking.RespondedStudents)
	require.Len(t, rankedTracking.NonResponders, 2)
	require.Len(t, rankedTracking.GuardianFollowUp, 2)
	require.ElementsMatch(t, []string{preference.ResponseTrackingGuardianPending, preference.ResponseTrackingGuardianNoEmail}, []string{rankedTracking.GuardianFollowUp[0].ContactStatus, rankedTracking.GuardianFollowUp[1].ContactStatus})

	foreignOrganizationID := harness.MintOrganization(t)
	_, err = service.GetInterestProfileResponseTracking(ctx, string(foreignOrganizationID), fixture.year.ID, fixture.program.ID, survey.Survey.ID)
	require.Error(t, err)
}

func preferenceAdminActor(t *testing.T, harness *testharness.Harness, organizationID ids.XID) audit.Actor {
	t.Helper()
	var userID ids.XID
	require.NoError(t, harness.Migrator.QueryRow(harness.Context, `insert into users (provider_subject, email) values ($1, $2) returning id`, "preference-admin-"+string(organizationID), "synthetic-admin-"+string(organizationID)+"@example.test").Scan(&userID))
	_, err := harness.Migrator.Exec(harness.Context, `insert into organization_members (organization_id, user_id, role) values ($1, $2, 'administrator')`, organizationID, userID)
	require.NoError(t, err)
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "synthetic-admin-" + string(organizationID) + "@example.test"}
}

func openPreferenceSurvey(t *testing.T, harness *testharness.Harness, organizationID ids.XID, actor audit.Actor, fixture preferenceFixture) ids.XID {
	t.Helper()
	service := preference.New(harness.Database)
	survey, err := service.CreateInterestProfileSurvey(harness.Context, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic Preference Survey", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
		Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}, {InterestAreaID: fixture.secondArea.ID}},
	})
	require.NoError(t, err)
	closingAt := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(harness.Context, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)
	return survey.Survey.ID
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
