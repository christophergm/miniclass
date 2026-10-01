package policy

import "testing"

func TestCurrentPreservesExistingGuardianLegalVersions(t *testing.T) {
	if Current.GuardianOnboarding.Terms.Version != "terms-v1" {
		t.Fatalf("terms version = %q, want terms-v1", Current.GuardianOnboarding.Terms.Version)
	}
	if Current.GuardianOnboarding.Privacy.Version != "privacy-v1" {
		t.Fatalf("privacy version = %q, want privacy-v1", Current.GuardianOnboarding.Privacy.Version)
	}
	for name, document := range map[string]LegalDocument{
		"terms":   Current.GuardianOnboarding.Terms,
		"privacy": Current.GuardianOnboarding.Privacy,
	} {
		if document.EffectiveDate == "" {
			t.Errorf("%s effective date is empty", name)
		}
		if document.Notice == "" {
			t.Errorf("%s notice is empty", name)
		}
	}
}
