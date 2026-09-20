package guardianrecords

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCollapsesCaseAndWhitespace(t *testing.T) {
	require.Equal(t, "casey one", normalize("  Casey   ONE "))
}

func TestPlaceholderStudentsAreNeverCandidates(t *testing.T) {
	for _, name := range []struct{ given, family string }{
		{given: "Placeholder", family: "Student"},
		{given: "Unknown", family: "Child"},
		{given: "N/A", family: "Student"},
	} {
		require.True(t, isPlaceholder(data.Student{LegalGivenName: name.given, LegalFamilyName: name.family}))
	}
	require.False(t, isPlaceholder(data.Student{LegalGivenName: "Casey", LegalFamilyName: "Synthetic"}))
}
