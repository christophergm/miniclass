package program

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestSessionSummaryIncludesRankedChoiceBeforeAndAfter(t *testing.T) {
	deadline := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	later := deadline.Add(time.Hour)
	before := data.Session{ID: "synthetic-session", State: data.SessionVotingOpen, RankedChoice: &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &deadline}}
	after := before
	after.RankedChoice = &data.RankedChoiceConfiguration{RankDepth: 3, Deadline: &later}
	var summary struct {
		RankedChoice data.RankedChoiceConfiguration `json:"ranked_choice"`
		Before       struct {
			RankedChoice data.RankedChoiceConfiguration `json:"ranked_choice"`
		} `json:"before"`
	}
	require.NoError(t, json.Unmarshal(sessionSummary(&before, after), &summary))
	require.Equal(t, *before.RankedChoice, summary.Before.RankedChoice)
	require.Equal(t, *after.RankedChoice, summary.RankedChoice)
}

func TestValidateRankedChoiceUpdate(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	past, future, later := now.Add(-time.Hour), now.Add(time.Hour), now.Add(2*time.Hour)
	config := func(depth int, deadline *time.Time) *data.RankedChoiceConfiguration {
		return &data.RankedChoiceConfiguration{RankDepth: depth, Deadline: deadline}
	}
	tests := []struct {
		name     string
		state    data.SessionState
		old, new *data.RankedChoiceConfiguration
		want     error
	}{
		{"planning enables voting", data.SessionPlanning, nil, config(3, &future), nil},
		{"before voting changes depth", data.SessionCatalogPublished, config(3, &future), config(4, &later), nil},
		{"before voting replaces expired deadline", data.SessionCatalogPublished, config(3, &past), config(4, &future), nil},
		{"before voting validates depth", data.SessionPlanning, nil, config(0, &future), ErrRankedChoiceRankDepthInvalid},
		{"before voting validates deadline", data.SessionPlanning, nil, config(3, &past), ErrRankedChoiceDeadlineInvalid},
		{"open extends deadline", data.SessionVotingOpen, config(3, &future), config(3, &later), nil},
		{"open shortens deadline", data.SessionVotingOpen, config(3, &later), config(3, &future), nil},
		{"open locks depth", data.SessionVotingOpen, config(3, &future), config(4, &future), ErrRankedChoiceConfigurationLocked},
		{"open cannot enable voting", data.SessionVotingOpen, nil, config(3, &future), ErrRankedChoiceConfigurationLocked},
		{"open requires deadline", data.SessionVotingOpen, config(3, &future), config(3, nil), ErrRankedChoiceDeadlineRequired},
		{"open rejects past replacement", data.SessionVotingOpen, config(3, &future), config(3, &past), ErrRankedChoiceDeadlineInvalid},
		{"open rejects replacement at now", data.SessionVotingOpen, config(3, &future), config(3, &now), ErrRankedChoiceDeadlineInvalid},
		{"expired cannot silently reopen", data.SessionVotingOpen, config(3, &past), config(3, &future), ErrRankedChoiceConfigurationLocked},
		{"deadline at now is expired", data.SessionVotingOpen, config(3, &now), config(3, &future), ErrRankedChoiceConfigurationLocked},
		{"closed cannot edit deadline", data.SessionVotingClosed, config(3, &future), config(3, &later), ErrRankedChoiceConfigurationLocked},
		{"assigning cannot edit deadline", data.SessionAssigning, config(3, &future), config(3, &later), ErrRankedChoiceConfigurationLocked},
		{"published cannot edit deadline", data.SessionPublished, config(3, &future), config(3, &later), ErrRankedChoiceConfigurationLocked},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRankedChoiceUpdate(data.Session{State: test.state, RankedChoice: test.old}, test.new, now)
			if test.want != nil {
				require.ErrorIs(t, err, test.want)
			} else {
				require.NoError(t, err)
			}
		})
	}
	for _, state := range []data.SessionState{data.SessionPlanning, data.SessionCatalogPublished, data.SessionVotingOpen, data.SessionVotingClosed, data.SessionAssigning, data.SessionPublished} {
		t.Run("unchanged expired configuration in "+string(state), func(t *testing.T) {
			require.NoError(t, validateRankedChoiceUpdate(data.Session{State: state, RankedChoice: config(3, &past)}, config(3, &past), now))
		})
	}
}
