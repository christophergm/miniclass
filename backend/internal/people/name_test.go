package people

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisplayNamePrefersPreferredGivenName(t *testing.T) {
	preferred := "Alex"
	legal := "Alexander"
	family := "Rivera"
	require.Equal(t, "Alex Rivera", DisplayName(&preferred, &legal, &family))
}

func TestDisplayNameFallsBackToLegalGivenName(t *testing.T) {
	legal := "Alex"
	family := "Rivera"
	require.Equal(t, "Alex Rivera", DisplayName(nil, &legal, &family))
}

func TestDisplayNameTrimsEmptyComponents(t *testing.T) {
	preferred := " "
	legal := " Alex "
	family := " Rivera "
	require.Equal(t, "Alex Rivera", DisplayName(&preferred, &legal, &family))
}
