package handlers

import (
	"testing"

	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/stretchr/testify/require"
)

func TestPreferenceFormResponseHidesRosterNameFromStudentCodeRespondents(t *testing.T) {
	form := preference.PreferenceForm{
		Type:         preference.FormTypeInterestProfile,
		ID:           ids.XID("survey-1"),
		SchoolYearID: ids.XID("year-1"),
		ProgramID:    ids.XID("program-1"),
		StudentID:    ids.XID("student-1"),
		StudentName:  "Synthetic Student",
		Name:         "Synthetic Interest Form",
	}

	studentResponse := preferenceFormResponse(form, false)
	require.Equal(t, "student-1", studentResponse.StudentID)
	require.Empty(t, studentResponse.StudentName)

	adminResponse := preferenceFormResponse(form, true)
	require.Equal(t, "Synthetic Student", adminResponse.StudentName)
}
