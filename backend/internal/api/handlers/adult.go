package handlers

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/api/problems"
	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/auth"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type AdultService interface {
	Create(context.Context, string, ids.XID, audit.Actor, people.AdultCreateInput) (data.Adult, error)
	List(context.Context, string, ids.XID) ([]data.Adult, error)
	Get(context.Context, string, ids.XID, ids.XID) (data.Adult, error)
	Update(context.Context, string, ids.XID, ids.XID, audit.Actor, people.AdultUpdateInput) (data.Adult, error)
	Delete(context.Context, string, ids.XID, ids.XID, audit.Actor) error
}

type AdultHandler struct{ service AdultService }

func NewAdultHandler(service AdultService) *AdultHandler { return &AdultHandler{service: service} }

type AdultResponse struct {
	ID                  string    `json:"id" doc:"Opaque adult identifier."`
	OrganizationID      string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID        string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	LegalGivenName      string    `json:"legal_given_name"`
	LegalFamilyName     string    `json:"legal_family_name"`
	PreferredGivenName  *string   `json:"preferred_given_name,omitempty"`
	Email               *string   `json:"email,omitempty"`
	Phone               *string   `json:"phone,omitempty"`
	ExternalIdentifier  *string   `json:"external_identifier,omitempty"`
	ParticipationIntent string    `json:"participation_intent" enum:"lead,help,unavailable"`
	DisplayName         string    `json:"display_name"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type AdultPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	AdultID      string `path:"adultID" minLength:"1" doc:"Opaque adult identifier."`
}

type AdultYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
}

type AdultListOutput struct{ Body []AdultResponse }
type AdultOutput struct{ Body AdultResponse }
type AdultDeleteOutput struct{}

type CreateAdultInput struct {
	AdultYearPathInput
	Body struct {
		LegalGivenName      string  `json:"legal_given_name" minLength:"1"`
		LegalFamilyName     string  `json:"legal_family_name" minLength:"1"`
		PreferredGivenName  *string `json:"preferred_given_name,omitempty"`
		Email               *string `json:"email,omitempty"`
		Phone               *string `json:"phone,omitempty"`
		ExternalIdentifier  *string `json:"external_identifier,omitempty"`
		ParticipationIntent string  `json:"participation_intent" enum:"lead,help,unavailable"`
	}
}

type ListAdultsInput struct{ AdultYearPathInput }

type GetAdultInput struct{ AdultPathInput }

type UpdateAdultInput struct {
	AdultPathInput
	Body struct {
		LegalGivenName      *string `json:"legal_given_name,omitempty" minLength:"1"`
		LegalFamilyName     *string `json:"legal_family_name,omitempty" minLength:"1"`
		PreferredGivenName  *string `json:"preferred_given_name,omitempty"`
		Email               *string `json:"email,omitempty"`
		Phone               *string `json:"phone,omitempty"`
		ExternalIdentifier  *string `json:"external_identifier,omitempty"`
		ParticipationIntent *string `json:"participation_intent,omitempty" enum:"lead,help,unavailable"`
	}
}

type DeleteAdultInput struct{ AdultPathInput }

func (h *AdultHandler) List(ctx context.Context, input *ListAdultsInput) (*AdultListOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, adultProblem(errors.New("adult service is not configured"))
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, adultNotFound()
	}
	rows, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, adultProblem(err)
	}
	response := make([]AdultResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, adultResponse(row))
	}
	return &AdultListOutput{Body: response}, nil
}

func (h *AdultHandler) Create(ctx context.Context, input *CreateAdultInput) (*AdultOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, adultProblem(errors.New("adult service is not configured"))
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, adultNotFound()
	}
	row, err := h.service.Create(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), adultActor(account), people.AdultCreateInput{
		LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName,
		PreferredGivenName: input.Body.PreferredGivenName, Email: input.Body.Email, Phone: input.Body.Phone,
		ExternalIdentifier:  input.Body.ExternalIdentifier,
		ParticipationIntent: data.AdultParticipationIntent(strings.TrimSpace(input.Body.ParticipationIntent)),
	})
	if err != nil {
		return nil, adultProblem(err)
	}
	return &AdultOutput{Body: adultResponse(row)}, nil
}

func (h *AdultHandler) Get(ctx context.Context, input *GetAdultInput) (*AdultOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.AdultID) == "" {
		return nil, adultNotFound()
	}
	row, err := h.service.Get(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.AdultID))
	if err != nil {
		return nil, adultProblem(err)
	}
	return &AdultOutput{Body: adultResponse(row)}, nil
}

func (h *AdultHandler) Update(ctx context.Context, input *UpdateAdultInput) (*AdultOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.AdultID) == "" {
		return nil, adultNotFound()
	}
	serviceInput := people.AdultUpdateInput{LegalGivenName: input.Body.LegalGivenName, LegalFamilyName: input.Body.LegalFamilyName}
	if input.Body.PreferredGivenName != nil {
		value := input.Body.PreferredGivenName
		serviceInput.PreferredGivenName = &value
	}
	if input.Body.Email != nil {
		value := input.Body.Email
		serviceInput.Email = &value
	}
	if input.Body.Phone != nil {
		value := input.Body.Phone
		serviceInput.Phone = &value
	}
	if input.Body.ExternalIdentifier != nil {
		value := input.Body.ExternalIdentifier
		serviceInput.ExternalIdentifier = &value
	}
	if input.Body.ParticipationIntent != nil {
		intent := data.AdultParticipationIntent(strings.TrimSpace(*input.Body.ParticipationIntent))
		serviceInput.ParticipationIntent = &intent
	}
	row, err := h.service.Update(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.AdultID), adultActor(account), serviceInput)
	if err != nil {
		return nil, adultProblem(err)
	}
	return &AdultOutput{Body: adultResponse(row)}, nil
}

func (h *AdultHandler) Delete(ctx context.Context, input *DeleteAdultInput) (*AdultDeleteOutput, error) {
	account, err := adultAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.AdultID) == "" {
		return nil, adultNotFound()
	}
	if err := h.service.Delete(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.AdultID), adultActor(account)); err != nil {
		return nil, adultProblem(err)
	}
	return &AdultDeleteOutput{}, nil
}

func adultAccount(ctx context.Context) (auth.AccountPrincipal, error) {
	principal, ok := auth.PrincipalFromContext(ctx)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "resolved principal is missing")
	}
	account, ok := principal.(auth.AccountPrincipal)
	if !ok {
		return auth.AccountPrincipal{}, problems.New(http.StatusInternalServerError, problems.AuthenticationUnavailable, "account principal has an unsupported type")
	}
	return account, nil
}

func adultActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func adultResponse(row data.Adult) AdultResponse {
	preferred := row.PreferredGivenName
	legalGiven, legalFamily := row.LegalGivenName, row.LegalFamilyName
	return AdultResponse{
		ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID),
		LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, PreferredGivenName: row.PreferredGivenName,
		Email: row.Email, Phone: row.Phone, ExternalIdentifier: row.ExternalIdentifier,
		ParticipationIntent: string(row.ParticipationIntent), DisplayName: people.DisplayName(preferred, &legalGiven, &legalFamily),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func adultNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "adult not found")
}

func adultProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return adultNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.AdultExternalIdentifierConflict, "the external identifier is already used in this school year")
	case strings.Contains(err.Error(), "names are required"), strings.Contains(err.Error(), "invalid participation intent"):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, err.Error())
	case errors.Is(err, people.ErrNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change adult")
	}
}
