package people

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestApplyAdultUpdatePreservesOmittedFieldsAndClearsEmptyOptionalValues(t *testing.T) {
	preferred := "Alex"
	email := "alex@example.test"
	current := data.Adult{
		LegalGivenName: "Alexander", LegalFamilyName: "Rivera",
		PreferredGivenName: &preferred, Email: &email,
		ParticipationIntent: data.AdultParticipationHelp,
	}
	newEmail := ""
	newIntent := data.AdultParticipationLead
	emailPtr := &newEmail
	updated, changed := applyAdultUpdate(current, AdultUpdateInput{
		Email: &emailPtr, ParticipationIntent: &newIntent,
	})
	require.True(t, changed)
	require.Equal(t, "Alexander", updated.LegalGivenName)
	require.Equal(t, "Alex", *updated.PreferredGivenName)
	require.Equal(t, "", *updated.Email)
	require.Equal(t, data.AdultParticipationLead, updated.ParticipationIntent)
}

func TestApplyAdultUpdateRejectsNoChanges(t *testing.T) {
	current := data.Adult{LegalGivenName: "Alex", LegalFamilyName: "Rivera", ParticipationIntent: data.AdultParticipationHelp}
	_, changed := applyAdultUpdate(current, AdultUpdateInput{})
	require.False(t, changed)
}
