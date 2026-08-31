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
	"github.com/chrismott/miniclass/internal/vocabulary"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type VocabularyService interface {
	List(context.Context, string, ids.XID, bool) (vocabulary.Snapshot, error)
	GetGrade(context.Context, string, ids.XID, ids.XID) (data.GradeLevel, error)
	CreateGrade(context.Context, string, ids.XID, audit.Actor, string, string) (data.GradeLevel, error)
	UpdateGrade(context.Context, string, ids.XID, ids.XID, audit.Actor, vocabulary.GradeLevelUpdate) (data.GradeLevel, error)
	ReorderGrades(context.Context, string, ids.XID, audit.Actor, []ids.XID) ([]data.GradeLevel, error)
	GetHomeroom(context.Context, string, ids.XID, ids.XID) (data.Homeroom, error)
	CreateHomeroom(context.Context, string, ids.XID, audit.Actor, string, *string) (data.Homeroom, error)
	UpdateHomeroom(context.Context, string, ids.XID, ids.XID, audit.Actor, vocabulary.HomeroomUpdate) (data.Homeroom, error)
	UpdateHomeroomLabel(context.Context, string, audit.Actor, string) (data.VocabularySettings, error)
}

type VocabularyHandler struct {
	service VocabularyService
}

func NewVocabularyHandler(service VocabularyService) *VocabularyHandler {
	return &VocabularyHandler{service: service}
}

type VocabularyListInput struct {
	SchoolYearID   string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	IncludeRetired bool   `query:"include_retired" doc:"Include retired entries for administration; picker callers should omit this."`
}

type VocabularyListOutput struct {
	Body VocabularyResponse
}

type GradeLevelListOutput struct {
	Body []GradeLevelOutput
}

type HomeroomListOutput struct {
	Body []HomeroomOutput
}

type VocabularyResponse struct {
	SchoolYearID  string             `json:"school_year_id" doc:"Opaque school-year identifier."`
	HomeroomLabel string             `json:"homeroom_label" doc:"Configured label for a homeroom."`
	GradeLevels   []GradeLevelOutput `json:"grade_levels"`
	Homerooms     []HomeroomOutput   `json:"homerooms"`
}

type GradeLevelOutput struct {
	ID           string     `json:"id" doc:"Opaque grade-level identifier."`
	SchoolYearID string     `json:"school_year_id" doc:"Opaque school-year identifier."`
	Code         string     `json:"code"`
	Label        string     `json:"label"`
	Ordinal      int        `json:"ordinal" doc:"Explicit ordering value; never inferred from code or label."`
	RetiredAt    *time.Time `json:"retired_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type HomeroomOutput struct {
	ID                 string     `json:"id" doc:"Opaque homeroom identifier."`
	SchoolYearID       string     `json:"school_year_id" doc:"Opaque school-year identifier."`
	Name               string     `json:"name"`
	ExternalIdentifier *string    `json:"external_identifier" nullable:"true"`
	RetiredAt          *time.Time `json:"retired_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type GradeLevelPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	GradeLevelID string `path:"gradeLevelID" minLength:"1" doc:"Opaque grade-level identifier."`
}

type HomeroomPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	HomeroomID   string `path:"homeroomID" minLength:"1" doc:"Opaque homeroom identifier."`
}

type GradeLevelItemOutput struct{ Body GradeLevelOutput }
type HomeroomItemOutput struct{ Body HomeroomOutput }

type CreateGradeLevelInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	Body         struct {
		Code  string `json:"code" minLength:"1"`
		Label string `json:"label" minLength:"1"`
	}
}

type CreateHomeroomInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	Body         struct {
		Name               string  `json:"name" minLength:"1"`
		ExternalIdentifier *string `json:"external_identifier,omitempty" nullable:"true"`
	}
}

type UpdateGradeLevelInput struct {
	GradeLevelPathInput
	Body struct {
		Code    *string `json:"code,omitempty" minLength:"1"`
		Label   *string `json:"label,omitempty" minLength:"1"`
		Retired *bool   `json:"retired,omitempty"`
	}
}

type UpdateHomeroomInput struct {
	HomeroomPathInput
	Body struct {
		Name               *string `json:"name,omitempty" minLength:"1"`
		ExternalIdentifier *string `json:"external_identifier,omitempty" nullable:"true"`
		Retired            *bool   `json:"retired,omitempty"`
	}
}

type ReorderGradeLevelsInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1" doc:"Opaque school-year identifier."`
	Body         struct {
		IDs []string `json:"ids" minItems:"1" doc:"Every grade-level ID in its new ordinal order."`
	}
}

type ReorderGradeLevelsOutput struct {
	Body []GradeLevelOutput
}

type UpdateVocabularySettingsInput struct {
	Body struct {
		HomeroomLabel string `json:"homeroom_label" minLength:"1"`
	}
}

type UpdateVocabularySettingsOutput struct {
	Body struct {
		OrganizationID string `json:"organization_id"`
		HomeroomLabel  string `json:"homeroom_label"`
	}
}

func (h *VocabularyHandler) List(ctx context.Context, input *VocabularyListInput) (*VocabularyListOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, vocabularyProblem(errors.New("vocabulary service is not configured"))
	}
	includeRetired := input != nil && input.IncludeRetired
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	snapshot, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), includeRetired)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &VocabularyListOutput{Body: vocabularyResponse(snapshot)}, nil
}

func (h *VocabularyHandler) ListGrades(ctx context.Context, input *VocabularyListInput) (*GradeLevelListOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, vocabularyProblem(errors.New("vocabulary service is not configured"))
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	includeRetired := input.IncludeRetired
	snapshot, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), includeRetired)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	response := make([]GradeLevelOutput, 0, len(snapshot.Grades))
	for _, row := range snapshot.Grades {
		response = append(response, gradeLevelResponse(row))
	}
	return &GradeLevelListOutput{Body: response}, nil
}

func (h *VocabularyHandler) ListHomerooms(ctx context.Context, input *VocabularyListInput) (*HomeroomListOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, vocabularyProblem(errors.New("vocabulary service is not configured"))
	}
	if input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	includeRetired := input.IncludeRetired
	snapshot, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), includeRetired)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	response := make([]HomeroomOutput, 0, len(snapshot.Homerooms))
	for _, row := range snapshot.Homerooms {
		response = append(response, homeroomResponse(row))
	}
	return &HomeroomListOutput{Body: response}, nil
}

func (h *VocabularyHandler) GetGrade(ctx context.Context, input *GradeLevelPathInput) (*GradeLevelItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.GradeLevelID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "grade level not found")
	}
	row, err := h.service.GetGrade(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.GradeLevelID))
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &GradeLevelItemOutput{Body: gradeLevelResponse(row)}, nil
}

func (h *VocabularyHandler) CreateGrade(ctx context.Context, input *CreateGradeLevelInput) (*GradeLevelItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, vocabularyProblem(errors.New("vocabulary service is not configured"))
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "grade-level body is required")
	}
	if strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	row, err := h.service.CreateGrade(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), vocabularyActor(account), input.Body.Code, input.Body.Label)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &GradeLevelItemOutput{Body: gradeLevelResponse(row)}, nil
}

func (h *VocabularyHandler) UpdateGrade(ctx context.Context, input *UpdateGradeLevelInput) (*GradeLevelItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.GradeLevelID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "grade level not found")
	}
	row, err := h.service.UpdateGrade(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.GradeLevelID), vocabularyActor(account), vocabulary.GradeLevelUpdate{
		Code: input.Body.Code, Label: input.Body.Label, Retired: input.Body.Retired,
	})
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &GradeLevelItemOutput{Body: gradeLevelResponse(row)}, nil
}

func (h *VocabularyHandler) ReorderGrades(ctx context.Context, input *ReorderGradeLevelsInput) (*ReorderGradeLevelsOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "grade-level order is required")
	}
	ordered := make([]ids.XID, 0, len(input.Body.IDs))
	for _, id := range input.Body.IDs {
		ordered = append(ordered, ids.XID(strings.TrimSpace(id)))
	}
	rows, err := h.service.ReorderGrades(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), vocabularyActor(account), ordered)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	response := make([]GradeLevelOutput, 0, len(rows))
	for _, row := range rows {
		response = append(response, gradeLevelResponse(row))
	}
	return &ReorderGradeLevelsOutput{Body: response}, nil
}

func (h *VocabularyHandler) GetHomeroom(ctx context.Context, input *HomeroomPathInput) (*HomeroomItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.HomeroomID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "homeroom not found")
	}
	row, err := h.service.GetHomeroom(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HomeroomID))
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &HomeroomItemOutput{Body: homeroomResponse(row)}, nil
}

func (h *VocabularyHandler) CreateHomeroom(ctx context.Context, input *CreateHomeroomInput) (*HomeroomItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil {
		return nil, vocabularyProblem(errors.New("vocabulary service is not configured"))
	}
	if input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "homeroom body is required")
	}
	if strings.TrimSpace(input.SchoolYearID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "school year not found")
	}
	row, err := h.service.CreateHomeroom(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), vocabularyActor(account), input.Body.Name, input.Body.ExternalIdentifier)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &HomeroomItemOutput{Body: homeroomResponse(row)}, nil
}

func (h *VocabularyHandler) UpdateHomeroom(ctx context.Context, input *UpdateHomeroomInput) (*HomeroomItemOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.HomeroomID) == "" {
		return nil, problems.New(http.StatusNotFound, problems.ResourceNotFound, "homeroom not found")
	}
	var externalIdentifier **string
	if input.Body.ExternalIdentifier != nil {
		value := input.Body.ExternalIdentifier
		externalIdentifier = &value
	}
	row, err := h.service.UpdateHomeroom(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.HomeroomID), vocabularyActor(account), vocabulary.HomeroomUpdate{
		Name: input.Body.Name, ExternalIdentifier: externalIdentifier, Retired: input.Body.Retired,
	})
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &HomeroomItemOutput{Body: homeroomResponse(row)}, nil
}

func (h *VocabularyHandler) UpdateSettings(ctx context.Context, input *UpdateVocabularySettingsInput) (*UpdateVocabularySettingsOutput, error) {
	account, err := vocabularyAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, problems.New(http.StatusBadRequest, problems.ResourceNotFound, "vocabulary settings body is required")
	}
	settings, err := h.service.UpdateHomeroomLabel(ctx, string(account.OrganizationID), vocabularyActor(account), input.Body.HomeroomLabel)
	if err != nil {
		return nil, vocabularyProblem(err)
	}
	return &UpdateVocabularySettingsOutput{Body: struct {
		OrganizationID string `json:"organization_id"`
		HomeroomLabel  string `json:"homeroom_label"`
	}{OrganizationID: string(settings.OrganizationID), HomeroomLabel: settings.HomeroomLabel}}, nil
}

func vocabularyAccount(ctx context.Context) (auth.AccountPrincipal, error) {
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

func vocabularyActor(account auth.AccountPrincipal) audit.Actor {
	userID := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &userID, Label: account.Email}
}

func vocabularyResponse(snapshot vocabulary.Snapshot) VocabularyResponse {
	response := VocabularyResponse{SchoolYearID: string(snapshot.SchoolYearID), HomeroomLabel: snapshot.Settings.HomeroomLabel}
	response.GradeLevels = make([]GradeLevelOutput, 0, len(snapshot.Grades))
	for _, row := range snapshot.Grades {
		response.GradeLevels = append(response.GradeLevels, gradeLevelResponse(row))
	}
	response.Homerooms = make([]HomeroomOutput, 0, len(snapshot.Homerooms))
	for _, row := range snapshot.Homerooms {
		response.Homerooms = append(response.Homerooms, homeroomResponse(row))
	}
	return response
}

func gradeLevelResponse(row data.GradeLevel) GradeLevelOutput {
	return GradeLevelOutput{ID: string(row.ID), SchoolYearID: string(row.SchoolYearID), Code: row.Code, Label: row.Label, Ordinal: row.Ordinal, RetiredAt: row.RetiredAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func homeroomResponse(row data.Homeroom) HomeroomOutput {
	return HomeroomOutput{ID: string(row.ID), SchoolYearID: string(row.SchoolYearID), Name: row.Name, ExternalIdentifier: row.ExternalIdentifier, RetiredAt: row.RetiredAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func vocabularyProblem(err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return problems.New(http.StatusNotFound, problems.ResourceNotFound, "vocabulary entry not found")
	case errors.Is(err, vocabulary.ErrNoChanges):
		return problems.New(http.StatusConflict, problems.SchoolYearTransitionInvalid, err.Error())
	case errors.Is(err, vocabulary.ErrInvalid):
		return problems.New(http.StatusBadRequest, problems.SchoolYearTransitionInvalid, err.Error())
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case isUniqueViolation(err):
		return problems.New(http.StatusConflict, problems.HomeroomExternalIdentifierConflict, "the homeroom external identifier is already used in this school year")
	case strings.Contains(err.Error(), "is empty"), strings.Contains(err.Error(), "must be positive"):
		return problems.New(http.StatusBadRequest, problems.SchoolYearTransitionInvalid, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change vocabulary")
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
