package people

import "strings"

// DisplayName composes the canonical person name. A preferred given name is
// used wherever a person is named, while the legal family name remains the
// family-name component.
func DisplayName(preferredGivenName, legalGivenName, legalFamilyName *string) string {
	given := ""
	if preferredGivenName != nil {
		given = strings.TrimSpace(*preferredGivenName)
	}
	if given == "" && legalGivenName != nil {
		given = strings.TrimSpace(*legalGivenName)
	}
	family := ""
	if legalFamilyName != nil {
		family = strings.TrimSpace(*legalFamilyName)
	}
	return strings.TrimSpace(strings.Join([]string{given, family}, " "))
}
