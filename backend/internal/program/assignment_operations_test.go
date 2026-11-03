package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestAssignmentInputsUsesEffectiveHighRankMaxForNewManualPlacements(t *testing.T) {
	t.Parallel()

	rank := 4
	rows := map[ids.XID]data.Assignment{
		"new-student": {
			SchoolYearID: "year", ProgramID: "program", SessionID: "session",
			StudentID: "new-student", OfferingID: "offering", Origin: "manual",
		},
		"historical-student": {
			SchoolYearID: "year", ProgramID: "program", SessionID: "session",
			StudentID: "historical-student", OfferingID: "historical-offering", Origin: "solver", RealizedQuality: "high",
		},
	}
	answers := []data.SessionResultAnswer{{StudentID: "new-student", OfferingID: "offering", Answer: data.RankedChoiceRanked, Rank: &rank}}

	inputs := assignmentInputs(rows, answers, 3)
	realized := map[ids.XID]string{}
	for _, input := range inputs {
		realized[input.StudentID] = input.RealizedQuality
	}

	require.Equal(t, "acceptable", realized["new-student"])
	require.Equal(t, "high", realized["historical-student"], "stored realized quality is historical")
}
