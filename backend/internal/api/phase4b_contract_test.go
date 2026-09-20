package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/chrismott/miniclass/internal/auth"
	"github.com/stretchr/testify/require"
)

func TestPhase4BOperationsDeclareTheExpectedCapabilities(t *testing.T) {
	encoded, err := json.Marshal(NewOpenAPI(RouterOptions{}))
	require.NoError(t, err)
	var document struct {
		Paths map[string]map[string]map[string]any `json:"paths"`
	}
	require.NoError(t, json.Unmarshal(encoded, &document))

	expected := map[string]string{
		"GET /api/guardian/students":                                              string(auth.CapabilityGuardianAccess),
		"GET /api/guardian/students/candidates":                                   string(auth.CapabilityGuardianAccess),
		"POST /api/guardian/students/candidates":                                  string(auth.CapabilityGuardianAccess),
		"POST /api/guardian/students":                                             string(auth.CapabilityGuardianAccess),
		"PATCH /api/guardian/students/{studentID}":                                string(auth.CapabilityGuardianAccess),
		"DELETE /api/guardian/students/{studentID}":                               string(auth.CapabilityGuardianAccess),
		"PATCH /api/guardian/profile":                                             string(auth.CapabilityGuardianAccess),
		"DELETE /api/guardian/profile":                                            string(auth.CapabilityGuardianAccess),
		"POST /api/school-years/{schoolYearID}/purge":                             string(auth.CapabilityManageSchoolYear),
		"POST /api/school-years/{schoolYearID}/placeholder-students":              string(auth.CapabilityManageRoster),
		"POST /api/school-years/{schoolYearID}/student-reconciliations":           string(auth.CapabilityManageRoster),
		"GET /api/school-years/{schoolYearID}/student-review-signals":             string(auth.CapabilityManageRoster),
		"POST /api/school-years/{schoolYearID}/student-corrections":               string(auth.CapabilityManageRoster),
		"PATCH /api/school-years/{schoolYearID}/student-corrections/{studentID}":  string(auth.CapabilityManageRoster),
		"DELETE /api/school-years/{schoolYearID}/student-corrections/{studentID}": string(auth.CapabilityManageRoster),
	}

	for key, capability := range expected {
		parts := strings.SplitN(key, " ", 2)
		operations, ok := document.Paths[parts[1]]
		require.True(t, ok, "missing Phase 4B route %s", key)
		operation, ok := operations[strings.ToLower(parts[0])]
		require.True(t, ok, "missing Phase 4B method %s", key)
		require.Equal(t, capability, operation[auth.RequiredCapabilityExtension], key)
	}
}
