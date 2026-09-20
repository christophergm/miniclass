package handlers

import (
	"encoding/json"
	"testing"

	"github.com/chrismott/miniclass/internal/guardianrecords"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestGuardianCandidateResponseUsesMinimalDisclosureFields(t *testing.T) {
	preferred := "CJ"
	gradeID := ids.XID("grade-private")
	row := guardianrecords.Student{
		ID:                 ids.XID("student-selection-handle"),
		LegalGivenName:     "Casey",
		LegalFamilyName:    "Synthetic",
		PreferredGivenName: &preferred,
		GradeLevelID:       &gradeID,
		GradeLabel:         "Synthetic Grade",
		HomeroomID:         ids.XID("homeroom-private"),
		HomeroomLabel:      "Synthetic Room",
	}

	encoded, err := json.Marshal(guardianCandidateResponses([]guardianrecords.Student{row}))
	require.NoError(t, err)
	var fields []map[string]any
	require.NoError(t, json.Unmarshal(encoded, &fields))
	require.Len(t, fields, 1)
	require.Equal(t, "student-selection-handle", fields[0]["id"])
	require.Equal(t, "Casey", fields[0]["legal_given_name"])
	require.Equal(t, "Synthetic", fields[0]["legal_family_name"])
	require.Equal(t, "Synthetic Grade", fields[0]["grade_label"])
	require.Equal(t, "Synthetic Room", fields[0]["homeroom_label"])
	require.NotContains(t, fields[0], "preferred_given_name")
	require.NotContains(t, fields[0], "grade_level_id")
	require.NotContains(t, fields[0], "homeroom_id")
	require.NotContains(t, fields[0], "external_identifier")
	require.NotContains(t, fields[0], "guardian")
}
