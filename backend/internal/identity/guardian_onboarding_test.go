package identity

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeGuardianEmailCanonicalizesAddressWithoutAcceptingDisplayNames(t *testing.T) {
	canonical, err := normalizeGuardianEmail("  Guardian@Example.TEST ")
	require.NoError(t, err)
	require.Equal(t, "guardian@example.test", canonical)

	for _, value := range []string{"Guardian Example <guardian@example.test>", "not-an-email", "guardian@example.test\nother@example.test"} {
		_, err := normalizeGuardianEmail(value)
		require.Error(t, err, value)
	}
}

func TestParseGuardianInvitationCSVReportsRowsAndEnforcesLimit(t *testing.T) {
	rows, err := parseGuardianInvitationCSV([]byte("email,name\nGuardian@Example.TEST,One\n,Two\nsecond@example.test,Three\n"))
	require.NoError(t, err)
	require.Equal(t, []string{"Guardian@Example.TEST", "", "second@example.test"}, rows)

	var oversized strings.Builder
	oversized.WriteString("email\n")
	for index := 0; index < guardianInvitationImportLimit+1; index++ {
		oversized.WriteString("x")
		if index < guardianInvitationImportLimit {
			oversized.WriteString("@example.test")
		}
		oversized.WriteByte('\n')
	}
	_, err = parseGuardianInvitationCSV([]byte(oversized.String()))
	require.Error(t, err)
}

func TestParseGuardianInvitationCSVRequiresEmailColumn(t *testing.T) {
	_, err := parseGuardianInvitationCSV([]byte("name\nGuardian"))
	require.EqualError(t, err, "invitation CSV must contain an email column")
}
