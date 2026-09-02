package preference

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"context"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

const (
	SurveyWarningEmptyAudience = "empty_audience"
	SurveyWarningReopened      = "reopened_with_warning"
)

var (
	ErrSurveyDefinitionLocked        = errors.New("interest profile survey definition is locked after opening")
	ErrSurveyClosingTimeRequired     = errors.New("interest profile survey requires a closing timestamp")
	ErrSurveyClosingTimeInvalid      = errors.New("interest profile survey closing timestamp must be after its opening timestamp and in the future")
	ErrSurveyTransitionInvalid       = errors.New("interest profile survey lifecycle transition is invalid")
	ErrSurveyHasSubmissions          = errors.New("interest profile survey cannot be deleted after submissions exist")
	ErrSurveyQuestionRequired        = errors.New("interest profile survey requires at least one question")
	ErrSurveyScaleRequired           = errors.New("interest profile survey requires at least one scale option")
	ErrSurveyAudienceInvalid         = errors.New("interest profile survey audience filter is invalid")
	ErrSurveyNotAcceptingSubmissions = errors.New("interest profile survey is not accepting submissions")
	ErrSurveyCodeInvalid             = errors.New("interest profile survey access code is invalid or revoked")
)

type InterestProfileSurveyQuestionInput struct {
	InterestAreaID ids.XID
}

type InterestProfileSurveyScaleOptionInput struct {
	Value   string
	Label   string
	Ordinal int
}

type InterestProfileSurveyAudienceInput struct {
	Type          data.InterestProfileSurveyAudienceType
	StudentIDs    []ids.XID
	GradeLevelID  *ids.XID
	PriorSurveyID *ids.XID
	ResponseState *data.InterestProfileSurveyResponseState
}

type InterestProfileSurveyInput struct {
	Name         string
	Introduction string
	OpensAt      *time.Time
	ClosesAt     *time.Time
	Audience     InterestProfileSurveyAudienceInput
	ScaleVersion string
	Questions    []InterestProfileSurveyQuestionInput
	ScaleOptions []InterestProfileSurveyScaleOptionInput
}

type InterestProfileSurveyUpdate struct {
	InterestProfileSurveyInput
}

type InterestProfileSurveyTransitionInput struct {
	State           data.InterestProfileSurveyState
	ClosingAt       *time.Time
	RegenerateCodes bool
	Reason          string
}

type SurveyAccessCode struct {
	StudentID ids.XID
	Code      string
}

type InterestProfileSurveyView struct {
	Survey             data.InterestProfileSurvey
	DefinitionStudents []data.InterestProfileSurveyAudienceStudent
	Questions          []data.InterestProfileSurveyQuestion
	ScaleOptions       []data.InterestProfileSurveyScaleOption
	AudienceSnapshot   []data.InterestProfileSurveyAudienceSnapshot
	ActiveCodes        []data.InterestProfileSurveyAccessCode
}

type InterestProfileSurveyTransitionResult struct {
	Survey      InterestProfileSurveyView
	Warnings    []string
	AccessCodes []SurveyAccessCode
}

var defaultInterestProfileSurveyScale = []InterestProfileSurveyScaleOptionInput{
	{Value: string(data.InterestProfileVeryInterested), Label: "Very interested", Ordinal: 1},
	{Value: string(data.InterestProfileInterested), Label: "Interested", Ordinal: 2},
	{Value: string(data.InterestProfileNotInterested), Label: "Not interested", Ordinal: 3},
}

func (s *Service) CreateInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID ids.XID, input InterestProfileSurveyInput) (InterestProfileSurveyView, error) {
	if s == nil || s.database == nil {
		return InterestProfileSurveyView{}, ErrPreferenceServiceNil
	}
	var result InterestProfileSurveyView
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		if _, err := tx.GetProgram(ctx, schoolYearID, programID); err != nil {
			return err
		}
		questions, err := surveyQuestions(ctx, tx, schoolYearID, programID, input.Questions)
		if err != nil {
			return err
		}
		options, err := surveyScaleOptions(input.ScaleOptions)
		if err != nil {
			return err
		}
		audience, err := validateSurveyAudience(input.Audience)
		if err != nil {
			return err
		}
		if audience.PriorSurveyID != nil {
			prior, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, *audience.PriorSurveyID)
			if err != nil {
				return err
			}
			if prior.ID == "" {
				return ErrSurveyAudienceInvalid
			}
		}
		created, err := tx.CreateInterestProfileSurvey(ctx, schoolYearID, programID, input.Name, input.Introduction, input.OpensAt, input.ClosesAt, audience.Type, audience.GradeLevelID, audience.PriorSurveyID, audience.ResponseState, input.ScaleVersion)
		if err != nil {
			return err
		}
		if err := replaceSurveyDefinition(ctx, tx, created, audience.StudentIDs, questions, options); err != nil {
			return err
		}
		result, err = surveyView(ctx, tx, created)
		if err != nil {
			return err
		}
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyDefinitionChange, ObjectType: "interest_profile_survey", ObjectID: &id, SchoolYearID: &year, ChangeSummary: surveySummary(created)})
	})
	if err != nil {
		return InterestProfileSurveyView{}, fmt.Errorf("create interest profile survey: %w", err)
	}
	return result, nil
}

func (s *Service) ListInterestProfileSurveys(ctx context.Context, organizationID string, schoolYearID, programID ids.XID) ([]InterestProfileSurveyView, error) {
	if s == nil || s.database == nil {
		return nil, ErrPreferenceServiceNil
	}
	var result []InterestProfileSurveyView
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		surveys, err := tx.ListInterestProfileSurveys(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result = make([]InterestProfileSurveyView, 0, len(surveys))
		for _, survey := range surveys {
			view, err := surveyView(ctx, tx, survey)
			if err != nil {
				return err
			}
			result = append(result, view)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list interest profile surveys: %w", err)
	}
	return result, nil
}

func (s *Service) GetInterestProfileSurvey(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID) (InterestProfileSurveyView, error) {
	if s == nil || s.database == nil {
		return InterestProfileSurveyView{}, ErrPreferenceServiceNil
	}
	var result InterestProfileSurveyView
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		result, err = surveyView(ctx, tx, survey)
		return err
	})
	if err != nil {
		return InterestProfileSurveyView{}, fmt.Errorf("get interest profile survey: %w", err)
	}
	return result, nil
}

func (s *Service) UpdateInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, input InterestProfileSurveyUpdate) (InterestProfileSurveyView, error) {
	if s == nil || s.database == nil {
		return InterestProfileSurveyView{}, ErrPreferenceServiceNil
	}
	var result InterestProfileSurveyView
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		if effectiveSurveyState(current, time.Now().UTC()) != data.InterestProfileSurveyDraft {
			return ErrSurveyDefinitionLocked
		}
		questions, err := surveyQuestions(ctx, tx, schoolYearID, programID, input.Questions)
		if err != nil {
			return err
		}
		options, err := surveyScaleOptions(input.ScaleOptions)
		if err != nil {
			return err
		}
		audience, err := validateSurveyAudience(input.Audience)
		if err != nil {
			return err
		}
		if audience.PriorSurveyID != nil {
			prior, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, *audience.PriorSurveyID)
			if err != nil {
				return err
			}
			if prior.ID == "" || prior.ID == surveyID {
				return ErrSurveyAudienceInvalid
			}
		}
		updated, err := tx.UpdateInterestProfileSurvey(ctx, schoolYearID, programID, surveyID, input.Name, input.Introduction, input.OpensAt, input.ClosesAt, audience.Type, audience.GradeLevelID, audience.PriorSurveyID, audience.ResponseState, input.ScaleVersion)
		if err != nil {
			return err
		}
		if err := replaceSurveyDefinition(ctx, tx, updated, audience.StudentIDs, questions, options); err != nil {
			return err
		}
		result, err = surveyView(ctx, tx, updated)
		if err != nil {
			return err
		}
		year := updated.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyDefinitionChange, ObjectType: "interest_profile_survey", ObjectID: &surveyID, SchoolYearID: &year, ChangeSummary: surveySummary(updated)})
	})
	if err != nil {
		return InterestProfileSurveyView{}, fmt.Errorf("update interest profile survey: %w", err)
	}
	return result, nil
}

func (s *Service) DeleteInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID) error {
	if s == nil || s.database == nil {
		return ErrPreferenceServiceNil
	}
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		count, err := tx.CountInterestProfileSurveySubmissions(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		if count > 0 {
			return ErrSurveyHasSubmissions
		}
		deleted, err := tx.DeleteInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		if !deleted {
			return errors.New("interest profile survey not found")
		}
		year := survey.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyDefinitionChange, ObjectType: "interest_profile_survey", ObjectID: &surveyID, SchoolYearID: &year, Reason: "organizer deleted a survey without submissions", ChangeSummary: json.RawMessage(`{"deleted":true}`)})
	})
	if err != nil {
		return fmt.Errorf("delete interest profile survey: %w", err)
	}
	return nil
}

func (s *Service) TransitionInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, input InterestProfileSurveyTransitionInput) (InterestProfileSurveyTransitionResult, error) {
	if s == nil || s.database == nil {
		return InterestProfileSurveyTransitionResult{}, ErrPreferenceServiceNil
	}
	var result InterestProfileSurveyTransitionResult
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		current, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		logicalState := effectiveSurveyState(current, now)
		if input.State == "" {
			return ErrSurveyTransitionInvalid
		}
		if logicalState == data.InterestProfileSurveyDraft && input.State == data.InterestProfileSurveyOpen {
			return openSurvey(ctx, tx, current, input, now, &result)
		}
		if logicalState == data.InterestProfileSurveyOpen && input.State == data.InterestProfileSurveyClosed {
			updated, err := tx.SetInterestProfileSurveyState(ctx, schoolYearID, programID, surveyID, data.InterestProfileSurveyClosed, current.OpensAt, current.ClosesAt, current.OpenedAt)
			if err != nil {
				return err
			}
			result.Survey, err = surveyView(ctx, tx, updated)
			if err != nil {
				return err
			}
			year := updated.SchoolYearID
			return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyLifecycle, ObjectType: "interest_profile_survey", ObjectID: &surveyID, SchoolYearID: &year, ChangeSummary: surveyLifecycleSummary(current, updated), Reason: strings.TrimSpace(input.Reason)})
		}
		if logicalState == data.InterestProfileSurveyClosed && input.State == data.InterestProfileSurveyOpen {
			if strings.TrimSpace(input.Reason) == "" {
				return errors.New("reopening an interest profile survey requires a reason")
			}
			closingAt := input.ClosingAt
			if closingAt == nil {
				return ErrSurveyClosingTimeRequired
			}
			if !closingAt.After(now) {
				return ErrSurveyClosingTimeInvalid
			}
			updated, err := tx.SetInterestProfileSurveyState(ctx, schoolYearID, programID, surveyID, data.InterestProfileSurveyOpen, &now, closingAt, current.OpenedAt)
			if err != nil {
				return err
			}
			if input.RegenerateCodes {
				codes, err := regenerateCodes(ctx, tx, updated)
				if err != nil {
					return err
				}
				result.AccessCodes = codes
			}
			result.Warnings = append(result.Warnings, SurveyWarningReopened)
			result.Survey, err = surveyView(ctx, tx, updated)
			if err != nil {
				return err
			}
			year := updated.SchoolYearID
			return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyLifecycle, ObjectType: "interest_profile_survey", ObjectID: &surveyID, SchoolYearID: &year, Reason: strings.TrimSpace(input.Reason), ChangeSummary: surveyLifecycleSummary(current, updated)})
		}
		return ErrSurveyTransitionInvalid
	})
	if err != nil {
		return InterestProfileSurveyTransitionResult{}, fmt.Errorf("transition interest profile survey: %w", err)
	}
	return result, nil
}

func (s *Service) RegenerateInterestProfileSurveyCodes(ctx context.Context, organizationID string, actor audit.Actor, schoolYearID, programID, surveyID ids.XID, reason string) ([]SurveyAccessCode, error) {
	if s == nil || s.database == nil {
		return nil, ErrPreferenceServiceNil
	}
	var result []SurveyAccessCode
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		if effectiveSurveyState(survey, time.Now().UTC()) == data.InterestProfileSurveyDraft {
			return ErrSurveyTransitionInvalid
		}
		result, err = regenerateCodes(ctx, tx, survey)
		if err != nil {
			return err
		}
		year := survey.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyCodeChange, ObjectType: "interest_profile_survey_access_code", ObjectID: &surveyID, SchoolYearID: &year, Reason: strings.TrimSpace(reason), ChangeSummary: mustJSON(map[string]any{"regenerated": len(result)})})
	})
	if err != nil {
		return nil, fmt.Errorf("regenerate interest profile survey codes: %w", err)
	}
	return result, nil
}

func (s *Service) SubmitInterestProfileSurvey(ctx context.Context, organizationID string, actor audit.Actor, input InterestProfileSurveySubmissionInput) (data.InterestProfileSubmission, error) {
	if s == nil || s.database == nil {
		return data.InterestProfileSubmission{}, ErrPreferenceServiceNil
	}
	if strings.TrimSpace(input.Code) == "" {
		return data.InterestProfileSubmission{}, ErrSurveyCodeInvalid
	}
	var result data.InterestProfileSubmission
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, input.SchoolYearID, input.ProgramID, input.SurveyID)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if effectiveSurveyState(survey, now) != data.InterestProfileSurveyOpen || (survey.OpensAt != nil && now.Before(*survey.OpensAt)) || survey.ClosesAt == nil || !now.Before(*survey.ClosesAt) {
			return ErrSurveyNotAcceptingSubmissions
		}
		studentID, err := tx.FindActiveInterestProfileSurveyAccessCode(ctx, input.SchoolYearID, input.ProgramID, input.SurveyID, surveyCodeHash(input.Code))
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return ErrSurveyCodeInvalid
		}
		areas, err := tx.ListInterestProfileSurveyQuestions(ctx, input.SchoolYearID, input.ProgramID, input.SurveyID)
		if err != nil {
			return err
		}
		known := make(map[ids.XID]struct{}, len(areas))
		for _, area := range areas {
			known[area.InterestAreaID] = struct{}{}
		}
		for _, answer := range input.Answers {
			if _, ok := known[answer.InterestAreaID]; !ok {
				return fmt.Errorf("%w: %s", ErrInterestAreaNotInProgram, answer.InterestAreaID)
			}
		}
		result, _, err = tx.CreateInterestProfileSurveySubmission(ctx, input.SchoolYearID, input.ProgramID, input.SurveyID, studentID, input.Channel, input.ActorAdultID, input.Answers)
		if err != nil {
			return err
		}
		id, year := result.ID, result.SchoolYearID
		return tx.Record(ctx, audit.Entry{Action: audit.ActionPreferenceSubmission, ObjectType: "interest_profile_submission", ObjectID: &id, SchoolYearID: &year, ChangeSummary: mustJSON(map[string]any{"student_id": result.StudentID, "survey_id": result.SurveyID, "channel": result.Channel, "response_count": len(input.Answers)})})
	})
	if err != nil {
		return data.InterestProfileSubmission{}, fmt.Errorf("submit interest profile survey: %w", err)
	}
	return result, nil
}

type InterestProfileSurveySubmissionInput struct {
	SchoolYearID ids.XID
	ProgramID    ids.XID
	SurveyID     ids.XID
	Code         string
	Channel      data.PreferenceSubmissionChannel
	ActorAdultID *ids.XID
	Answers      []data.InterestProfileAnswer
}

func openSurvey(ctx context.Context, tx *data.Tx, current data.InterestProfileSurvey, input InterestProfileSurveyTransitionInput, now time.Time, result *InterestProfileSurveyTransitionResult) error {
	closingAt := current.ClosesAt
	if input.ClosingAt != nil {
		closingAt = input.ClosingAt
	}
	if closingAt == nil {
		return ErrSurveyClosingTimeRequired
	}
	if !closingAt.After(now) {
		return ErrSurveyClosingTimeInvalid
	}
	openingAt := current.OpensAt
	if openingAt == nil {
		openingAt = &now
	}
	if !closingAt.After(*openingAt) {
		return ErrSurveyClosingTimeInvalid
	}
	questions, err := tx.ListInterestProfileSurveyQuestions(ctx, current.SchoolYearID, current.ProgramID, current.ID)
	if err != nil {
		return err
	}
	if len(questions) == 0 {
		return ErrSurveyQuestionRequired
	}
	options, err := tx.ListInterestProfileSurveyScaleOptions(ctx, current.SchoolYearID, current.ProgramID, current.ID)
	if err != nil {
		return err
	}
	if len(options) == 0 {
		return ErrSurveyScaleRequired
	}
	audience, err := snapshotAudience(ctx, tx, current)
	if err != nil {
		return err
	}
	for _, studentID := range audience {
		if _, err := tx.CreateInterestProfileSurveyAudienceSnapshot(ctx, current.SchoolYearID, current.ProgramID, current.ID, studentID); err != nil {
			return err
		}
	}
	updated, err := tx.SetInterestProfileSurveyState(ctx, current.SchoolYearID, current.ProgramID, current.ID, data.InterestProfileSurveyOpen, openingAt, closingAt, &now)
	if err != nil {
		return err
	}
	accessCodes, err := issueCodes(ctx, tx, updated, audience)
	if err != nil {
		return err
	}
	result.AccessCodes = accessCodes
	if len(audience) == 0 {
		result.Warnings = append(result.Warnings, SurveyWarningEmptyAudience)
	}
	result.Survey, err = surveyView(ctx, tx, updated)
	if err != nil {
		return err
	}
	year := updated.SchoolYearID
	return tx.Record(ctx, audit.Entry{Action: audit.ActionSurveyLifecycle, ObjectType: "interest_profile_survey", ObjectID: &current.ID, SchoolYearID: &year, ChangeSummary: surveyLifecycleSummary(current, updated)})
}

func snapshotAudience(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey) ([]ids.XID, error) {
	members, err := tx.ListInterestProfileSurveyMembers(ctx, survey.SchoolYearID, survey.ProgramID)
	if err != nil {
		return nil, err
	}
	allowed := make(map[ids.XID]struct{}, len(members))
	switch survey.AudienceType {
	case data.SurveyAudienceAllMembers:
		for _, member := range members {
			allowed[member.StudentID] = struct{}{}
		}
	case data.SurveyAudienceExplicitStudents:
		students, err := tx.ListInterestProfileSurveyDefinitionStudents(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
		if err != nil {
			return nil, err
		}
		for _, student := range students {
			allowed[student.StudentID] = struct{}{}
		}
	case data.SurveyAudienceGradeLevel:
		for _, member := range members {
			if survey.AudienceGradeLevelID != nil && member.GradeLevelID != nil && *survey.AudienceGradeLevelID == *member.GradeLevelID {
				allowed[member.StudentID] = struct{}{}
			}
		}
	case data.SurveyAudienceResponseState:
		if survey.AudiencePriorSurveyID == nil || survey.AudienceResponseState == nil {
			return nil, ErrSurveyAudienceInvalid
		}
		responders, err := tx.ListInterestProfileSurveyPriorResponders(ctx, survey.SchoolYearID, survey.ProgramID, *survey.AudiencePriorSurveyID)
		if err != nil {
			return nil, err
		}
		responded := make(map[ids.XID]struct{}, len(responders))
		for _, studentID := range responders {
			responded[studentID] = struct{}{}
		}
		for _, member := range members {
			_, hasResponded := responded[member.StudentID]
			if (*survey.AudienceResponseState == data.SurveyResponded && hasResponded) || (*survey.AudienceResponseState == data.SurveyNotResponded && !hasResponded) {
				allowed[member.StudentID] = struct{}{}
			}
		}
	default:
		return nil, ErrSurveyAudienceInvalid
	}
	result := make([]ids.XID, 0, len(allowed))
	for _, member := range members {
		if _, ok := allowed[member.StudentID]; ok {
			result = append(result, member.StudentID)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result, nil
}

func issueCodes(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey, studentIDs []ids.XID) ([]SurveyAccessCode, error) {
	result := make([]SurveyAccessCode, 0, len(studentIDs))
	for _, studentID := range studentIDs {
		code, err := newSurveyCode()
		if err != nil {
			return nil, err
		}
		if _, err := tx.CreateInterestProfileSurveyAccessCode(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID, studentID, surveyCodeHash(code)); err != nil {
			return nil, err
		}
		result = append(result, SurveyAccessCode{StudentID: studentID, Code: code})
	}
	return result, nil
}

func regenerateCodes(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey) ([]SurveyAccessCode, error) {
	if _, err := tx.RevokeInterestProfileSurveyAccessCodes(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID); err != nil {
		return nil, err
	}
	snapshots, err := tx.ListInterestProfileSurveyAudienceSnapshot(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return nil, err
	}
	students := make([]ids.XID, 0, len(snapshots))
	for _, snapshot := range snapshots {
		students = append(students, snapshot.StudentID)
	}
	return issueCodes(ctx, tx, survey, students)
}

func newSurveyCode() (string, error) {
	buffer := make([]byte, 24)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate interest profile survey access code: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func surveyCodeHash(code string) string {
	digest := sha256.Sum256([]byte(code))
	return hex.EncodeToString(digest[:])
}

func surveyView(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey) (InterestProfileSurveyView, error) {
	survey.State = effectiveSurveyState(survey, time.Now().UTC())
	definitionStudents, err := tx.ListInterestProfileSurveyDefinitionStudents(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return InterestProfileSurveyView{}, err
	}
	questions, err := tx.ListInterestProfileSurveyQuestions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return InterestProfileSurveyView{}, err
	}
	options, err := tx.ListInterestProfileSurveyScaleOptions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return InterestProfileSurveyView{}, err
	}
	audienceSnapshot, err := tx.ListInterestProfileSurveyAudienceSnapshot(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return InterestProfileSurveyView{}, err
	}
	activeCodes, err := tx.ListActiveInterestProfileSurveyAccessCodes(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return InterestProfileSurveyView{}, err
	}
	return InterestProfileSurveyView{Survey: survey, DefinitionStudents: definitionStudents, Questions: questions, ScaleOptions: options, AudienceSnapshot: audienceSnapshot, ActiveCodes: activeCodes}, nil
}

func effectiveSurveyState(survey data.InterestProfileSurvey, now time.Time) data.InterestProfileSurveyState {
	if survey.State == data.InterestProfileSurveyOpen && survey.ClosesAt != nil && !now.Before(*survey.ClosesAt) {
		return data.InterestProfileSurveyClosed
	}
	return survey.State
}

func validateSurveyAudience(input InterestProfileSurveyAudienceInput) (InterestProfileSurveyAudienceInput, error) {
	if input.Type == "" {
		input.Type = data.SurveyAudienceAllMembers
	}
	seen := make(map[ids.XID]struct{}, len(input.StudentIDs))
	for _, studentID := range input.StudentIDs {
		if studentID == "" {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
		if _, ok := seen[studentID]; ok {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
		seen[studentID] = struct{}{}
	}
	switch input.Type {
	case data.SurveyAudienceAllMembers, data.SurveyAudienceExplicitStudents:
		if input.GradeLevelID != nil || input.PriorSurveyID != nil || input.ResponseState != nil {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
		if input.Type == data.SurveyAudienceAllMembers && len(input.StudentIDs) > 0 {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
	case data.SurveyAudienceGradeLevel:
		if input.GradeLevelID == nil || input.PriorSurveyID != nil || input.ResponseState != nil || len(input.StudentIDs) > 0 {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
	case data.SurveyAudienceResponseState:
		if input.GradeLevelID != nil || input.PriorSurveyID == nil || input.ResponseState == nil || len(input.StudentIDs) > 0 || (*input.ResponseState != data.SurveyResponded && *input.ResponseState != data.SurveyNotResponded) {
			return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
		}
	default:
		return InterestProfileSurveyAudienceInput{}, ErrSurveyAudienceInvalid
	}
	sort.Slice(input.StudentIDs, func(i, j int) bool { return input.StudentIDs[i] < input.StudentIDs[j] })
	return input, nil
}

func surveyQuestions(ctx context.Context, tx *data.Tx, schoolYearID, programID ids.XID, inputs []InterestProfileSurveyQuestionInput) ([]data.InterestProfileSurveyQuestionInput, error) {
	if len(inputs) == 0 {
		return []data.InterestProfileSurveyQuestionInput{}, nil
	}
	areas, err := tx.ListInterestAreas(ctx, schoolYearID, programID, true)
	if err != nil {
		return nil, err
	}
	byID := make(map[ids.XID]data.InterestArea, len(areas))
	for _, area := range areas {
		if area.RetiredAt == nil {
			byID[area.ID] = area
		}
	}
	result := make([]data.InterestProfileSurveyQuestionInput, 0, len(inputs))
	seen := make(map[ids.XID]struct{}, len(inputs))
	for index, input := range inputs {
		area, ok := byID[input.InterestAreaID]
		if !ok {
			return nil, fmt.Errorf("survey question interest area %q is not active in this program", input.InterestAreaID)
		}
		if _, ok := seen[input.InterestAreaID]; ok {
			return nil, fmt.Errorf("survey question interest area %q is repeated", input.InterestAreaID)
		}
		seen[input.InterestAreaID] = struct{}{}
		result = append(result, data.InterestProfileSurveyQuestionInput{InterestAreaID: area.ID, Ordinal: index + 1, Label: area.Label})
	}
	return result, nil
}

func surveyScaleOptions(inputs []InterestProfileSurveyScaleOptionInput) ([]data.InterestProfileSurveyScaleOptionInput, error) {
	if len(inputs) == 0 {
		inputs = defaultInterestProfileSurveyScale
	}
	seenValues := make(map[string]struct{}, len(inputs))
	seenOrdinals := make(map[int]struct{}, len(inputs))
	result := make([]data.InterestProfileSurveyScaleOptionInput, 0, len(inputs))
	for index, input := range inputs {
		value, label := strings.TrimSpace(input.Value), strings.TrimSpace(input.Label)
		ordinal := input.Ordinal
		if ordinal == 0 {
			ordinal = index + 1
		}
		if value == "" || label == "" || ordinal < 1 {
			return nil, ErrSurveyScaleRequired
		}
		if _, ok := seenValues[value]; ok {
			return nil, fmt.Errorf("survey scale value %q is repeated", value)
		}
		if _, ok := seenOrdinals[ordinal]; ok {
			return nil, fmt.Errorf("survey scale ordinal %d is repeated", ordinal)
		}
		seenValues[value], seenOrdinals[ordinal] = struct{}{}, struct{}{}
		result = append(result, data.InterestProfileSurveyScaleOptionInput{Value: value, Label: label, Ordinal: ordinal})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Ordinal < result[j].Ordinal })
	return result, nil
}

func replaceSurveyDefinition(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey, studentIDs []ids.XID, questions []data.InterestProfileSurveyQuestionInput, options []data.InterestProfileSurveyScaleOptionInput) error {
	if err := tx.DeleteInterestProfileSurveyDefinitionStudents(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID); err != nil {
		return err
	}
	for _, studentID := range studentIDs {
		if _, err := tx.CreateInterestProfileSurveyDefinitionStudent(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID, studentID); err != nil {
			return err
		}
	}
	if err := tx.DeleteInterestProfileSurveyQuestions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID); err != nil {
		return err
	}
	for _, question := range questions {
		if _, err := tx.CreateInterestProfileSurveyQuestion(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID, question); err != nil {
			return err
		}
	}
	if err := tx.DeleteInterestProfileSurveyScaleOptions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID); err != nil {
		return err
	}
	for _, option := range options {
		if _, err := tx.CreateInterestProfileSurveyScaleOption(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID, option); err != nil {
			return err
		}
	}
	return nil
}

func surveySummary(survey data.InterestProfileSurvey) json.RawMessage {
	return mustJSON(map[string]any{"name": survey.Name, "program_id": survey.ProgramID, "audience_type": survey.AudienceType, "scale_version": survey.ScaleVersion})
}

func surveyLifecycleSummary(from, to data.InterestProfileSurvey) json.RawMessage {
	return mustJSON(map[string]any{"from": from.State, "to": to.State, "closes_at": to.ClosesAt})
}
