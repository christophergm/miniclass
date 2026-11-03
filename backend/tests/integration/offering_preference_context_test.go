package integration

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/program"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// SPEC §§13.3–13.5, §16.5, and §19.1: the sidebar is based on current,
// tenant-scoped submissions and retains an already-assigned grade exception.
func TestOfferingPreferenceContextUsesCurrentSubmissionsAndPlacementTags(t *testing.T) {
	harness := testharness.Open(t)
	ctx, organizationID := harness.Context, harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "offering-context")
	otherGrade, err := factory.CreateGradeLevel(ctx, fixture.year.ID, "offering-context-other", "Synthetic Other Grade")
	require.NoError(t, err)
	room, err := factory.CreateHomeroom(ctx, fixture.year.ID, "Synthetic Context Room")
	require.NoError(t, err)
	students := []data.Student{fixture.student}
	for _, name := range []string{"Topic", "Blank", "Outside"} {
		gradeID := fixture.grade.ID
		if name == "Outside" {
			gradeID = otherGrade.ID
		}
		student, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: name, LegalFamilyName: "Synthetic", GradeLevelID: &gradeID, HomeroomID: room.ID})
		require.NoError(t, err)
		_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, student.ID)
		require.NoError(t, err)
		students = append(students, student)
	}
	session, err := factory.CreateSession(ctx, fixture.year.ID, fixture.program.ID, "Synthetic Context Session", []time.Time{time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)})
	require.NoError(t, err)
	target, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Context Target", "Synthetic target description", nil, 3, fixture.grade.ID, fixture.grade.ID, "", "", "", &fixture.area.ID)
	require.NoError(t, err)
	elsewhere, err := factory.CreateOffering(ctx, fixture.year.ID, fixture.program.ID, session.ID, "Synthetic Context Elsewhere", "", nil, 3, fixture.grade.ID, otherGrade.ID, "", "", "", nil)
	require.NoError(t, err)
	deadline := time.Now().UTC().Add(time.Hour)
	_, err = factory.ConfigureRankedChoice(ctx, fixture.year.ID, fixture.program.ID, session.ID, 1, deadline)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingOpen, false, "", nil)
	require.NoError(t, err)

	preferenceService := preference.New(harness.Database)
	survey, err := preferenceService.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{
		Name: "Synthetic Context Interests", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers}, Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}},
	})
	require.NoError(t, err)
	_, err = preferenceService.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &deadline})
	require.NoError(t, err)
	for _, student := range []data.Student{students[1], students[2]} {
		_, err = preferenceService.SubmitInterestProfileSurvey(ctx, string(organizationID), actor, preference.InterestProfileSurveySubmissionInput{
			SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: survey.Survey.ID, StudentID: student.ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
			Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileVeryInterested}},
		})
		require.NoError(t, err)
	}
	rankOne := 1
	_, err = preferenceService.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: students[0].ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: target.ID, Answer: data.RankedChoiceRanked, Rank: &rankOne}, {OfferingID: elsewhere.ID, Answer: data.RankedChoiceNoResponse}},
	})
	require.NoError(t, err)
	_, err = preferenceService.SubmitRankedChoices(ctx, string(organizationID), actor, preference.RankedChoiceSubmissionInput{
		SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: students[2].ID, Channel: data.PreferenceChannelAdministratorOnBehalf,
		Responses: []data.RankedChoiceResponseInput{{OfferingID: target.ID, Answer: data.RankedChoiceNoResponse}, {OfferingID: elsewhere.ID, Answer: data.RankedChoiceInterested}},
	})
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionVotingClosed, false, "", nil)
	require.NoError(t, err)
	_, err = factory.TransitionSession(ctx, fixture.year.ID, fixture.program.ID, session.ID, data.SessionAssigning, false, "", nil)
	require.NoError(t, err)

	programService := program.New(harness.Database)
	first, err := programService.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID}, students[0].ID, target.ID)
	require.NoError(t, err)
	second, err := programService.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, ExpectedRevision: first.DraftRevision}, students[2].ID, elsewhere.ID)
	require.NoError(t, err)
	_, err = programService.MoveAssignment(ctx, string(organizationID), actor, program.AssignmentOperationInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, ExpectedRevision: second.DraftRevision, ConfirmViolations: true, Reason: "Synthetic grade exception"}, students[3].ID, target.ID)
	require.NoError(t, err)

	context, err := programService.GetOfferingPreferenceContext(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, target.ID)
	require.NoError(t, err)
	byStudent := offeringContextStudents(context)
	require.Equal(t, program.OfferingPreferenceRank, byStudent[students[0].ID].Preference.Category)
	require.Equal(t, "here", byStudent[students[0].ID].Placement.Status)
	require.Equal(t, program.OfferingPreferenceTopicVeryInterested, byStudent[students[1].ID].Preference.Category)
	require.Equal(t, "unplaced", byStudent[students[1].ID].Placement.Status)
	require.Equal(t, program.OfferingPreferenceUnknown, byStudent[students[2].ID].Preference.Category)
	require.Equal(t, data.InterestProfileVeryInterested, byStudent[students[2].ID].Preference.TopicRatingHint)
	require.Equal(t, "elsewhere", byStudent[students[2].ID].Placement.Status)
	require.Equal(t, elsewhere.Name, byStudent[students[2].ID].Placement.OfferingName)
	require.True(t, byStudent[students[3].ID].OutsideGradeRange)
	require.Equal(t, "here", byStudent[students[3].ID].Placement.Status)

	otherOrganizationID := harness.MintOrganization(t)
	_, err = programService.GetOfferingPreferenceContext(ctx, string(otherOrganizationID), fixture.year.ID, fixture.program.ID, session.ID, target.ID)
	require.Error(t, err, "another organization cannot read this offering context")
}

func offeringContextStudents(context program.OfferingPreferenceContext) map[ids.XID]program.OfferingPreferenceStudent {
	result := make(map[ids.XID]program.OfferingPreferenceStudent)
	for _, group := range context.Groups {
		for _, student := range group.Students {
			result[student.Participant.StudentID] = student
		}
	}
	return result
}
