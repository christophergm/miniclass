package api

import (
	"encoding/json"
	"testing"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/stretchr/testify/require"
)

// SPEC §20.1. The audit vocabulary is Go-owned and grows with features, so the
// published enum has to be derived from it. It was previously restated as a
// struct tag, and school_year_state_transition was recorded, returned by the
// endpoint, and absent from the contract the frontend generates its types
// from. This test fails if the enum is ever hand-written again.
func TestAuditLogActionEnumMatchesTheAuditVocabulary(t *testing.T) {
	document := NewOpenAPI(RouterOptions{})
	encoded, err := json.Marshal(document)
	require.NoError(t, err)
	var raw struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Enum []string `json:"enum"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	require.NoError(t, json.Unmarshal(encoded, &raw))

	entry, ok := raw.Components.Schemas["AuditLogEntry"]
	require.True(t, ok, "AuditLogEntry is not published in components.schemas")
	action, ok := entry.Properties["action"]
	require.True(t, ok, "AuditLogEntry publishes no action property")

	published := make([]string, 0, len(audit.Actions()))
	for _, value := range audit.Actions() {
		published = append(published, string(value))
	}
	require.Equal(t, published, action.Enum,
		"the published action enum has drifted from audit.Actions")
	require.Contains(t, action.Enum, "school_year_state_transition",
		"the action the school-year lifecycle records is missing from the contract")
}
