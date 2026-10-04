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
	testharness "github.com/chrismott/miniclass/internal/testing"
	"github.com/chrismott/miniclass/internal/testing/factories"
	"github.com/stretchr/testify/require"
)

func TestGuardianPreferencesUseLiveScopeAndInstrumentEligibility(t *testing.T) {
	harness := testharness.Open(t)
	ctx := harness.Context
	organizationID := harness.MintOrganization(t)
	actor := preferenceAdminActor(t, harness, organizationID)
	factory := factories.New(harness.Database, string(organizationID), actor)
	fixture := createPreferenceFixture(t, harness, factory, "guardian live scope")
	surveyID := openPreferenceSurvey(t, harness, organizationID, actor, fixture)
	session, offering := openPreferenceSession(t, harness, factory, fixture)
	adult, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Live Guardian"})
	require.NoError(t, err)
	relationship, err := factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: fixture.student.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)
	other, err := factory.CreateAdult(ctx, fixture.year.ID, people.AdultCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Unrelated Guardian"})
	require.NoError(t, err)
	guardianActor := audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic guardian"}
	service := preference.New(harness.Database)
	interestInput := preference.InterestProfileSurveySubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SurveyID: surveyID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelGuardian, ActorAdultID: &adult.ID, GuardianAdultID: &adult.ID, Answers: []data.InterestProfileAnswer{{InterestAreaID: fixture.area.ID, Rating: data.InterestProfileInterested}}}
	rankedInput := preference.RankedChoiceSubmissionInput{SchoolYearID: fixture.year.ID, ProgramID: fixture.program.ID, SessionID: session.ID, StudentID: fixture.student.ID, Channel: data.PreferenceChannelGuardian, ActorAdultID: &adult.ID, GuardianAdultID: &adult.ID, Responses: []data.RankedChoiceResponseInput{{OfferingID: offering.ID, Answer: data.RankedChoiceInterested}}}
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, interestInput)
	require.NoError(t, err)
	valid, err := service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, rankedInput)
	require.NoError(t, err)

	for _, adultID := range []data.Adult{other} {
		foreignInterest, foreignRanked := interestInput, rankedInput
		foreignInterest.ActorAdultID, foreignInterest.GuardianAdultID = &adultID.ID, &adultID.ID
		foreignRanked.ActorAdultID, foreignRanked.GuardianAdultID = &adultID.ID, &adultID.ID
		_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, foreignInterest)
		require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)
		_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, foreignRanked)
		require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)
	}
	mismatchInterest, mismatchRanked := interestInput, rankedInput
	mismatchInterest.GuardianAdultID, mismatchRanked.GuardianAdultID = &other.ID, &other.ID
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, mismatchInterest)
	require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, mismatchRanked)
	require.ErrorIs(t, err, preference.ErrRankedChoiceGuardianScope)
	foreignOrg := harness.MintOrganization(t)
	_, err = service.SubmitInterestProfileSurvey(ctx, string(foreignOrg), guardianActor, interestInput)
	require.Error(t, err)
	_, err = service.SubmitRankedChoices(ctx, string(foreignOrg), guardianActor, rankedInput)
	require.Error(t, err)

	late, err := factory.CreateStudent(ctx, fixture.year.ID, people.StudentCreateInput{LegalGivenName: "Synthetic", LegalFamilyName: "Late Guardian Student", GradeLevelID: &fixture.grade.ID, HomeroomID: fixture.student.HomeroomID})
	require.NoError(t, err)
	_, err = factory.CreateGuardianRelationship(ctx, fixture.year.ID, people.GuardianRelationshipCreateInput{AdultID: adult.ID, StudentID: late.ID, RelationshipType: data.GuardianRelationshipParent})
	require.NoError(t, err)
	lateInterest, lateRanked := interestInput, rankedInput
	lateInterest.StudentID, lateRanked.StudentID = late.ID, late.ID
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, lateInterest)
	require.ErrorIs(t, err, preference.ErrPreferenceStudentNotProgramMember)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, lateRanked)
	require.ErrorIs(t, err, preference.ErrPreferenceStudentNotProgramMember)
	_, err = factory.AddProgramMembership(ctx, fixture.year.ID, fixture.program.ID, late.ID)
	require.NoError(t, err)
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, lateInterest)
	require.NoError(t, err, "late eligible membership needs no credential issuance")
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, lateRanked)
	require.NoError(t, err)
	_, err = factory.CreateSessionNonParticipation(ctx, fixture.year.ID, fixture.program.ID, session.ID, late.ID, "synthetic exclusion")
	require.NoError(t, err)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, lateRanked)
	require.ErrorIs(t, err, preference.ErrRankedChoiceStudentExcluded)

	explicit, err := service.CreateInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, preference.InterestProfileSurveyInput{Name: "Synthetic Explicit Audience", Audience: preference.InterestProfileSurveyAudienceInput{Type: data.SurveyAudienceExplicitStudents, StudentIDs: []ids.XID{fixture.student.ID}}, Questions: []preference.InterestProfileSurveyQuestionInput{{InterestAreaID: fixture.area.ID}}})
	require.NoError(t, err)
	closingAt := time.Now().UTC().Add(time.Hour)
	_, err = service.TransitionInterestProfileSurvey(ctx, string(organizationID), actor, fixture.year.ID, fixture.program.ID, explicit.Survey.ID, preference.InterestProfileSurveyTransitionInput{State: data.InterestProfileSurveyOpen, ClosingAt: &closingAt})
	require.NoError(t, err)
	lateInterest.SurveyID = explicit.Survey.ID
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, lateInterest)
	require.ErrorIs(t, err, preference.ErrSurveyStudentExcluded)

	require.NoError(t, people.New(harness.Database).DeleteGuardianRelationship(ctx, string(organizationID), fixture.year.ID, relationship.ID, actor))
	_, err = service.SubmitInterestProfileSurvey(ctx, string(organizationID), guardianActor, interestInput)
	require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)
	_, err = service.SubmitRankedChoices(ctx, string(organizationID), guardianActor, rankedInput)
	require.ErrorIs(t, err, preference.ErrPreferenceStudentOutOfScope)
	latest, _, err := service.LatestRankedChoices(ctx, string(organizationID), fixture.year.ID, fixture.program.ID, session.ID, fixture.student.ID)
	require.NoError(t, err)
	require.Equal(t, valid.ID, latest.ID, "failed scope checks cannot replace valid answers")
	err = harness.Database.InTenantRead(ctx, string(organizationID), func(ctx context.Context, tx *data.Tx) error {
		history, err := tx.ListInterestProfileSubmissions(ctx, fixture.year.ID, fixture.program.ID, fixture.student.ID)
		if err != nil {
			return err
		}
		require.Len(t, history, 1, "failed writes must not leave submission history")
		return nil
	})
	require.NoError(t, err)
}
