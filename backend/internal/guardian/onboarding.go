// Package guardian owns the consent-first guardian onboarding contract. The
// identity package supplies the application-owned implementation so raw
// bearer values stay at the authentication boundary.
package guardian

import (
	"context"
	"errors"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

const (
	TermsVersion   = "terms-v1"
	PrivacyVersion = "privacy-v1"
	TermsNotice    = "You may register only the guardian and student relationship you are authorized to provide."
	PrivacyNotice  = "MiniClass uses this verified mailbox and roster relationship to provide the current school-year guardian experience."
)

var (
	ErrRegistrationInvalid = errors.New("guardian registration entry is invalid or expired")
	ErrInvitationInvalid   = errors.New("guardian invitation is invalid or expired")
	ErrOnboardingInvalid   = errors.New("guardian onboarding session is invalid or expired")
	ErrMailboxUnverified   = errors.New("guardian mailbox has not been verified")
	ErrConsentRequired     = errors.New("current terms and privacy acceptance is required")
	ErrConsentInvalid      = errors.New("guardian terms and privacy acceptance is invalid")
	ErrOnboardingRateLimit = errors.New("guardian onboarding rate limit exceeded")
	ErrOTPInvalid          = errors.New("guardian onboarding OTP is invalid or expired")
	ErrSignupNoticeInvalid = errors.New("guardian signup notice acceptance is invalid")
)

type SignupNotice struct {
	Content string
	Version int
	Hash    []byte
}

type Policy struct {
	TermsVersion   string
	TermsNotice    string
	PrivacyVersion string
	PrivacyNotice  string
	SignupNotice   *SignupNotice
}

type Session struct {
	Token          string
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	Email          string
	Verified       bool
	Consented      bool
	ExpiresAt      time.Time
	IdleExpiresAt  time.Time
	Policy         Policy
}

type RegistrationEntry struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	Token          string
	ExpiresAt      time.Time
	Generation     int
}

type InvitationContact struct {
	ID              ids.XID
	Email           string
	Status          string
	ExpiresAt       time.Time
	ConsumedAt      *time.Time
	RevokedAt       *time.Time
	Generation      int
	InvitationToken string
}

type InvitationImportRow struct {
	Row       int    `json:"row"`
	Email     string `json:"email,omitempty"`
	Status    string `json:"status" enum:"issued,duplicate,error"`
	Error     string `json:"error,omitempty"`
	Token     string `json:"token,omitempty" doc:"Returned only at issuance; never stored or exported."`
	ContactID string `json:"contact_id,omitempty"`
}

type InvitationImportResult struct {
	Rows []InvitationImportRow `json:"rows"`
}

type InvitationExportRow struct {
	Email      string     `json:"email"`
	Status     string     `json:"status"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	ConsumedAt *time.Time `json:"consumed_at,omitempty"`
}

type BeginInput struct {
	EntryToken string
	Now        time.Time
}

type RedeemInput struct {
	InvitationToken string
	Now             time.Time
}

type OTPRequestInput struct {
	SessionToken string
	Email        string
	Now          time.Time
}

type OTPRequestResult struct {
	Accepted    bool
	ChallengeID string
}

type OTPVerifyInput struct {
	SessionToken string
	ChallengeID  string
	Code         string
	Now          time.Time
}

type ConsentInput struct {
	SessionToken        string
	Email               string
	TermsVersion        string
	PrivacyVersion      string
	SignupNoticeVersion *int
	SignupNoticeHash    []byte
	SourceSurface       string
	Now                 time.Time
}

type CompleteInput struct {
	SessionToken      string
	AdultGivenName    string
	AdultFamilyName   string
	StudentGivenName  string
	StudentFamilyName string
	GradeLevelID      ids.XID
	HomeroomID        ids.XID
	RelationshipType  data.GuardianRelationshipType
	Now               time.Time
}

type Completion struct {
	AdultID        ids.XID
	StudentID      ids.XID
	RelationshipID ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
}

type Service interface {
	CreateRegistrationEntry(context.Context, ids.XID, ids.XID, audit.Actor, time.Time) (RegistrationEntry, error)
	GetRegistrationEntry(context.Context, ids.XID, ids.XID) (RegistrationEntry, error)
	RevokeRegistrationEntry(context.Context, ids.XID, ids.XID, audit.Actor, time.Time) error
	ImportInvitationContacts(context.Context, ids.XID, ids.XID, []byte, audit.Actor, time.Time) (InvitationImportResult, error)
	ListInvitationContacts(context.Context, ids.XID, ids.XID, audit.Actor) ([]InvitationExportRow, error)
	RevokeInvitationContact(context.Context, ids.XID, ids.XID, ids.XID, audit.Actor, time.Time) error
	RevokeOnboardingSession(context.Context, ids.XID, ids.XID, ids.XID, audit.Actor, time.Time) error
	UpdateSignupNotice(context.Context, ids.XID, *string, audit.Actor, time.Time) (Policy, error)
	Begin(context.Context, BeginInput) (Session, error)
	Redeem(context.Context, RedeemInput) (Session, error)
	RequestOTP(context.Context, OTPRequestInput) (OTPRequestResult, error)
	VerifyOTP(context.Context, OTPVerifyInput) (Session, error)
	AcceptConsent(context.Context, ConsentInput) (Session, error)
	Complete(context.Context, CompleteInput) (Completion, error)
	GetSession(context.Context, string, time.Time) (Session, error)
}
