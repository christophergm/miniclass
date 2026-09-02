package auth

import "testing"

func TestPublicCapabilityIsNeitherGrantedNorPartOfTheMatrix(t *testing.T) {
	for _, role := range MatrixRoles() {
		if HasRoleCapability(role, CapabilityPublic) {
			t.Fatalf("role %q was granted the public capability", role)
		}
	}
	for _, capability := range MatrixCapabilities() {
		if capability == CapabilityPublic {
			t.Fatal("the public capability is part of the §6.6 matrix")
		}
	}
	if HasRoleCapability(OrganizationRole("unknown"), CapabilityPublic) {
		t.Fatal("an unknown role was granted the public capability")
	}
}

func TestRequiresMFAForAdministrativeCapabilities(t *testing.T) {
	for _, capability := range MatrixCapabilities() {
		if !RequiresMFA(capability) {
			t.Fatalf("administrative capability %q does not require MFA", capability)
		}
	}
	for _, capability := range []Capability{CapabilityAuthenticated, CapabilityGuardianAccess, CapabilitySession, CapabilityPublic} {
		if RequiresMFA(capability) {
			t.Fatalf("non-administrative capability %q unexpectedly requires MFA", capability)
		}
	}
}
