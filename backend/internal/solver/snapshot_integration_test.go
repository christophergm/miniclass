package solver

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/solvercontract"
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

// SPEC §13.4 distinguishes an unanswered submitted catalog from no submission:
// only the latter falls back to the standing interest profile (#325).
func TestAuthoritativeSnapshotRankedChoiceNoResponse(t *testing.T) {
	for _, test := range []struct {
		name    string
		submit  bool
		mixed   bool
		quality string
	}{
		{name: "mixed answers omit unanswered and grade-hidden offerings", submit: true, mixed: true, quality: solvercontract.QualityTop},
		{name: "all unanswered retains an empty submitted catalog with a profile", submit: true, quality: solvercontract.QualityNeutral},
		{name: "no submission retains nil ranked choices and the profile", quality: solvercontract.QualityTop},
	} {
		t.Run(test.name, func(t *testing.T) {
			harness := testharness.Open(t)
			ctx := harness.Context
			organizationID := harness.MintOrganization(t)
			actor := snapshotPreferenceAdminActor(t, harness, organizationID)
			factory := factories.New(harness.Database, string(organizationID), actor)
			year, err := factory.CreateSchoolYear(ctx, fmt.Sprintf("Synthetic snapshot year %s", organizationID))
			require.NoError(t, err)
			grade, err := factory.CreateGradeLevel(ctx, year.ID, "snapshot-junior", "Synthetic Snapshot Junior")
			require.NoError(t, err)
			seniorGrade, err := factory.CreateGradeLevel(ctx, year.ID, "snapshot-senior", "Synthetic Snapshot Senior")
			require.NoError(t, err)
			homeroom, err := factory.CreateHomeroom(ctx, year.ID, "Synthetic Snapshot Room")
			require.NoError(t, err)
			student, err := factory.CreateStudent(ctx, year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Snapshot Student", GradeLevelID: &grade.ID, HomeroomID: homeroom.ID})
			require.NoError(t, err)
			programRow, err := factory.CreateProgram(ctx, year.ID, "Synthetic Snapshot Program")
			require.NoError(t, err)
			_, err = factory.AddProgramMembership(ctx, year.ID, programRow.ID, student.ID)
			require.NoError(t, err)
			area, err := factory.CreateInterestArea(ctx, year.ID, programRow.ID, "Synthetic Snapshot Interest")
			require.NoError(t, err)

			preferences := preference.New(harness.Database)
			survey, err := preferences.CreateInterestProfileSurvey(ctx, string(organizationID), actor, year.ID, programRow.ID, preference.InterestProfileSurveyInput{
				Name: "Synthetic Snapshot Survey", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceAllMembers},
				Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: area.ID}},
			})
			require.NoError(t, err)
			deadline := time.Now().UTC().Add(time.Hour)
			_, err = preferences.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, year.ID, programRow.ID, survey.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &deadline})
			require.NoError(t, err)
			_, err = factory.SubmitInterestProfileSurvey(ctx, preference.InterestProfileSurveySubmissionInput{
				SchoolYearID: year.ID, ProgramID: programRow.ID, SurveyID: survey.Survey.ID, StudentID: student.ID,
				Channel: data.PreferenceChannelAdministratorOnBehalf,
				Answers: []data.InterestProfileAnswer{{InterestAreaID: area.ID, Rating: data.InterestProfileVeryInterested}},
			})
			require.NoError(t, err)

			session, err := factory.CreateSession(ctx, year.ID, programRow.ID, "Synthetic Snapshot Session", []time.Time{time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)})
			require.NoError(t, err)
			offerings := make([]data.Offering, 0, 5)
			for index, name := range []string{"Ranked", "Interested", "Not Interested", "Unanswered", "Grade Hidden"} {
				offeringGrade := grade.ID
				if index == 4 {
					offeringGrade = seniorGrade.ID
				}
				offering, err := factory.CreateOffering(ctx, year.ID, programRow.ID, session.ID, "Synthetic Snapshot "+name, "Synthetic description", nil, 1, offeringGrade, offeringGrade, "", "", "", &area.ID)
				require.NoError(t, err)
				offerings = append(offerings, offering)
			}
			_, err = factory.ConfigureRankedChoice(ctx, year.ID, programRow.ID, session.ID, 2, deadline)
			require.NoError(t, err)
			_, err = factory.TransitionSession(ctx, year.ID, programRow.ID, session.ID, data.SessionCatalogPublished, false, "", nil)
			require.NoError(t, err)
			_, err = factory.TransitionSession(ctx, year.ID, programRow.ID, session.ID, data.SessionVotingOpen, false, "", nil)
			require.NoError(t, err)

			var expectedChoices *solvercontract.RankedChoices
			if test.submit {
				responses := make([]data.RankedChoiceResponseInput, 0, 4)
				for _, offering := range offerings[:4] {
					responses = append(responses, data.RankedChoiceResponseInput{OfferingID: offering.ID, Answer: data.RankedChoiceNoResponse})
				}
				expectedChoices = &solvercontract.RankedChoices{Choices: []solvercontract.RankedChoice{}}
				if test.mixed {
					rank := 1
					responses[0].Answer, responses[0].Rank = data.RankedChoiceRanked, &rank
					responses[1].Answer = data.RankedChoiceInterested
					responses[2].Answer = data.RankedChoiceNotInterested
					expectedChoices.Choices = []solvercontract.RankedChoice{
						{OfferingID: string(offerings[0].ID), Response: solvercontract.RankedResponse, Rank: 1},
						{OfferingID: string(offerings[1].ID), Response: solvercontract.InterestedResponse},
						{OfferingID: string(offerings[2].ID), Response: solvercontract.NotInterestedResponse},
					}
				}
				_, err = factory.SubmitRankedChoices(ctx, preference.RankedChoiceSubmissionInput{
					SchoolYearID: year.ID, ProgramID: programRow.ID, SessionID: session.ID, StudentID: student.ID,
					Channel: data.PreferenceChannelAdministratorOnBehalf, Responses: responses,
				})
				require.NoError(t, err)
				_, storedResponses, err := preferences.LatestRankedChoices(ctx, string(organizationID), year.ID, programRow.ID, session.ID, student.ID)
				require.NoError(t, err)
				require.Len(t, storedResponses, len(offerings))
				storedAnswers := make(map[ids.XID]data.RankedChoiceAnswer, len(storedResponses))
				for _, response := range storedResponses {
					storedAnswers[response.OfferingID] = response.Answer
				}
				require.Equal(t, data.RankedChoiceNoResponse, storedAnswers[offerings[3].ID])
				require.Equal(t, data.RankedChoiceNoResponse, storedAnswers[offerings[4].ID], "grade-hidden offering is stored as unanswered, not omitted")
			}
			_, err = factory.TransitionSession(ctx, year.ID, programRow.ID, session.ID, data.SessionVotingClosed, false, "", nil)
			require.NoError(t, err)
			_, err = factory.TransitionSession(ctx, year.ID, programRow.ID, session.ID, data.SessionAssigning, false, "", nil)
			require.NoError(t, err)

			seed := int64(325)
			response := solvercontract.Response{Version: solvercontract.Version, Seed: seed, Status: "optimal", Assignments: []solvercontract.Assignment{{ParticipantID: string(student.ID), OfferingID: string(offerings[0].ID), RealizedQuality: test.quality}}}
			client := &snapshotCheckingResponseClient{t: t, responseClient: responseClient{response: response}}
			service := New(harness.Database, client)
			snapshot, err := service.CompileSnapshot(ctx, string(organizationID), year.ID, programRow.ID, session.ID)
			require.NoError(t, err)
			require.Len(t, snapshot.Request.Participants, 1)
			participant := snapshot.Request.Participants[0]
			require.Equal(t, string(student.ID), participant.ID)
			require.Equal(t, []solvercontract.InterestRating{{InterestAreaID: string(area.ID), Rating: solvercontract.VeryInterestedRating}}, participant.InterestProfile)
			if expectedChoices == nil {
				require.Nil(t, participant.RankedChoices)
			} else {
				require.NotNil(t, participant.RankedChoices)
				require.NotNil(t, participant.RankedChoices.Choices)
				require.ElementsMatch(t, expectedChoices.Choices, participant.RankedChoices.Choices)
			}
			snapshot.Request.Seed = seed
			client.expected, err = solvercontract.CanonicalJSON(snapshot.Request)
			require.NoError(t, err)

			run, err := service.StartAuthoritative(ctx, string(organizationID), actor, year.ID, programRow.ID, session.ID, &seed)
			require.NoError(t, err)
			require.Equal(t, 1, client.calls)
			require.Equal(t, "applied", run.ApplicationStatus)
			loaded, err := service.Get(ctx, string(organizationID), year.ID, programRow.ID, session.ID, run.ID)
			require.NoError(t, err)
			var recorded solvercontract.Request
			require.NoError(t, json.Unmarshal(loaded.RequestDocument, &recorded))
			canonical, err := solvercontract.CanonicalJSON(recorded)
			require.NoError(t, err)
			require.Equal(t, client.expected, canonical)
			fingerprint, err := solvercontract.Fingerprint(recorded)
			require.NoError(t, err)
			require.Equal(t, fingerprint, loaded.InputFingerprint)
			draft := listDraft(t, harness.Database, organizationID, year.ID, programRow.ID, session.ID)
			require.Len(t, draft, 1)
			require.Equal(t, student.ID, draft[0].StudentID)
			require.Equal(t, offerings[0].ID, draft[0].OfferingID)
			require.Equal(t, &run.ID, draft[0].SolveRunID)
			require.Equal(t, test.quality, draft[0].RealizedQuality)
		})
	}
}

type snapshotCheckingResponseClient struct {
	responseClient
	t        *testing.T
	expected []byte
	calls    int
}

func (c *snapshotCheckingResponseClient) Solve(ctx context.Context, request solvercontract.Request) (solvercontract.Response, error) {
	canonical, err := solvercontract.CanonicalJSON(request)
	require.NoError(c.t, err)
	require.Equal(c.t, c.expected, canonical)
	c.calls++
	return c.responseClient.Solve(ctx, request)
}

func snapshotPreferenceAdminActor(t *testing.T, harness *testharness.Harness, organizationID ids.XID) audit.Actor {
	t.Helper()
	var userID ids.XID
	require.NoError(t, harness.Migrator.QueryRow(harness.Context, `insert into users (provider_subject, email) values ($1, $2) returning id`, "snapshot-admin-"+string(organizationID), "synthetic-snapshot-"+string(organizationID)+"@example.test").Scan(&userID))
	_, err := harness.Migrator.Exec(harness.Context, `insert into organization_members (organization_id, user_id, role) values ($1, $2, 'administrator')`, organizationID, userID)
	require.NoError(t, err)
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "synthetic-snapshot-" + string(organizationID) + "@example.test"}
}
