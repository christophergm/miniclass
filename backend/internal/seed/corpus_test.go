package seed

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestGenerateIsDeterministicAndMatchesAppendixDistribution(t *testing.T) {
	first := Generate()
	second := Generate()

	require.NoError(t, first.Validate())
	require.Equal(t, first, second)
	require.Equal(t, []int{20, 27, 22, 21, 30, 19}, gradeDistribution(first))
	require.Equal(t, map[data.AdultParticipationIntent]int{
		data.AdultParticipationLead: 13, data.AdultParticipationHelp: 45, data.AdultParticipationUnavailable: 44,
	}, participationDistribution(first))
}

func gradeDistribution(c Corpus) []int {
	result := make([]int, len(c.Grades))
	for _, student := range c.Students {
		result[student.Grade-1]++
	}
	return result
}

func participationDistribution(c Corpus) map[data.AdultParticipationIntent]int {
	result := make(map[data.AdultParticipationIntent]int)
	for _, adult := range c.Adults {
		result[adult.ParticipationIntent]++
	}
	return result
}
