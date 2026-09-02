package auth

import (
	"context"
	"errors"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/ids"
)

const (
	ModeGuardian        = "guardian"
	ModeAdministration  = "administration"
	GuardianOTPAttempts = 5
)

var (
	ErrOTPInvalid              = errors.New("adult OTP is invalid or expired")
	ErrOTPUnavailable          = errors.New("adult OTP delivery is unavailable")
	ErrAdultEmailAmbiguous     = errors.New("adult email is ambiguous")
	ErrMFAAlreadyEnrolled      = errors.New("MFA is already enrolled")
	ErrMFANotEnrolled          = errors.New("MFA is not enrolled")
	ErrMFAInvalid              = errors.New("MFA proof is invalid")
	ErrMFAResetReasonRequired  = errors.New("MFA reset reason is required")
	ErrAdultAccountLinkMissing = errors.New("adult account link is missing")
)

// OTPDelivery is the transactional authentication-email boundary. The raw
// code exists only in this call and is never passed to a persistence type.
type OTPDelivery interface {
	DeliverOTP(context.Context, string, string) error
}

type OTPRequest struct {
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	Email          string
	Now            time.Time
}

type OTPRequestResult struct {
	Accepted    bool
	ChallengeID string
}

type OTPVerification struct {
	ChallengeID string
	Code        string
	Now         time.Time
}

type GuardianSession struct {
	Bearer         string
	SessionID      ids.XID
	AdultID        ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ExpiresAt      time.Time
	IdleExpiresAt  time.Time
	StudentIDs     []ids.XID
}

type MFAEnrollment struct {
	Secret        string
	RecoveryCodes []string
	Generation    int
}

type MFAVerification struct {
	UserID         ids.XID
	OrganizationID ids.XID
	Code           string
	RecoveryCode   string
	Now            time.Time
}

type AdministrativeSession struct {
	Bearer        string
	SessionID     ids.XID
	UserID        ids.XID
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
}

type MFAReset struct {
	OrganizationID ids.XID
	Actor          audit.Actor
	TargetUserID   ids.XID
	Reason         string
	Now            time.Time
}

type AdultAccountLinkInput struct {
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	AdultID        ids.XID
	UserID         ids.XID
	Actor          audit.Actor
}

type AdultAccountLink struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	AdultID        ids.XID
	UserID         ids.XID
}

// AdultAuthentication contains the application-owned Phase 4 use cases.
type AdultAuthentication interface {
	RequestAdultOTP(context.Context, OTPRequest) (OTPRequestResult, error)
	VerifyAdultOTP(context.Context, OTPVerification) (GuardianSession, error)
	CreateGuardianSessionForAccount(context.Context, ids.XID, ids.XID, ids.XID, time.Time) (GuardianSession, error)
	VerifyMFAForGuardian(context.Context, GuardianPrincipal, string, string, time.Time) (AdministrativeSession, error)
	ResolveSession(context.Context, string) (Principal, error)
	RevokeSession(context.Context, string) error
	RevokeSessionByID(context.Context, ids.XID) error
	EnrollMFA(context.Context, ids.XID, ids.XID, audit.Actor, time.Time) (MFAEnrollment, error)
	VerifyMFA(context.Context, MFAVerification) (AdministrativeSession, error)
	ResetMFA(context.Context, MFAReset) error
	CreateAdultAccountLink(context.Context, AdultAccountLinkInput) (AdultAccountLink, error)
	ListAdultAccountLinks(context.Context, ids.XID, ids.XID) ([]AdultAccountLink, error)
	DeleteAdultAccountLink(context.Context, AdultAccountLinkInput, ids.XID) error
}
