package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type contextKey string

const (
	// RequiredCapabilityExtension is the Huma/OpenAPI operation metadata key.
	RequiredCapabilityExtension = "x-required-capability"
	// AllowUnresolvedPrincipalExtension marks the invitation-claim exception.
	AllowUnresolvedPrincipalExtension            = "x-allow-unresolved-principal"
	identityContextKey                contextKey = "miniclass.auth.identity"
	principalContextKey               contextKey = "miniclass.auth.principal"
)

var (
	ErrNoOrganization        = errors.New("account has no organization membership")
	ErrMultipleOrganizations = errors.New("account has multiple organization memberships")
	ErrInvitationInvalid     = errors.New("admin invitation is invalid")
	ErrInvitationEmail       = errors.New("admin invitation email does not match")
	ErrInvitationUnverified  = errors.New("invitation claim requires a verified email")
)

// AccountResolver maps an already verified provider subject to local
// membership. It must not trust organization or role claims from the JWT.
type AccountResolver interface {
	ResolveAccount(context.Context, string) (Account, error)
}

type InvitationClaimInput struct {
	Bearer          string
	ProviderSubject string
	Email           string
	EmailVerified   bool
}

type InvitationClaimer interface {
	ClaimAdminInvitation(context.Context, InvitationClaimInput) (Account, error)
}

// Failure is a transport-neutral authentication/authorization failure. The
// API maps Code to its RFC 9457 problem registry.
type Failure struct {
	Status int
	Code   FailureCode
	Detail string
}

type FailureCode string

const (
	FailureAuthenticationRequired FailureCode = "authentication-required"
	FailureInvalidToken           FailureCode = "invalid-token"
	FailureAuthUnavailable        FailureCode = "authentication-unavailable"
	FailureNoOrganization         FailureCode = "no-organization"
	FailureMultipleOrganizations  FailureCode = "multiple-organizations"
	FailureCapabilityRequired     FailureCode = "capability-required"
	FailureMissingCapability      FailureCode = "capability-not-declared"
)

// ErrorWriter writes a failure using the host framework's error format.
type ErrorWriter func(huma.Context, Failure)

// Middleware authenticates, resolves the tenant principal, and enforces the
// capability declared on the matched Huma operation. Huma's operation
// metadata is the source of the requirement, so a route cannot silently omit
// its authorization check.
func Middleware(verifier Verifier, resolver AccountResolver, writeError ErrorWriter) func(huma.Context, func(huma.Context)) {
	if writeError == nil {
		writeError = defaultErrorWriter
	}
	return func(ctx huma.Context, next func(huma.Context)) {
		operation := ctx.Operation()
		if operation == nil {
			writeError(ctx, Failure{Status: http.StatusInternalServerError, Code: FailureAuthUnavailable, Detail: "authentication operation metadata is unavailable"})
			return
		}
		capability, ok := operation.Metadata[RequiredCapabilityExtension].(string)
		if !ok {
			capability, ok = operation.Extensions[RequiredCapabilityExtension].(string)
		}
		if !ok || strings.TrimSpace(capability) == "" {
			writeError(ctx, Failure{Status: http.StatusInternalServerError, Code: FailureMissingCapability, Detail: "required capability is not declared"})
			return
		}

		if verifier == nil || resolver == nil {
			writeError(ctx, Failure{Status: http.StatusInternalServerError, Code: FailureAuthUnavailable, Detail: "authentication is not configured"})
			return
		}
		bearer, err := bearerFromHeader(ctx.Header("Authorization"))
		if err != nil {
			writeError(ctx, Failure{Status: http.StatusUnauthorized, Code: FailureAuthenticationRequired, Detail: "a bearer token is required"})
			return
		}
		verified, err := verifier.Verify(ctx.Context(), bearer)
		if err != nil {
			writeError(ctx, Failure{Status: http.StatusUnauthorized, Code: FailureInvalidToken, Detail: "the bearer token is invalid"})
			return
		}
		withIdentity := huma.WithValue(ctx, identityContextKey, verified)
		account, err := resolver.ResolveAccount(withIdentity.Context(), verified.Subject)
		if err != nil {
			if !errors.Is(err, ErrNoOrganization) && !errors.Is(err, ErrMultipleOrganizations) {
				writeError(withIdentity, Failure{Status: http.StatusInternalServerError, Code: FailureAuthUnavailable, Detail: "unable to resolve the authenticated account"})
				return
			}
			allowUnresolved, _ := operation.Metadata[AllowUnresolvedPrincipalExtension].(bool)
			if !allowUnresolved {
				allowUnresolved, _ = operation.Extensions[AllowUnresolvedPrincipalExtension].(bool)
			}
			if !(allowUnresolved && errors.Is(err, ErrNoOrganization)) {
				failure := Failure{Status: http.StatusForbidden, Code: FailureNoOrganization, Detail: "the verified account has no organization membership"}
				if errors.Is(err, ErrMultipleOrganizations) {
					failure.Status = http.StatusConflict
					failure.Code = FailureMultipleOrganizations
					failure.Detail = "the account has more than one organization membership"
				}
				writeError(withIdentity, failure)
				return
			}
			next(withIdentity)
			return
		}

		principal := AccountPrincipal{
			UserID:         account.User.ID,
			Subject:        account.User.ProviderSubject,
			Email:          account.User.Email,
			OrganizationID: account.Membership.OrganizationID,
			Organization:   account.Membership.OrganizationName,
			Role:           OrganizationRole(account.Membership.Role),
		}
		if !principal.HasCapability(Capability(capability)) {
			writeError(huma.WithValue(withIdentity, principalContextKey, Principal(principal)), Failure{
				Status: http.StatusForbidden,
				Code:   FailureCapabilityRequired,
				Detail: "the principal lacks the required capability",
			})
			return
		}
		next(huma.WithValue(withIdentity, principalContextKey, Principal(principal)))
	}
}

func bearerFromHeader(value string) (string, error) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("authorization header is not a bearer token")
	}
	return parts[1], nil
}

// IdentityFromContext retrieves the verified identity assertion for handlers
// such as invitation claim, which may run before local membership exists.
func IdentityFromContext(ctx context.Context) (VerifiedIdentity, bool) {
	identity, ok := ctx.Value(identityContextKey).(VerifiedIdentity)
	return identity, ok
}

// PrincipalFromContext retrieves the resolved local principal.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

func defaultErrorWriter(ctx huma.Context, failure Failure) {
	ctx.SetHeader("Content-Type", "application/problem+json")
	ctx.SetStatus(failure.Status)
	_ = json.NewEncoder(ctx.BodyWriter()).Encode(map[string]any{
		"type": string(failure.Code), "title": http.StatusText(failure.Status), "status": failure.Status, "detail": failure.Detail,
	})
}
