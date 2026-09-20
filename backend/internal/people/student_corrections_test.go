package people

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestStudentReviewSignalsExposeRegistrationMatchingPlaceholderAndActivity(t *testing.T) {
	grade := ids.XID("grade-test")
	homeroom := ids.XID("homeroom-test")
	students := []data.Student{
		{ID: "student-placeholder", LegalGivenName: "Unknown", LegalFamilyName: "A", IsPlaceholder: true},
		{ID: "student-one", LegalGivenName: "Alex", LegalFamilyName: "Rivera", GradeLevelID: &grade, HomeroomID: homeroom},
		{ID: "student-two", LegalGivenName: " alex ", LegalFamilyName: "RIVERA", GradeLevelID: &grade, HomeroomID: homeroom},
	}
	email := "guardian@example.test"
	adultWithEmail := data.Adult{ID: "adult-email", Email: &email}
	adultWithoutEmail := data.Adult{ID: "adult-no-email"}
	relationships := []data.GuardianRelationship{
		{StudentID: students[0].ID, AdultID: adultWithEmail.ID},
		{StudentID: students[1].ID, AdultID: adultWithoutEmail.ID},
	}
	signals := studentReviewSignals(students, relationships, []data.Adult{adultWithEmail, adultWithoutEmail}, []data.StudentCorrectionReviewCount{{StudentID: students[1].ID, CorrectionCount: 3}})

	codes := make(map[string]int)
	for _, signal := range signals {
		codes[signal.Code]++
	}
	require.Equal(t, 1, codes["placeholder-student"])
	require.Equal(t, 2, codes["matching-duplicate"])
	require.Equal(t, 1, codes["registration-no-email"])
	require.Equal(t, 1, codes["unusual-activity"])
	require.Equal(t, 1, codes["registration-unlinked"])
}

func TestCorrectionReasonIsRequired(t *testing.T) {
	_, err := correctionReason("  ")
	require.ErrorIs(t, err, ErrCorrectionReasonRequired)
	reason, err := correctionReason(" corrected from signed roster note ")
	require.NoError(t, err)
	require.Equal(t, "corrected from signed roster note", reason)
}
