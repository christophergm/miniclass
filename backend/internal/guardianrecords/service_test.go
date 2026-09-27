package guardianrecords

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/stretchr/testify/require"
)

func TestNormalizeFoldsUnicodeAndNamePunctuation(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		want  string
	}{
		{name: "case and whitespace", input: "  Casey   ONE ", want: "casey one"},
		{name: "diacritics", input: "Zoë Díaz", want: "zoe diaz"},
		{name: "apostrophe", input: "O’Connor", want: "oconnor"},
		{name: "hyphen", input: "Mary—Jane", want: "maryjane"},
		{name: "punctuation leaves whitespace", input: "Mary - Jane", want: "mary jane"},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, normalize(test.input))
		})
	}
}

func TestPlaceholderStudentsAreNeverCandidates(t *testing.T) {
	for _, name := range []struct{ given, family string }{
		{given: "Placeholder", family: "Student"},
		{given: "Unknown", family: "Child"},
		{given: "N/A", family: "Student"},
	} {
		require.True(t, isPlaceholder(data.Student{LegalGivenName: name.given, LegalFamilyName: name.family}))
	}
	require.True(t, isPlaceholder(data.Student{LegalGivenName: "Alex", LegalFamilyName: "Rivera", IsPlaceholder: true}))
	require.False(t, isPlaceholder(data.Student{LegalGivenName: "Casey", LegalFamilyName: "Synthetic"}))
}
