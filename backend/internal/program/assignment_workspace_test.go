package program

import (
	"testing"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestActiveAssignmentRosterExcludesDeletedAndMissingStudents(t *testing.T) {
	deletedAt := time.Now()
	memberships := []data.ProgramMembership{
		{ID: "membership-active", StudentID: "active"},
		{ID: "membership-deleted", StudentID: "deleted"},
		{ID: "membership-missing", StudentID: "missing"},
		{ID: "membership-unplaced", StudentID: "unplaced"},
	}
	assignments := []data.Assignment{
		{ID: "assignment-active", StudentID: "active"},
		{ID: "assignment-deleted", StudentID: "deleted"},
		{ID: "assignment-missing", StudentID: "missing"},
		{ID: "assignment-nonparticipant", StudentID: "nonparticipant"},
	}
	students := []data.Student{{ID: "active"}, {ID: "deleted", DeletedAt: &deletedAt}, {ID: "unplaced"}, {ID: "nonparticipant"}}
	participants, placements := activeAssignmentRoster(memberships, assignments, students)
	require.Equal(t, []data.ProgramMembership{memberships[0], memberships[3]}, participants)
	require.Equal(t, []data.Assignment{assignments[0], assignments[3]}, placements)
	// The active-only database list omits the deleted student entirely.
	participants, placements = activeAssignmentRoster(memberships, assignments, []data.Student{students[0], students[2], students[3]})
	require.Equal(t, []data.ProgramMembership{memberships[0], memberships[3]}, participants)
	require.Equal(t, []data.Assignment{assignments[0], assignments[3]}, placements)
	require.Len(t, memberships, 4)
	require.Len(t, assignments, 4)
	participants, placements = activeAssignmentRoster(nil, nil, nil)
	require.NotNil(t, participants)
	require.Empty(t, participants)
	require.NotNil(t, placements)
	require.Empty(t, placements)
}

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
