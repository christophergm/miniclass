package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestSessionLifecycleLegalEdges(t *testing.T) {
	states := []data.SessionState{
		data.SessionPlanning,
		data.SessionCatalogPublished,
		data.SessionVotingOpen,
		data.SessionVotingClosed,
		data.SessionAssigning,
		data.SessionPublished,
		data.SessionComplete,
	}
	legal := map[[2]data.SessionState]bool{
		{data.SessionPlanning, data.SessionCatalogPublished}:   true,
		{data.SessionCatalogPublished, data.SessionVotingOpen}: true,
		{data.SessionCatalogPublished, data.SessionAssigning}:  true,
		{data.SessionVotingOpen, data.SessionVotingClosed}:     true,
		{data.SessionVotingClosed, data.SessionAssigning}:      true,
		{data.SessionVotingClosed, data.SessionVotingOpen}:     true,
		{data.SessionAssigning, data.SessionPublished}:         true,
		{data.SessionAssigning, data.SessionVotingClosed}:      true,
		{data.SessionPublished, data.SessionComplete}:          true,
		{data.SessionPublished, data.SessionAssigning}:         true,
	}

	for _, from := range states {
		for _, to := range states {
			want := legal[[2]data.SessionState{from, to}]
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				require.Equal(t, want, IsLegalSessionTransition(from, to))
			})
		}
	}
}

func TestPlanSessionTransitionTable(t *testing.T) {
	tests := []struct {
		name           string
		from           data.SessionState
		to             data.SessionState
		offeringCount  int
		draftStale     bool
		wantError      error
		backward       bool
		markDraftStale bool
		warningCodes   []string
	}{
		{name: "voting requires a catalog", from: data.SessionCatalogPublished, to: data.SessionVotingOpen, wantError: ErrSessionTransitionGate},
		{name: "voting opens with a catalog", from: data.SessionCatalogPublished, to: data.SessionVotingOpen, offeringCount: 1},
		{name: "stale assignments cannot publish", from: data.SessionAssigning, to: data.SessionPublished, draftStale: true, wantError: ErrSessionTransitionGate},
		{name: "draft assignments publish", from: data.SessionAssigning, to: data.SessionPublished},
		{name: "reopening voting warns", from: data.SessionVotingClosed, to: data.SessionVotingOpen, backward: true, warningCodes: []string{"backward-transition"}},
		{name: "reopening with a draft marks it stale", from: data.SessionAssigning, to: data.SessionVotingClosed, backward: true, markDraftStale: true, warningCodes: []string{"backward-transition", "stale-draft"}},
		{name: "leaving published warns about links", from: data.SessionPublished, to: data.SessionAssigning, backward: true, markDraftStale: true, warningCodes: []string{"backward-transition", "stale-draft", "published-links-invalidated"}},
		{name: "invalid edge", from: data.SessionPlanning, to: data.SessionComplete, wantError: ErrSessionTransitionInvalid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			plan, err := PlanSessionTransition(test.from, test.to, test.offeringCount, test.draftStale)
			if test.wantError != nil {
				require.ErrorIs(t, err, test.wantError)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.backward, plan.Backward)
			require.Equal(t, test.markDraftStale, plan.MarkDraftAssignmentsStale)
			codes := make([]string, 0, len(plan.Warnings))
			for _, warning := range plan.Warnings {
				codes = append(codes, warning.Code)
			}
			require.ElementsMatch(t, test.warningCodes, codes)
		})
	}
}

func TestSessionStateLabelsAreClear(t *testing.T) {
	require.Equal(t, "CatalogPublished", sessionStateLabel(data.SessionCatalogPublished))
	require.Equal(t, "none", sessionStateList(LegalSessionTransitions(data.SessionComplete)))
}
