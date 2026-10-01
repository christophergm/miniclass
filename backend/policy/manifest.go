// Package policy exposes the versioned legal policy bundled into each build.
//
// The JSON manifest is deliberately kept beside this package: go:embed cannot
// include a parent directory, and frontend builds can import this tracked file
// directly without a runtime metadata endpoint.
package policy

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed manifest.json
var manifestJSON []byte

type LegalDocument struct {
	Title         string `json:"title"`
	Version       string `json:"version"`
	EffectiveDate string `json:"effective_date"`
	Notice        string `json:"notice"`
}

type GuardianOnboarding struct {
	Terms   LegalDocument `json:"terms"`
	Privacy LegalDocument `json:"privacy"`
}

type Manifest struct {
	GuardianOnboarding GuardianOnboarding `json:"guardian_onboarding"`
}

// Current is the canonical policy manifest compiled into the backend.
var Current = load()

func load() Manifest {
	var manifest Manifest
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		panic(fmt.Sprintf("parse embedded policy manifest: %v", err))
	}
	validate("guardian_onboarding.terms", manifest.GuardianOnboarding.Terms)
	validate("guardian_onboarding.privacy", manifest.GuardianOnboarding.Privacy)
	return manifest
}

func validate(name string, document LegalDocument) {
	if strings.TrimSpace(document.Title) == "" || strings.TrimSpace(document.Version) == "" || strings.TrimSpace(document.EffectiveDate) == "" || strings.TrimSpace(document.Notice) == "" {
		panic(fmt.Sprintf("embedded policy manifest: %s is incomplete", name))
	}
	if _, err := time.Parse(time.DateOnly, document.EffectiveDate); err != nil {
		panic(fmt.Sprintf("embedded policy manifest: %s has invalid effective_date: %v", name, err))
	}
}
