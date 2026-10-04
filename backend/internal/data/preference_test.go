package data

import (
	"context"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestRetiredPreferenceChannelFailsBeforeDatabaseAccess(t *testing.T) {
	tx := &Tx{actor: audit.Actor{Type: audit.ActorTypeLink, Label: "legacy respondent"}}
	_, _, err := tx.CreateInterestProfileSurveySubmission(context.Background(), "year", "program", "survey", "student", PreferenceChannelStudentCode, nil, nil)
	require.ErrorIs(t, err, ErrPreferenceChannelRetired)
	_, _, err = tx.CreateRankedChoiceSubmission(context.Background(), "year", "program", "session", "student", PreferenceChannelStudentCode, nil, nil)
	require.ErrorIs(t, err, ErrPreferenceChannelRetired)
}

func TestSupportedPreferenceAttributionRemainsStrict(t *testing.T) {
	adultID, userID := ids.XID("adult"), ids.XID("user")
	guardian := audit.Actor{Type: audit.ActorTypeLink, Label: "synthetic guardian"}
	admin := audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: "synthetic administrator"}
	require.NoError(t, validateSubmissionAttribution(guardian, PreferenceChannelGuardian, &adultID))
	require.NoError(t, validateSubmissionAttribution(admin, PreferenceChannelAdministratorOnBehalf, nil))
	require.Error(t, validateSubmissionAttribution(guardian, PreferenceChannelGuardian, nil))
	require.Error(t, validateSubmissionAttribution(admin, PreferenceChannelGuardian, &adultID))
	require.Error(t, validateSubmissionAttribution(guardian, PreferenceChannelAdministratorOnBehalf, nil))
	require.Error(t, validateSubmissionAttribution(admin, PreferenceChannelAdministratorOnBehalf, &adultID))
	require.Error(t, validateSubmissionAttribution(guardian, PreferenceSubmissionChannel("unknown"), &adultID))
}
