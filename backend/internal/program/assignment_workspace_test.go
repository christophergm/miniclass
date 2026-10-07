package program

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestAssignmentParticipantsUsesPersistedRosterAndVocabularyContext(t *testing.T) {
	preferred := " Synthetic Preferred "
	blankPreferred := "  "
	retiredAt := time.Now()
	memberships := []data.ProgramMembership{
		{ID: "membership-a", StudentID: "student-a", LegalGivenName: "Synthetic Legal", LegalFamilyName: " Student ", GradeLevelID: xid("grade-a")},
		{ID: "membership-b", StudentID: "student-b", LegalGivenName: " Synthetic Second ", LegalFamilyName: " Student ", GradeLevelID: xid("grade-b")},
		{ID: "membership-c", StudentID: "student-c", LegalGivenName: "Synthetic Third", LegalFamilyName: "Student", GradeMissing: true},
		{ID: "membership-d", StudentID: "student-d", LegalGivenName: "Synthetic Fourth", LegalFamilyName: "Student", GradeLevelID: xid("unknown-grade")},
	}
	result := assignmentParticipants(memberships,
		[]data.Student{
			{ID: "student-b", PreferredGivenName: &blankPreferred, HomeroomID: "room-b"},
			{ID: "student-a", PreferredGivenName: &preferred, HomeroomID: "room-a", DeletedAt: &retiredAt},
			{ID: "student-c", HomeroomID: "room-a"},
			{ID: "student-d", HomeroomID: "unknown-room"},
			{ID: "non-member", PreferredGivenName: &preferred, HomeroomID: "room-a"},
		},
		[]data.GradeLevel{
			{ID: "grade-b", Label: "Synthetic First", Ordinal: 1},
			{ID: "grade-a", Label: "Synthetic Kindergarten", Ordinal: 0, RetiredAt: &retiredAt},
		},
		[]data.Homeroom{{ID: "room-b", Name: "Synthetic Room B"}, {ID: "room-a", Name: "Synthetic Room A", RetiredAt: &retiredAt}},
	)
	require.Len(t, result, len(memberships))
	for i, membership := range memberships {
		require.Equal(t, membership, result[i].ProgramMembership)
	}
	require.Equal(t, "Synthetic Preferred Student", result[0].DisplayName)
	require.Equal(t, "Synthetic Kindergarten", result[0].GradeLabel)
	require.NotNil(t, result[0].GradeOrdinal)
	require.Zero(t, *result[0].GradeOrdinal)
	require.Equal(t, "Synthetic Room A", result[0].HomeroomName)
	require.Equal(t, "Synthetic Second Student", result[1].DisplayName)
	require.Equal(t, "Synthetic First", result[1].GradeLabel)
	require.Equal(t, 1, *result[1].GradeOrdinal)
	require.Equal(t, "Synthetic Room B", result[1].HomeroomName)
	require.Equal(t, "Synthetic Third Student", result[2].DisplayName)
	require.Nil(t, result[2].GradeOrdinal)
	require.Empty(t, result[2].GradeLabel)
	require.Equal(t, "Synthetic Room A", result[2].HomeroomName)
	require.Nil(t, result[3].GradeOrdinal)
	require.Empty(t, result[3].GradeLabel)
	require.Empty(t, result[3].HomeroomName)
	require.Empty(t, assignmentParticipants(nil, nil, nil, nil))
	require.NotNil(t, assignmentParticipants(nil, nil, nil, nil))
}
