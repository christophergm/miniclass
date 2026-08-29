package people

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/stretchr/testify/require"
)

func TestApplyStudentUpdatePreservesOmittedFieldsAndClearsOptionalValues(t *testing.T) {
	preferred := "Alex"
	external := "student-1"
	prior := ids.XID("prior-student")
	current := data.Student{
		LegalGivenName: "Alexander", LegalFamilyName: "Rivera", PreferredGivenName: &preferred,
		GradeLevelID: xidPointer("grade-1"), HomeroomID: "room-a", ExternalIdentifier: &external, PriorYearStudentID: &prior,
	}
	var clearPreferred *string
	var clearExternal *string
	var clearPrior *ids.XID
	updated, changed := applyStudentUpdate(current, StudentUpdateInput{
		PreferredGivenName: &clearPreferred, ExternalIdentifier: &clearExternal, PriorYearStudentID: &clearPrior,
	})
	require.True(t, changed)
	require.Equal(t, "Alexander", updated.LegalGivenName)
	require.Nil(t, updated.PreferredGivenName)
	require.Equal(t, ids.XID("grade-1"), *updated.GradeLevelID)
	require.Equal(t, ids.XID("room-a"), updated.HomeroomID)
	require.Nil(t, updated.ExternalIdentifier)
	require.Nil(t, updated.PriorYearStudentID)
}

func TestApplyStudentUpdateRejectsNoChanges(t *testing.T) {
	current := data.Student{LegalGivenName: "Alex", LegalFamilyName: "Rivera", GradeLevelID: xidPointer("grade-1"), HomeroomID: "room-a"}
	_, changed := applyStudentUpdate(current, StudentUpdateInput{})
	require.False(t, changed)
}

func xidPointer(value ids.XID) *ids.XID { return &value }
