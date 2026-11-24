package preference

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestBuildResponseTrackingUsesStudentsAsDenominator(t *testing.T) {
	studentTwoPreferred := "Bobby"
	students := []data.ResponseTrackingStudentRow{
		{ID: "student-1", GradeLevelID: xidPtr("grade-1"), GradeLevelLabel: "Grade 1", HomeroomID: "room-a", HomeroomName: "Room A", LegalGivenName: "Ada", LegalFamilyName: "One", Responded: true},
		{ID: "student-2", GradeLevelID: xidPtr("grade-1"), GradeLevelLabel: "Grade 1", HomeroomID: "room-a", HomeroomName: "Room A", LegalGivenName: "Robert", LegalFamilyName: "Two", PreferredGivenName: &studentTwoPreferred},
		{ID: "student-3", GradeLevelID: nil, GradeLevelLabel: "", HomeroomID: "room-b", HomeroomName: "Room B", LegalGivenName: "Cara", LegalFamilyName: "Three"},
	}
	guardianOneEmail := "one@example.test"
	relationships := []data.GuardianRelationship{
		{ID: "relationship-1", AdultID: "adult-1", StudentID: "student-2"},
		{ID: "relationship-2", AdultID: "adult-2", StudentID: "student-2"},
	}
	adults := []data.Adult{
		{ID: "adult-1", LegalGivenName: "Alex", LegalFamilyName: "One", Email: &guardianOneEmail},
		{ID: "adult-2", LegalGivenName: "Blair", LegalFamilyName: "Two"},
	}

	result := buildResponseTracking(ResponseTrackingInterestProfile, "survey-1", "Interests", "year-1", "program-1", students, relationships, adults)

	require.Equal(t, 3, result.TotalStudents)
	require.Equal(t, 1, result.RespondedStudents)
	require.InDelta(t, 33.333333, result.CompletionPercentage, 0.000001)
	require.Len(t, result.NonResponders, 2)
	require.Equal(t, ResponseTrackingGuardianFollowUpStatus, result.NonResponders[0].ContactStatus)
	require.Equal(t, ResponseTrackingUnreachable, result.NonResponders[1].ContactStatus)
	require.Equal(t, "Cara Three", result.NonResponders[1].DisplayName)
	require.Len(t, result.GuardianFollowUp, 2)
	require.ElementsMatch(t, []string{ResponseTrackingGuardianPending, ResponseTrackingGuardianNoEmail}, []string{result.GuardianFollowUp[0].ContactStatus, result.GuardianFollowUp[1].ContactStatus})
	require.Equal(t, 2, result.GradeBreakdown[0].TotalStudents)
	require.Equal(t, 1, result.GradeBreakdown[0].RespondedStudents)
	require.Equal(t, "Unassigned", result.GradeBreakdown[1].Label)
}

func TestBuildResponseTrackingGroupsFollowUpByGuardian(t *testing.T) {
	students := []data.ResponseTrackingStudentRow{
		{ID: "student-1", HomeroomID: "room-a", LegalGivenName: "Zed", LegalFamilyName: "Sibling"},
		{ID: "student-2", HomeroomID: "room-a", LegalGivenName: "Amy", LegalFamilyName: "Sibling"},
		{ID: "student-3", HomeroomID: "room-a", LegalGivenName: "Responded", LegalFamilyName: "Child", Responded: true},
	}
	sharedEmail := "shared@example.test"
	relationships := []data.GuardianRelationship{
		{ID: "relationship-1", AdultID: "adult-b", StudentID: "student-1"},
		{ID: "relationship-2", AdultID: "adult-b", StudentID: "student-2"},
		{ID: "relationship-3", AdultID: "adult-b", StudentID: "student-3"},
		{ID: "relationship-4", AdultID: "adult-a", StudentID: "student-1"},
		{ID: "relationship-5", AdultID: "adult-a", StudentID: "student-1"},
	}
	adults := []data.Adult{
		{ID: "adult-b", LegalGivenName: "Blair", LegalFamilyName: "Sibling", Email: &sharedEmail},
		{ID: "adult-a", LegalGivenName: "Alex", LegalFamilyName: "Sibling"},
	}

	result := buildResponseTracking(ResponseTrackingRankedChoice, "session-1", "Session", "year-1", "program-1", students, relationships, adults)

	require.Equal(t, 3, result.TotalStudents)
	require.Equal(t, 1, result.RespondedStudents)
	require.Len(t, result.GuardianFollowUp, 2)
	require.Equal(t, ids.XID("adult-a"), result.GuardianFollowUp[0].AdultID)
	require.Equal(t, ResponseTrackingGuardianNoEmail, result.GuardianFollowUp[0].ContactStatus)
	require.Equal(t, []ResponseTrackingGuardianStudent{{StudentID: "student-1", StudentName: "Zed Sibling"}}, result.GuardianFollowUp[0].Students)
	require.Equal(t, ids.XID("adult-b"), result.GuardianFollowUp[1].AdultID)
	require.Equal(t, ResponseTrackingGuardianPending, result.GuardianFollowUp[1].ContactStatus)
	require.Equal(t, []ResponseTrackingGuardianStudent{
		{StudentID: "student-2", StudentName: "Amy Sibling"},
		{StudentID: "student-1", StudentName: "Zed Sibling"},
	}, result.GuardianFollowUp[1].Students)
}

func xidPtr(value string) *ids.XID {
	id := ids.XID(value)
	return &id
}
