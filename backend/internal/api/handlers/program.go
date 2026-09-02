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
	"github.com/chrismott/miniclass/internal/preference"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type ProgramService interface {
	Create(context.Context, string, audit.Actor, ids.XID, string) (data.Program, error)
	List(context.Context, string, ids.XID) ([]data.Program, error)
	ListInterestAreas(context.Context, string, ids.XID, ids.XID, bool) ([]data.InterestArea, error)
	GetInterestArea(context.Context, string, ids.XID, ids.XID, ids.XID) (data.InterestArea, error)
	CreateInterestArea(context.Context, string, audit.Actor, ids.XID, ids.XID, string) (data.InterestArea, error)
	UpdateInterestArea(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, programservice.InterestAreaUpdate) (data.InterestArea, error)
	ReorderInterestAreas(context.Context, string, audit.Actor, ids.XID, ids.XID, []ids.XID) ([]data.InterestArea, error)
	ListMemberships(context.Context, string, ids.XID, ids.XID) ([]data.ProgramMembership, error)
	AddMembership(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) (data.ProgramMembership, error)
	DeleteMembership(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) error
	CountStudentsWithoutGrade(context.Context, string, ids.XID) (int64, error)
	CreateSession(context.Context, string, audit.Actor, ids.XID, ids.XID, string, []time.Time) (data.Session, error)
	ListSessions(context.Context, string, ids.XID, ids.XID) ([]data.Session, error)
	GetSession(context.Context, string, ids.XID, ids.XID, ids.XID) (data.Session, error)
	UpdateSession(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, programservice.SessionUpdate) (data.Session, error)
	DeleteSession(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) error
	GetCatalogFeasibility(context.Context, string, ids.XID, ids.XID, ids.XID) (programservice.CatalogFeasibility, error)
	TransitionSession(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, programservice.SessionTransitionInput) (programservice.SessionTransitionResult, error)
	ListMeetingDates(context.Context, string, ids.XID, ids.XID, ids.XID) ([]data.MeetingDate, error)
	GetMeetingDate(context.Context, string, ids.XID, ids.XID, ids.XID, ids.XID) (data.MeetingDate, error)
	CreateMeetingDate(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, time.Time) (data.MeetingDate, error)
	UpdateMeetingDate(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID, time.Time) (data.MeetingDate, error)
	DeleteMeetingDate(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID) error
	ListOfferings(context.Context, string, ids.XID, ids.XID, ids.XID) ([]data.Offering, error)
	GetOffering(context.Context, string, ids.XID, ids.XID, ids.XID, ids.XID) (data.Offering, error)
	CreateOffering(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, string, string, *int, int, ids.XID, ids.XID, string, string, string, *ids.XID) (data.Offering, error)
	UpdateOffering(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID, programservice.OfferingUpdate) (data.Offering, error)
	DeleteOffering(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID) error
	CreateSessionNonParticipation(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID, string) (data.SessionNonParticipation, error)
	ListSessionNonParticipations(context.Context, string, ids.XID, ids.XID, ids.XID) ([]data.SessionNonParticipation, error)
	GetSessionNonParticipation(context.Context, string, ids.XID, ids.XID, ids.XID, ids.XID) (data.SessionNonParticipation, error)
	UpdateSessionNonParticipation(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID, programservice.SessionNonParticipationUpdate) (data.SessionNonParticipation, error)
	DeleteSessionNonParticipation(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, ids.XID) error
	ListParticipatingMemberships(context.Context, string, ids.XID, ids.XID, ids.XID) ([]data.ProgramMembership, error)
	GetProgramObjectiveWeights(context.Context, string, ids.XID, ids.XID) (data.ObjectiveWeightsView, error)
	UpdateProgramObjectiveWeights(context.Context, string, audit.Actor, ids.XID, ids.XID, data.ObjectiveWeights) (data.ObjectiveWeightsView, error)
	GetSessionObjectiveWeights(context.Context, string, ids.XID, ids.XID, ids.XID) (data.ObjectiveWeightsView, error)
	UpdateSessionObjectiveWeights(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, data.ObjectiveWeightOverrides, string) (data.ObjectiveWeightsView, error)
	ClearSessionObjectiveWeights(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) (data.ObjectiveWeightsView, error)
	CreateInterestProfileSurvey(context.Context, string, audit.Actor, ids.XID, ids.XID, preference.InterestProfileSurveyInput) (preference.InterestProfileSurveyView, error)
	ListInterestProfileSurveys(context.Context, string, ids.XID, ids.XID) ([]preference.InterestProfileSurveyView, error)
	GetInterestProfileSurvey(context.Context, string, ids.XID, ids.XID, ids.XID) (preference.InterestProfileSurveyView, error)
	UpdateInterestProfileSurvey(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, preference.InterestProfileSurveyUpdate) (preference.InterestProfileSurveyView, error)
	DeleteInterestProfileSurvey(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID) error
	TransitionInterestProfileSurvey(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, preference.InterestProfileSurveyTransitionInput) (preference.InterestProfileSurveyTransitionResult, error)
	RegenerateInterestProfileSurveyCodes(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, string) ([]preference.SurveyAccessCode, error)
	RevokeInterestProfileSurveyCodes(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, string) error
	RegenerateRankedChoiceAccessCodes(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, string) ([]preference.RankedChoiceAccessCode, error)
	RevokeRankedChoiceAccessCodes(context.Context, string, audit.Actor, ids.XID, ids.XID, ids.XID, string) error
}

type ProgramResponse struct {
	ID             string    `json:"id" doc:"Opaque program identifier."`
	OrganizationID string    `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string    `json:"school_year_id" doc:"Opaque school-year identifier."`
	Name           string    `json:"name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProgramMembershipResponse struct {
	ID              string    `json:"id" doc:"Opaque membership identifier."`
	OrganizationID  string    `json:"organization_id"`
	SchoolYearID    string    `json:"school_year_id"`
	ProgramID       string    `json:"program_id"`
	StudentID       string    `json:"student_id"`
	LegalGivenName  string    `json:"legal_given_name"`
	LegalFamilyName string    `json:"legal_family_name"`
	GradeLevelID    *string   `json:"grade_level_id" nullable:"true"`
	GradeMissing    bool      `json:"grade_missing" doc:"True when the member's current roster grade is unknown."`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type InterestAreaResponse struct {
	ID             string     `json:"id" doc:"Opaque interest-area identifier."`
	OrganizationID string     `json:"organization_id" doc:"Opaque organization identifier."`
	SchoolYearID   string     `json:"school_year_id" doc:"Opaque school-year identifier."`
	ProgramID      string     `json:"program_id" doc:"Opaque program identifier."`
	Label          string     `json:"label"`
	Ordinal        int        `json:"ordinal" doc:"Explicit vocabulary ordering."`
	RetiredAt      *time.Time `json:"retired_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ProgramRosterSummaryResponse struct {
	MissingGradeCount int64 `json:"missing_grade_count"`
}
type ProgramListOutput struct{ Body []ProgramResponse }
type ProgramOutput struct{ Body ProgramResponse }
type InterestAreaListOutput struct{ Body []InterestAreaResponse }
type InterestAreaOutput struct{ Body InterestAreaResponse }
type ReorderInterestAreasOutput struct{ Body []InterestAreaResponse }
type ProgramMembershipListOutput struct{ Body []ProgramMembershipResponse }
type ProgramMembershipOutput struct{ Body ProgramMembershipResponse }
type ProgramRosterSummaryOutput struct{ Body ProgramRosterSummaryResponse }
type ProgramYearPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
}
type ProgramPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
}
type InterestAreaPathInput struct {
	SchoolYearID   string `path:"schoolYearID" minLength:"1"`
	ProgramID      string `path:"programID" minLength:"1"`
	InterestAreaID string `path:"interestAreaID" minLength:"1"`
}
type InterestAreaCollectionInput struct {
	SchoolYearID   string `path:"schoolYearID" minLength:"1"`
	ProgramID      string `path:"programID" minLength:"1"`
	IncludeRetired bool   `query:"include_retired" doc:"Include retired areas for administration; picker callers should omit this."`
}
type CreateInterestAreaInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
	Body         struct {
		Label string `json:"label" minLength:"1"`
	}
}
type UpdateInterestAreaInput struct {
	InterestAreaPathInput
	Body struct {
		Label   *string `json:"label,omitempty" minLength:"1"`
		Retired *bool   `json:"retired,omitempty"`
	}
}
type InterestAreaItemOutput struct{ Body InterestAreaResponse }
type ReorderInterestAreasInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
	Body         struct {
		IDs []string `json:"ids" minItems:"1" doc:"Every interest-area ID in its new ordinal order."`
	}
}
type ProgramMembershipPathInput struct {
	SchoolYearID string `path:"schoolYearID" minLength:"1"`
	ProgramID    string `path:"programID" minLength:"1"`
	MembershipID string `path:"membershipID" minLength:"1"`
}
type ListProgramsInput struct{ ProgramYearPathInput }
type CreateProgramInput struct {
	ProgramYearPathInput
	Body struct {
		Name string `json:"name" minLength:"1"`
	}
}
type ListProgramMembershipsInput struct{ ProgramPathInput }
type CreateProgramMembershipInput struct {
	ProgramPathInput
	Body struct {
		StudentID string `json:"student_id" minLength:"1" doc:"Opaque student identifier."`
	}
}
type DeleteProgramMembershipInput struct{ ProgramMembershipPathInput }
type ProgramDeleteOutput struct{}
type MissingGradeSummaryInput struct{ ProgramYearPathInput }

type ObjectiveWeightsResponse struct {
	RankHighMax                   int     `json:"rank_high_max" minimum:"2"`
	DeficitUnwantedIncrement      float64 `json:"deficit_unwanted_increment" minimum:"0"`
	DeficitNeutralIncrement       float64 `json:"deficit_neutral_increment" minimum:"0"`
	DeficitAcceptableIncrement    float64 `json:"deficit_acceptable_increment" minimum:"0"`
	DeficitInfluence              float64 `json:"deficit_influence" minimum:"0"`
	RepeatOfferingPenalty         float64 `json:"repeat_offering_penalty" minimum:"0"`
	RepeatInterestAreaPenalty     float64 `json:"repeat_interest_area_penalty" minimum:"0"`
	TagPrefersWeight              float64 `json:"tag_prefers_weight" minimum:"0"`
	TagDiscouragesWeight          float64 `json:"tag_discourages_weight" minimum:"0"`
	PairingPrefersWeight          float64 `json:"pairing_prefers_weight" minimum:"0"`
	PairingDiscouragesWeight      float64 `json:"pairing_discourages_weight" minimum:"0"`
	BelowMinimumEnrollmentPenalty float64 `json:"below_minimum_enrollment_penalty" minimum:"0"`
	TagBalancePenalty             float64 `json:"tag_balance_penalty" minimum:"0"`
}
type ObjectiveWeightsInput struct {
	RankHighMax                   int     `json:"rank_high_max" minimum:"2"`
	DeficitUnwantedIncrement      float64 `json:"deficit_unwanted_increment" minimum:"0"`
	DeficitNeutralIncrement       float64 `json:"deficit_neutral_increment" minimum:"0"`
	DeficitAcceptableIncrement    float64 `json:"deficit_acceptable_increment" minimum:"0"`
	DeficitInfluence              float64 `json:"deficit_influence" minimum:"0"`
	RepeatOfferingPenalty         float64 `json:"repeat_offering_penalty" minimum:"0"`
	RepeatInterestAreaPenalty     float64 `json:"repeat_interest_area_penalty" minimum:"0"`
	TagPrefersWeight              float64 `json:"tag_prefers_weight" minimum:"0"`
	TagDiscouragesWeight          float64 `json:"tag_discourages_weight" minimum:"0"`
	PairingPrefersWeight          float64 `json:"pairing_prefers_weight" minimum:"0"`
	PairingDiscouragesWeight      float64 `json:"pairing_discourages_weight" minimum:"0"`
	BelowMinimumEnrollmentPenalty float64 `json:"below_minimum_enrollment_penalty" minimum:"0"`
	TagBalancePenalty             float64 `json:"tag_balance_penalty" minimum:"0"`
}
type ObjectiveWeightOverridesResponse struct {
	RankHighMax                   *int     `json:"rank_high_max,omitempty" nullable:"true" minimum:"2"`
	DeficitUnwantedIncrement      *float64 `json:"deficit_unwanted_increment,omitempty" nullable:"true" minimum:"0"`
	DeficitNeutralIncrement       *float64 `json:"deficit_neutral_increment,omitempty" nullable:"true" minimum:"0"`
	DeficitAcceptableIncrement    *float64 `json:"deficit_acceptable_increment,omitempty" nullable:"true" minimum:"0"`
	DeficitInfluence              *float64 `json:"deficit_influence,omitempty" nullable:"true" minimum:"0"`
	RepeatOfferingPenalty         *float64 `json:"repeat_offering_penalty,omitempty" nullable:"true" minimum:"0"`
	RepeatInterestAreaPenalty     *float64 `json:"repeat_interest_area_penalty,omitempty" nullable:"true" minimum:"0"`
	TagPrefersWeight              *float64 `json:"tag_prefers_weight,omitempty" nullable:"true" minimum:"0"`
	TagDiscouragesWeight          *float64 `json:"tag_discourages_weight,omitempty" nullable:"true" minimum:"0"`
	PairingPrefersWeight          *float64 `json:"pairing_prefers_weight,omitempty" nullable:"true" minimum:"0"`
	PairingDiscouragesWeight      *float64 `json:"pairing_discourages_weight,omitempty" nullable:"true" minimum:"0"`
	BelowMinimumEnrollmentPenalty *float64 `json:"below_minimum_enrollment_penalty,omitempty" nullable:"true" minimum:"0"`
	TagBalancePenalty             *float64 `json:"tag_balance_penalty,omitempty" nullable:"true" minimum:"0"`
}
type ProgramObjectiveWeightsResponse struct {
	ProgramID string                   `json:"program_id" doc:"Opaque program identifier."`
	Defaults  ObjectiveWeightsResponse `json:"defaults"`
	Effective ObjectiveWeightsResponse `json:"effective"`
}
type SessionObjectiveWeightsResponse struct {
	ProgramID string                           `json:"program_id" doc:"Opaque program identifier."`
	SessionID string                           `json:"session_id" doc:"Opaque session identifier."`
	Defaults  ObjectiveWeightsResponse         `json:"defaults"`
	Overrides ObjectiveWeightOverridesResponse `json:"overrides"`
	Effective ObjectiveWeightsResponse         `json:"effective"`
}
type ProgramObjectiveWeightsOutput struct {
	Body ProgramObjectiveWeightsResponse
}
type SessionObjectiveWeightsOutput struct {
	Body SessionObjectiveWeightsResponse
}
type GetProgramObjectiveWeightsInput struct{ ProgramPathInput }
type UpdateProgramObjectiveWeightsInput struct {
	ProgramPathInput
	Body ObjectiveWeightsInput
}
type GetSessionObjectiveWeightsInput struct{ SessionPathInput }
type UpdateSessionObjectiveWeightsInput struct {
	SessionPathInput
	Body struct {
		Overrides ObjectiveWeightOverridesResponse `json:"overrides"`
		Reason    string                           `json:"reason" minLength:"1" doc:"Why these session-specific overrides are being used."`
	}
}
type DeleteSessionObjectiveWeightsInput struct{ SessionPathInput }

type ProgramHandler struct{ service ProgramService }

func NewProgramHandler(service ProgramService) *ProgramHandler {
	return &ProgramHandler{service: service}
}

func (h *ProgramHandler) List(ctx context.Context, input *ListProgramsInput) (*ProgramListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	rows, err := h.service.List(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, programProblem(err)
	}
	result := make([]ProgramResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, programResponse(row))
	}
	return &ProgramListOutput{Body: result}, nil
}

func (h *ProgramHandler) Create(ctx context.Context, input *CreateProgramInput) (*ProgramOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.Create(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), input.Body.Name)
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramOutput{Body: programResponse(row)}, nil
}

func (h *ProgramHandler) ListInterestAreas(ctx context.Context, input *InterestAreaCollectionInput) (*InterestAreaListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, interestAreaNotFound()
	}
	rows, err := h.service.ListInterestAreas(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), input.IncludeRetired)
	if err != nil {
		return nil, interestAreaProblem(err)
	}
	result := make([]InterestAreaResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, interestAreaResponse(row))
	}
	return &InterestAreaListOutput{Body: result}, nil
}

func (h *ProgramHandler) GetInterestArea(ctx context.Context, input *InterestAreaPathInput) (*InterestAreaItemOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, interestAreaNotFound()
	}
	row, err := h.service.GetInterestArea(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.InterestAreaID))
	if err != nil {
		return nil, interestAreaProblem(err)
	}
	return &InterestAreaItemOutput{Body: interestAreaResponse(row)}, nil
}

func (h *ProgramHandler) CreateInterestArea(ctx context.Context, input *CreateInterestAreaInput) (*InterestAreaItemOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, interestAreaNotFound()
	}
	row, err := h.service.CreateInterestArea(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), input.Body.Label)
	if err != nil {
		return nil, interestAreaProblem(err)
	}
	return &InterestAreaItemOutput{Body: interestAreaResponse(row)}, nil
}

func (h *ProgramHandler) ReorderInterestAreas(ctx context.Context, input *ReorderInterestAreasInput) (*ReorderInterestAreasOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil || strings.TrimSpace(input.SchoolYearID) == "" || strings.TrimSpace(input.ProgramID) == "" {
		return nil, problems.New(http.StatusBadRequest, problems.ProgramConflict, "interest-area order is required")
	}
	ordered := make([]ids.XID, 0, len(input.Body.IDs))
	for _, id := range input.Body.IDs {
		ordered = append(ordered, ids.XID(strings.TrimSpace(id)))
	}
	rows, err := h.service.ReorderInterestAreas(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ordered)
	if err != nil {
		return nil, interestAreaProblem(err)
	}
	result := make([]InterestAreaResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, interestAreaResponse(row))
	}
	return &ReorderInterestAreasOutput{Body: result}, nil
}

func (h *ProgramHandler) UpdateInterestArea(ctx context.Context, input *UpdateInterestAreaInput) (*InterestAreaItemOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, interestAreaNotFound()
	}
	row, err := h.service.UpdateInterestArea(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.InterestAreaID), programservice.InterestAreaUpdate{Label: input.Body.Label, Retired: input.Body.Retired})
	if err != nil {
		return nil, interestAreaProblem(err)
	}
	return &InterestAreaItemOutput{Body: interestAreaResponse(row)}, nil
}

func (h *ProgramHandler) ListMemberships(ctx context.Context, input *ListProgramMembershipsInput) (*ProgramMembershipListOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	rows, err := h.service.ListMemberships(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID))
	if err != nil {
		return nil, programProblem(err)
	}
	result := make([]ProgramMembershipResponse, 0, len(rows))
	for _, row := range rows {
		result = append(result, programMembershipResponse(row))
	}
	return &ProgramMembershipListOutput{Body: result}, nil
}

func (h *ProgramHandler) AddMembership(ctx context.Context, input *CreateProgramMembershipInput) (*ProgramMembershipOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	row, err := h.service.AddMembership(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.Body.StudentID))
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramMembershipOutput{Body: programMembershipResponse(row)}, nil
}

func (h *ProgramHandler) DeleteMembership(ctx context.Context, input *DeleteProgramMembershipInput) (*ProgramDeleteOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	if err := h.service.DeleteMembership(ctx, string(account.OrganizationID), programActor(account), ids.XID(input.SchoolYearID), ids.XID(input.ProgramID), ids.XID(input.MembershipID)); err != nil {
		return nil, programProblem(err)
	}
	return &ProgramDeleteOutput{}, nil
}

func (h *ProgramHandler) MissingGradeSummary(ctx context.Context, input *MissingGradeSummaryInput) (*ProgramRosterSummaryOutput, error) {
	account, err := programAccount(ctx)
	if err != nil {
		return nil, err
	}
	if h == nil || h.service == nil || input == nil {
		return nil, programNotFound()
	}
	count, err := h.service.CountStudentsWithoutGrade(ctx, string(account.OrganizationID), ids.XID(input.SchoolYearID))
	if err != nil {
		return nil, programProblem(err)
	}
	return &ProgramRosterSummaryOutput{Body: ProgramRosterSummaryResponse{MissingGradeCount: count}}, nil
}

func programAccount(ctx context.Context) (auth.AccountPrincipal, error) {
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
func programActor(account auth.AccountPrincipal) audit.Actor {
	id := account.UserID
	return audit.Actor{Type: audit.ActorTypeUser, UserID: &id, Label: account.Email}
}
func programResponse(row data.Program) ProgramResponse {
	return ProgramResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), Name: row.Name, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func interestAreaResponse(row data.InterestArea) InterestAreaResponse {
	return InterestAreaResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), Label: row.Label, Ordinal: row.Ordinal, RetiredAt: row.RetiredAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func programMembershipResponse(row data.ProgramMembership) ProgramMembershipResponse {
	var grade *string
	if row.GradeLevelID != nil {
		value := string(*row.GradeLevelID)
		grade = &value
	}
	return ProgramMembershipResponse{ID: string(row.ID), OrganizationID: string(row.OrganizationID), SchoolYearID: string(row.SchoolYearID), ProgramID: string(row.ProgramID), StudentID: string(row.StudentID), LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName, GradeLevelID: grade, GradeMissing: row.GradeMissing, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func programNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "program not found")
}
func interestAreaNotFound() error {
	return problems.New(http.StatusNotFound, problems.ResourceNotFound, "interest area not found")
}

func interestAreaProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return interestAreaNotFound()
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the interest-area label is already used in this program")
	case strings.Contains(err.Error(), "label is required"), errors.Is(err, programservice.ErrInterestAreaNoChanges):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change interest-area data")
	}
}
func programProblem(err error) error {
	var pgErr *pgconn.PgError
	switch {
	case errors.Is(err, pgx.ErrNoRows), strings.Contains(err.Error(), "program membership not found"):
		return programNotFound()
	case errors.Is(err, programservice.ErrStudentGradeRequired):
		return problems.New(http.StatusBadRequest, problems.ProgramStudentGradeRequired, err.Error())
	case data.IsSchoolYearClosed(err):
		return problems.New(http.StatusConflict, problems.SchoolYearClosed, "the school year is closed and cannot be changed")
	case errors.As(err, &pgErr) && pgErr.Code == "23505":
		return problems.New(http.StatusConflict, problems.ProgramConflict, "the program or membership already exists")
	case strings.Contains(err.Error(), "name is required") || strings.Contains(err.Error(), "student id"):
		return problems.New(http.StatusBadRequest, problems.ProgramConflict, err.Error())
	default:
		return problems.New(http.StatusInternalServerError, problems.InternalError, "unable to change program data")
	}
}
