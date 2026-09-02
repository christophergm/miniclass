package auth

import (
	"sort"

	"github.com/chrismott/miniclass/internal/ids"
)

// Capability is a closed authorization permission. The first eight values
// are the §6.6 matrix; Authenticated is the non-domain gate used by identity
// and operational endpoints; Public is the absence of a gate, stated out loud.
type Capability string

const (
	CapabilityManageAdministrators Capability = "manage_administrators"
	CapabilityDeletePersonalData   Capability = "delete_personal_data"
	CapabilityManageSchoolYear     Capability = "manage_school_year"
	CapabilityManageRoster         Capability = "manage_roster"
	CapabilityManageCatalog        Capability = "manage_catalog"
	CapabilityManageAssignments    Capability = "manage_assignments"
	CapabilityManagePublishing     Capability = "manage_publishing"
	CapabilityReadAuditLog         Capability = "read_audit_log"
	CapabilityGuardianAccess       Capability = "guardian_access"
	CapabilitySession              Capability = "session_authenticated"
	CapabilityAuthenticated        Capability = "authenticated"
	// CapabilityPublic declares that an operation is deliberately
	// unauthenticated. It exists so that "no authentication" is something an
	// operation must say, rather than something it can achieve by saying
	// nothing: the middleware still rejects an operation with no declared
	// capability, and the operation-enumeration test still requires every
	// operation to carry one. It is a property of the operation and never of a
	// principal, so it is absent from the §6.6 matrix and no role grants it.
	CapabilityPublic Capability = "public"
)

// OrganizationRole is the role vocabulary used by the identity schema and
// §6.6. It is kept separate from generated SQL enums.
type OrganizationRole string

const (
	RoleOwner         OrganizationRole = "owner"
	RoleAdministrator OrganizationRole = "administrator"
	RoleCoordinator   OrganizationRole = "coordinator"
)

var roleCapabilities = map[OrganizationRole]map[Capability]bool{
	RoleOwner: {
		CapabilityManageAdministrators: true,
		CapabilityDeletePersonalData:   true,
		CapabilityManageSchoolYear:     true,
		CapabilityManageRoster:         true,
		CapabilityManageCatalog:        true,
		CapabilityManageAssignments:    true,
		CapabilityManagePublishing:     true,
		CapabilityReadAuditLog:         true,
	},
	RoleAdministrator: {
		CapabilityManageSchoolYear:  true,
		CapabilityManageRoster:      true,
		CapabilityManageCatalog:     true,
		CapabilityManageAssignments: true,
		CapabilityManagePublishing:  true,
		CapabilityReadAuditLog:      true,
	},
	RoleCoordinator: {
		CapabilityManageRoster:      true,
		CapabilityManageCatalog:     true,
		CapabilityManageAssignments: true,
	},
}

var matrixCapabilities = []Capability{
	CapabilityManageAdministrators,
	CapabilityDeletePersonalData,
	CapabilityManageSchoolYear,
	CapabilityManageRoster,
	CapabilityManageCatalog,
	CapabilityManageAssignments,
	CapabilityManagePublishing,
	CapabilityReadAuditLog,
}

// CapabilityMatrix returns a stable copy of the §6.6 role/capability table.
func CapabilityMatrix() map[OrganizationRole]map[Capability]bool {
	result := make(map[OrganizationRole]map[Capability]bool, len(roleCapabilities))
	for role, capabilities := range roleCapabilities {
		result[role] = make(map[Capability]bool, len(matrixCapabilities))
		for _, capability := range matrixCapabilities {
			result[role][capability] = capabilities[capability]
		}
	}
	return result
}

// MatrixRoles and MatrixCapabilities expose deterministic vocabulary for
// table-driven completeness tests and contract tooling.
func MatrixRoles() []OrganizationRole {
	return []OrganizationRole{RoleOwner, RoleAdministrator, RoleCoordinator}
}

func MatrixCapabilities() []Capability {
	return append([]Capability(nil), matrixCapabilities...)
}

// HasRoleCapability answers one closed-matrix cell. Unknown roles and
// capabilities are default-deny.
func HasRoleCapability(role OrganizationRole, capability Capability) bool {
	if capability == CapabilityPublic {
		// Denied for every role on purpose. Public is an operation's declaration
		// that it needs no principal, so granting it to a principal would turn a
		// route annotation into an authorization grant.
		return false
	}
	if capability == CapabilityAuthenticated {
		return role == RoleOwner || role == RoleAdministrator || role == RoleCoordinator
	}
	return roleCapabilities[role][capability]
}

// RequiresMFA identifies capabilities that expose administrative data or
// mutations. Provider identity may begin MFA enrollment, but an
// application-issued session with a fresh MFA proof is required for these
// capabilities.
func RequiresMFA(capability Capability) bool {
	switch capability {
	case CapabilityManageAdministrators, CapabilityDeletePersonalData, CapabilityManageSchoolYear,
		CapabilityManageRoster, CapabilityManageCatalog, CapabilityManageAssignments,
		CapabilityManagePublishing, CapabilityReadAuditLog:
		return true
	default:
		return false
	}
}

// Principal is the common authorization surface for account and future link
// principals. Callers ask for capabilities rather than inspecting roles.
type Principal interface {
	HasCapability(Capability) bool
}

// Account is the small identity-domain contract exposed to auth and API
// callers. The unscoped SQL accessor is adapted to this type by
// internal/identity and does not cross the package boundary.
type Account struct {
	User       AccountUser
	Membership AccountMembership
}

type AccountUser struct {
	ID              ids.XID
	ProviderSubject string
	Email           string
}

type AccountMembership struct {
	ID               ids.XID
	OrganizationID   ids.XID
	OrganizationName string
	Role             string
}

// AccountPrincipal is the Phase 1 administrator principal resolved from a
// verified provider subject and exactly one application membership.
type AccountPrincipal struct {
	UserID           ids.XID
	Subject          string
	Email            string
	OrganizationID   ids.XID
	Organization     string
	Role             OrganizationRole
	MFAAuthenticated bool
	SessionID        ids.XID
}

func (p AccountPrincipal) HasCapability(capability Capability) bool {
	if capability == CapabilitySession {
		return true
	}
	return HasRoleCapability(p.Role, capability)
}

func (p AccountPrincipal) PrincipalID() ids.XID         { return p.UserID }
func (p AccountPrincipal) ProviderSubject() string      { return p.Subject }
func (p AccountPrincipal) EmailAddress() string         { return p.Email }
func (p AccountPrincipal) OrganizationName() string     { return p.Organization }
func (p AccountPrincipal) OrganizationIDValue() ids.XID { return p.OrganizationID }
func (p AccountPrincipal) RoleName() OrganizationRole   { return p.Role }

// GuardianPrincipal is the narrow principal created by an adult OTP. Its
// student scope is refreshed from live relationships by the session resolver.
type GuardianPrincipal struct {
	AdultID        ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	SessionID      ids.XID
	Email          string
	StudentIDs     []ids.XID
}

func (p GuardianPrincipal) HasCapability(capability Capability) bool {
	return capability == CapabilityGuardianAccess || capability == CapabilitySession
}

// MatrixCells returns every cell in stable order, useful for exhaustive tests.
func MatrixCells() []struct {
	Role       OrganizationRole
	Capability Capability
	Allowed    bool
} {
	result := make([]struct {
		Role       OrganizationRole
		Capability Capability
		Allowed    bool
	}, 0, len(MatrixRoles())*len(MatrixCapabilities()))
	roles := MatrixRoles()
	capabilities := MatrixCapabilities()
	for _, role := range roles {
		for _, capability := range capabilities {
			result = append(result, struct {
				Role       OrganizationRole
				Capability Capability
				Allowed    bool
			}{Role: role, Capability: capability, Allowed: HasRoleCapability(role, capability)})
		}
	}
	return result
}

// SortCapabilities is used by diagnostics that report a principal's grants.
func SortCapabilities(capabilities []Capability) []Capability {
	result := append([]Capability(nil), capabilities...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
