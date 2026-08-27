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

type HouseholdService interface {
	CreateHousehold(context.Context, string, ids.XID, audit.Actor, people.HouseholdCreateInput) (data.Household, error)
	ListHouseholds(context.Context, string, ids.XID) ([]data.Household, error)
	GetHousehold(context.Context, string, ids.XID, ids.XID) (data.Household, error)
	UpdateHousehold(context.Context, string, ids.XID, ids.XID, audit.Actor, people.HouseholdUpdateInput) (data.Household, error)
	DeleteHousehold(context.Context, string, ids.XID, ids.XID, audit.Actor) error
	ListHouseholdMembership(context.Context, string, ids.XID) (people.HouseholdMembership, error)
	ListHouseholdStudents(context.Context, string, ids.XID, ids.XID) ([]data.HouseholdStudent, error)
	AddStudentToHousehold(context.Context, string, ids.XID, ids.XID, ids.XID, audit.Actor) (data.HouseholdStudent, error)
	RemoveStudentFromHousehold(context.Context, string, ids.XID, ids.XID, ids.XID, audit.Actor) error
	ListHouseholdAdults(context.Context, string, ids.XID, ids.XID) ([]data.HouseholdAdult, error)
	AddAdultToHousehold(context.Context, string, ids.XID, ids.XID, ids.XID, audit.Actor) (data.HouseholdAdult, error)
	RemoveAdultFromHousehold(context.Context, string, ids.XID, ids.XID, ids.XID, audit.Actor) error
}

type HouseholdHandler struct{ service HouseholdService }

func NewHouseholdHandler(service HouseholdService) *HouseholdHandler {
	return &HouseholdHandler{service: service}
}

type HouseholdResponse struct {
	ID             string    `json:"id" doc:"Opaque household identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	DisplayName    string    `json:"display_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type HouseholdStudentResponse struct {
	ID          string `json:"id" doc:"Opaque membership identifier."`
	HouseholdID string `json:"household_id"`
	StudentID   string `json:"student_id"`
}

type HouseholdAdultResponse struct {
	ID          string `json:"id" doc:"Opaque membership identifier."`
	HouseholdID string `json:"household_id"`
	AdultID     string `json:"adult_id"`
}

// HouseholdMembershipResponse carries a whole school year's membership so that
// "which households does this person belong to" costs one request rather than
// one per household. Only opaque identifiers cross the boundary; a display name
// is joined from the household listing by identifier (SPEC §8.7).
type HouseholdMembershipResponse struct {
	Students []HouseholdStudentResponse `json:"students" doc:"Every student membership in the school year."`
	Adults   []HouseholdAdultResponse   `json:"adults" doc:"Every adult membership in the school year."`
}

type HouseholdYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
}
type HouseholdPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	HouseholdID  string `path:"householdID" minLength:"1" doc:"Opaque household identifier."`
}
type HouseholdMemberPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	HouseholdID  string `path:"householdID" minLength:"1"`
	MemberID     string `path:"memberID" minLength:"1"`
}
type HouseholdListOutput struct{ Body []HouseholdResponse }
type HouseholdOutput struct{ Body HouseholdResponse }
type HouseholdDeleteOutput struct{}
type HouseholdMembershipOutput struct{ Body HouseholdMembershipResponse }
type HouseholdStudentListOutput struct{ Body []HouseholdStudentResponse }
type HouseholdStudentOutput struct{ Body HouseholdStudentResponse }
type HouseholdAdultListOutput struct{ Body []HouseholdAdultResponse }
type HouseholdAdultOutput struct{ Body HouseholdAdultResponse }
type CreateHouseholdInput struct {
	HouseholdYearPathInput
	Body struct {
		DisplayName string `json:"display_name" minLength:"1"`
	}
}
type ListHouseholdsInput struct{ HouseholdYearPathInput }
type ListHouseholdMembershipInput struct{ HouseholdYearPathInput }
type GetHouseholdInput struct{ HouseholdPathInput }
type UpdateHouseholdInput struct {
	HouseholdPathInput
	Body struct {
		DisplayName *string `json:"display_name,omitempty" minLength:"1"`
	}
}
type DeleteHouseholdInput struct{ HouseholdPathInput }
type HouseholdStudentYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	HouseholdID  string `path:"householdID" minLength:"1"`
}
type ListHouseholdStudentsInput struct{ HouseholdStudentYearPathInput }
type AddHouseholdStudentInput struct {
	HouseholdStudentYearPathInput
	Body struct {
		StudentID string `json:"student_id" minLength:"1"`
	}
}
type DeleteHouseholdStudentInput struct {
	HouseholdStudentYearPathInput
	StudentID string `path:"studentID" minLength:"1"`
}
type ListHouseholdAdultsInput struct{ HouseholdStudentYearPathInput }
type AddHouseholdAdultInput struct {
	HouseholdStudentYearPathInput
	Body struct {
		AdultID string `json:"adult_id" minLength:"1"`
	}
}
type DeleteHouseholdAdultInput struct {
	HouseholdStudentYearPathInput
	AdultID string `path:"adultID" minLength:"1"`
}

func (h *HouseholdHandler) List(ctx context.Context, input *ListHouseholdsInput) (*HouseholdListOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	rows, err := h.service.ListHouseholds(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, householdProblem(err)
	}
	result := make([]HouseholdResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, householdResponse(row))
	}
	return &HouseholdListOutput{Body: result}, nil
}

func (h *HouseholdHandler) Create(ctx context.Context, input *CreateHouseholdInput) (*HouseholdOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.CreateHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), householdActor(account), people.HouseholdCreateInput{DisplayName: input.Body.DisplayName})
	if err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdOutput{Body: householdResponse(row)}, nil
}

func (h *HouseholdHandler) Get(ctx context.Context, input *GetHouseholdInput) (*HouseholdOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.GetHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID))
	if err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdOutput{Body: householdResponse(row)}, nil
}

func (h *HouseholdHandler) Update(ctx context.Context, input *UpdateHouseholdInput) (*HouseholdOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.UpdateHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), householdActor(account), people.HouseholdUpdateInput{DisplayName: input.Body.DisplayName})
	if err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdOutput{Body: householdResponse(row)}, nil
}

func (h *HouseholdHandler) Delete(ctx context.Context, input *DeleteHouseholdInput) (*HouseholdDeleteOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	if err := h.service.DeleteHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), householdActor(account)); err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdDeleteOutput{}, nil
}

func (h *HouseholdHandler) ListMembership(ctx context.Context, input *ListHouseholdMembershipInput) (*HouseholdMembershipOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	membership, err := h.service.ListHouseholdMembership(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, householdProblem(err)
	}
	body := HouseholdMembershipResponse{
		Students: make([]HouseholdStudentResponse, 0, len(membership.Students)),
		Adults:   make([]HouseholdAdultResponse, 0, len(membership.Adults)),
	}
	for _, row := range membership.Students {
		body.Students = append(body.Students, householdStudentResponse(row))
	}
	for _, row := range membership.Adults {
		body.Adults = append(body.Adults, householdAdultResponse(row))
	}
	return &HouseholdMembershipOutput{Body: body}, nil
}

func (h *HouseholdHandler) ListStudents(ctx context.Context, input *ListHouseholdStudentsInput) (*HouseholdStudentListOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	rows, err := h.service.ListHouseholdStudents(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID))
	if err != nil {
		return nil, householdProblem(err)
	}
	result := make([]HouseholdStudentResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, householdStudentResponse(row))
	}
	return &HouseholdStudentListOutput{Body: result}, nil
}

func (h *HouseholdHandler) AddStudent(ctx context.Context, input *AddHouseholdStudentInput) (*HouseholdStudentOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.AddStudentToHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), ids.XID(input.Body.StudentID), householdActor(account))
	if err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdStudentOutput{Body: householdStudentResponse(row)}, nil
}

func (h *HouseholdHandler) DeleteStudent(ctx context.Context, input *DeleteHouseholdStudentInput) (*HouseholdDeleteOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	if err := h.service.RemoveStudentFromHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), ids.XID(input.StudentID), householdActor(account)); err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdDeleteOutput{}, nil
}

func (h *HouseholdHandler) ListAdults(ctx context.Context, input *ListHouseholdAdultsInput) (*HouseholdAdultListOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	rows, err := h.service.ListHouseholdAdults(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID))
	if err != nil {
		return nil, householdProblem(err)
	}
	result := make([]HouseholdAdultResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, householdAdultResponse(row))
	}
	return &HouseholdAdultListOutput{Body: result}, nil
}

func (h *HouseholdHandler) AddAdult(ctx context.Context, input *AddHouseholdAdultInput) (*HouseholdAdultOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	row, err := h.service.AddAdultToHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), ids.XID(input.Body.AdultID), householdActor(account))
	if err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdAdultOutput{Body: householdAdultResponse(row)}, nil
}

func (h *HouseholdHandler) DeleteAdult(ctx context.Context, input *DeleteHouseholdAdultInput) (*HouseholdDeleteOutput, error) {
	account, err := householdAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, householdNotFound()
	}
	if err := h.service.RemoveAdultFromHousehold(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HouseholdID), ids.XID(input.AdultID), householdActor(account)); err != nil {
		return nil, householdProblem(err)
	}
	return &HouseholdDeleteOutput{}, nil
}

func householdAccount(ctx context.Context) (auth.AccountPrincipal, error) {
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

func householdActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func householdResponse(row data.Household) HouseholdResponse {
	return HouseholdResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), DisplayName: row.DisplayName, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func householdStudentResponse(row data.HouseholdStudent) HouseholdStudentResponse {
	return HouseholdStudentResponse{ID: string(row.ID), HouseholdID: string(row.HouseholdID), StudentID: string(row.StudentID)}
}
func householdAdultResponse(row data.HouseholdAdult) HouseholdAdultResponse {
	return HouseholdAdultResponse{ID: string(row.ID), HouseholdID: string(row.HouseholdID), AdultID: string(row.AdultID)}
}
func householdNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "household not found")
}

func householdProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return householdNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.HouseholdConflict, "the household relationship already exists")
	case strings.Contains(err.Error(), "display name is required"), strings.Contains(err.Error(), "invalid relationship type"):
		return problems.New(http.StatusBadRequest, problems.ResourceNotFound, err.Error())
	case errors.Is(err, people.ErrHouseholdNoChanges), errors.Is(err, people.ErrGuardianRelationshipNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change household data")
	}
}
