// Package preference owns retained preference submissions, survey lifecycle,
// and their effective value rules.
package preference

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

var (
	ErrPreferenceServiceNil        = errors.New("preference service is nil")
	ErrRankedChoiceNotComplete     = errors.New("ranked choice response is not complete")
	ErrRankedChoiceInvalid         = errors.New("ranked choice response is invalid")
	ErrRankedChoiceNotConfigured   = errors.New("ranked-choice voting is not configured for this session")
	ErrRankedChoiceNotAccepting    = errors.New("ranked-choice voting is not accepting submissions")
	ErrRankedChoiceDeadlinePassed  = errors.New("ranked-choice voting deadline has passed")
	ErrRankedChoiceCodeRequired    = errors.New("ranked-choice student-code access requires a code")
	ErrRankedChoiceCodeInvalid     = errors.New("ranked-choice student-code access code is invalid or revoked")
	ErrRankedChoiceStudentMismatch = errors.New("ranked-choice access code is not bound to this student")
	ErrRankedChoiceStudentExcluded = errors.New("student is not participating in this session")
	ErrAccessCodeReasonRequired    = errors.New("access-code changes require a reason")
	ErrInterestAreaNotInProgram    = errors.New("interest area is not in the program")
)

type Service struct{ database *data.DB }

func New(database *data.DB) *Service { return &Service{database: database} }

type InterestProfileSubmissionInput struct {
	SchoolYearID ids.XID
	ProgramID    ids.XID
	StudentID    ids.XID
	Channel      data.PreferenceSubmissionChannel
	ActorAdultID *ids.XID
	Answers      []data.InterestProfileAnswer
}

type RankedChoiceSubmissionInput struct {
	SchoolYearID ids.XID
	ProgramID    ids.XID
	SessionID    ids.XID
	StudentID    ids.XID
	Code         string
	Channel      data.PreferenceSubmissionChannel
	ActorAdultID *ids.XID
	Responses    []data.RankedChoiceResponseInput
}

func (s *Service) SubmitInterestProfile(ctx context.Context, organizationID string, actor audit.Actor, input InterestProfileSubmissionInput) (data.InterestProfileSubmission, error) {
	if s == nil || s.database == nil {
		return data.InterestProfileSubmission{}, ErrPreferenceServiceNil
	}
	if err := validateInterestProfileInput(input); err != nil {
		return data.InterestProfileSubmission{}, err
	}
	var result data.InterestProfileSubmission
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		areas, err := tx.ListInterestAreas(ctx, input.SchoolYearID, input.ProgramID, true)
		if err != nil {
			return err
		}
		known := make(map[ids.XID]struct{}, len(areas))
		for _, area := range areas {
			known[area.ID] = struct{}{}
		}
		for _, answer := range input.Answers {
			if _, ok := known[answer.InterestAreaID]; !ok {
				return fmt.Errorf("%w: %s", ErrInterestAreaNotInProgram, answer.InterestAreaID)
			}
		}
		created, _, err := tx.CreateInterestProfileSubmission(ctx, input.SchoolYearID, input.ProgramID, input.StudentID, input.Channel, input.ActorAdultID, input.Answers)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{
			Action: audit.ActionPreferenceSubmission, ObjectType: "interest_profile_submission", ObjectID: &id, SchoolYearID: &year,
			ChangeSummary: mustJSON(map[string]any{"student_id": created.StudentID, "program_id": created.ProgramID, "channel": created.Channel, "response_count": len(input.Answers)}),
		})
	})
	if err != nil {
		return data.InterestProfileSubmission{}, fmt.Errorf("submit interest profile: %w", err)
	}
	return result, nil
}

func (s *Service) EffectiveInterestProfile(ctx context.Context, organizationID string, schoolYearID, programID, studentID ids.XID) ([]data.EffectiveInterestProfileValue, error) {
	if s == nil || s.database == nil {
		return nil, ErrPreferenceServiceNil
	}
	var result []data.EffectiveInterestProfileValue
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		result, err = tx.EffectiveInterestProfile(ctx, schoolYearID, programID, studentID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("get effective interest profile: %w", err)
	}
	return result, nil
}

func (s *Service) SubmitRankedChoices(ctx context.Context, organizationID string, actor audit.Actor, input RankedChoiceSubmissionInput) (data.RankedChoiceSubmission, error) {
	if s == nil || s.database == nil {
		return data.RankedChoiceSubmission{}, ErrPreferenceServiceNil
	}
	if err := validateRankedChoiceInput(input); err != nil {
		return data.RankedChoiceSubmission{}, err
	}
	var result data.RankedChoiceSubmission
	err := s.database.InTenant(ctx, organizationID, actor, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSessionForUpdate(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		if session.RankedChoice == nil {
			return ErrRankedChoiceNotConfigured
		}
		if session.State != data.SessionVotingOpen {
			return ErrRankedChoiceNotAccepting
		}
		now := time.Now().UTC()
		if session.RankedChoice.Deadline == nil || !now.Before(*session.RankedChoice.Deadline) {
			return ErrRankedChoiceDeadlinePassed
		}
		studentID := input.StudentID
		if input.Channel == data.PreferenceChannelStudentCode {
			if strings.TrimSpace(input.Code) == "" {
				return ErrRankedChoiceCodeRequired
			}
			resolvedStudentID, err := tx.FindActiveRankedChoiceAccessCode(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, rankedChoiceCodeHash(input.Code))
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return err
				}
				return ErrRankedChoiceCodeInvalid
			}
			if studentID != "" && studentID != resolvedStudentID {
				return ErrRankedChoiceStudentMismatch
			}
			studentID = resolvedStudentID
		}
		if studentID == "" {
			return errors.New("submit ranked choices: student id is required")
		}
		if err := ensureRankedChoiceParticipant(ctx, tx, input.SchoolYearID, input.ProgramID, input.SessionID, studentID); err != nil {
			return err
		}
		offerings, err := tx.ListOfferings(ctx, input.SchoolYearID, input.ProgramID, input.SessionID)
		if err != nil {
			return err
		}
		if err := ValidateRankedChoiceResponseSetWithDepth(input.Responses, offerings, session.RankedChoice.RankDepth); err != nil {
			return err
		}
		created, _, err := tx.CreateRankedChoiceSubmission(ctx, input.SchoolYearID, input.ProgramID, input.SessionID, studentID, input.Channel, input.ActorAdultID, input.Responses)
		if err != nil {
			return err
		}
		result = created
		id, year := created.ID, created.SchoolYearID
		return tx.Record(ctx, audit.Entry{
			Action: audit.ActionPreferenceSubmission, ObjectType: "ranked_choice_submission", ObjectID: &id, SchoolYearID: &year,
			ChangeSummary: mustJSON(map[string]any{"student_id": created.StudentID, "program_id": created.ProgramID, "session_id": created.SessionID, "channel": created.Channel, "response_count": len(input.Responses)}),
		})
	})
	if err != nil {
		return data.RankedChoiceSubmission{}, fmt.Errorf("submit ranked choices: %w", err)
	}
	return result, nil
}

func (s *Service) LatestRankedChoices(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, studentID ids.XID) (data.RankedChoiceSubmission, []data.RankedChoiceResponse, error) {
	if s == nil || s.database == nil {
		return data.RankedChoiceSubmission{}, nil, ErrPreferenceServiceNil
	}
	var submission data.RankedChoiceSubmission
	var responses []data.RankedChoiceResponse
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		var err error
		submission, err = tx.GetLatestRankedChoiceSubmission(ctx, schoolYearID, programID, sessionID, studentID)
		if err != nil {
			return err
		}
		responses, err = tx.ListRankedChoiceResponses(ctx, schoolYearID, programID, sessionID, submission.ID)
		return err
	})
	if err != nil {
		return data.RankedChoiceSubmission{}, nil, fmt.Errorf("get latest ranked choices: %w", err)
	}
	return submission, responses, nil
}

// ProfileState makes the distinction between an explicit unrated answer and
// a student with no profile submission at all observable to callers.
type ProfileState string

const (
	ProfileRated      ProfileState = "rated"
	ProfileUnrated    ProfileState = "unrated"
	ProfileNoResponse ProfileState = "no_response"
)

type ProfileValue struct {
	InterestAreaID ids.XID
	State          ProfileState
	Rating         data.InterestProfileRating
	SubmissionID   ids.XID
}

// EffectiveProfile overlays submissions in chronological order. A later
// submission only changes areas it contains; omitted areas retain their last
// value. Call ProfileValueForArea for the explicit no-response state.
func EffectiveProfile(submissions []data.InterestProfileSubmission, responses []data.InterestProfileResponse) []ProfileValue {
	ordered := append([]data.InterestProfileSubmission(nil), submissions...)
	slices.SortStableFunc(ordered, func(a, b data.InterestProfileSubmission) int {
		if a.SubmittedAt.Before(b.SubmittedAt) {
			return -1
		}
		if a.SubmittedAt.After(b.SubmittedAt) {
			return 1
		}
		return compareIDs(a.ID, b.ID)
	})
	bySubmission := make(map[ids.XID][]data.InterestProfileResponse)
	for _, response := range responses {
		bySubmission[response.SubmissionID] = append(bySubmission[response.SubmissionID], response)
	}
	latest := make(map[ids.XID]ProfileValue)
	for _, submission := range ordered {
		for _, response := range bySubmission[submission.ID] {
			state := ProfileRated
			if response.Rating == data.InterestProfileUnrated {
				state = ProfileUnrated
			}
			latest[response.InterestAreaID] = ProfileValue{InterestAreaID: response.InterestAreaID, State: state, Rating: response.Rating, SubmissionID: submission.ID}
		}
	}
	result := make([]ProfileValue, 0, len(latest))
	for _, value := range latest {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].InterestAreaID < result[j].InterestAreaID })
	return result
}

func ProfileValueForArea(areaID ids.XID, submissions []data.InterestProfileSubmission, responses []data.InterestProfileResponse) ProfileValue {
	for _, value := range EffectiveProfile(submissions, responses) {
		if value.InterestAreaID == areaID {
			return value
		}
	}
	return ProfileValue{InterestAreaID: areaID, State: ProfileNoResponse}
}

// ValidateRankedChoiceResponseSet verifies a complete response against the
// session catalog before any submission row is written. This is what prevents
// a malformed re-submission from replacing the latest valid response.
func ValidateRankedChoiceResponseSet(responses []data.RankedChoiceResponseInput, offerings []data.Offering) error {
	return ValidateRankedChoiceResponseSetWithDepth(responses, offerings, len(offerings))
}

// ValidateRankedChoiceResponseSetWithDepth verifies a complete response
// against the session catalog and its configured ranked depth before any
// submission row is written.
func ValidateRankedChoiceResponseSetWithDepth(responses []data.RankedChoiceResponseInput, offerings []data.Offering, rankDepth int) error {
	if len(offerings) == 0 {
		return fmt.Errorf("%w: the session has no offerings", ErrRankedChoiceNotComplete)
	}
	if len(responses) != len(offerings) {
		return fmt.Errorf("%w: expected one response for each of %d offerings, got %d", ErrRankedChoiceNotComplete, len(offerings), len(responses))
	}
	if rankDepth < 1 {
		return fmt.Errorf("%w: ranked-choice rank depth must be positive", ErrRankedChoiceInvalid)
	}
	known := make(map[ids.XID]struct{}, len(offerings))
	for _, offering := range offerings {
		known[offering.ID] = struct{}{}
	}
	seen := make(map[ids.XID]struct{}, len(responses))
	seenRanks := make(map[int]struct{}, len(responses))
	for _, response := range responses {
		if _, ok := known[response.OfferingID]; !ok {
			return fmt.Errorf("%w: offering %q is not in the session", ErrRankedChoiceInvalid, response.OfferingID)
		}
		if _, ok := seen[response.OfferingID]; ok {
			return fmt.Errorf("%w: offering %q is repeated", ErrRankedChoiceInvalid, response.OfferingID)
		}
		seen[response.OfferingID] = struct{}{}
		switch response.Answer {
		case data.RankedChoiceRanked:
			if response.Rank == nil || *response.Rank < 1 {
				return fmt.Errorf("%w: ranked answer requires a positive rank", ErrRankedChoiceInvalid)
			}
			maxRank := rankDepth
			if maxRank > len(offerings) {
				maxRank = len(offerings)
			}
			if *response.Rank > maxRank {
				return fmt.Errorf("%w: rank %d exceeds the number of offerings", ErrRankedChoiceInvalid, *response.Rank)
			}
			if _, ok := seenRanks[*response.Rank]; ok {
				return fmt.Errorf("%w: rank %d is repeated", ErrRankedChoiceInvalid, *response.Rank)
			}
			seenRanks[*response.Rank] = struct{}{}
		case data.RankedChoiceInterested, data.RankedChoiceNotInterested, data.RankedChoiceNoResponse:
			if response.Rank != nil {
				return fmt.Errorf("%w: %q cannot include a rank", ErrRankedChoiceInvalid, response.Answer)
			}
		default:
			return fmt.Errorf("%w: answer %q is invalid", ErrRankedChoiceInvalid, response.Answer)
		}
	}
	if len(seen) != len(known) {
		return fmt.Errorf("%w: one or more offerings are missing", ErrRankedChoiceNotComplete)
	}
	return nil
}

func validateInterestProfileInput(input InterestProfileSubmissionInput) error {
	if input.SchoolYearID == "" || input.ProgramID == "" || input.StudentID == "" {
		return errors.New("submit interest profile: school year, program, and student ids are required")
	}
	if len(input.Answers) == 0 {
		return errors.New("submit interest profile: at least one area response is required")
	}
	return nil
}

func validateRankedChoiceInput(input RankedChoiceSubmissionInput) error {
	if input.SchoolYearID == "" || input.ProgramID == "" || input.SessionID == "" {
		return errors.New("submit ranked choices: school year, program, and session ids are required")
	}
	if input.Channel != data.PreferenceChannelStudentCode && input.StudentID == "" {
		return errors.New("submit ranked choices: student id is required")
	}
	return nil
}

func ensureRankedChoiceParticipant(ctx context.Context, tx *data.Tx, schoolYearID, programID, sessionID, studentID ids.XID) error {
	memberships, err := tx.ListProgramMemberships(ctx, schoolYearID, programID)
	if err != nil {
		return err
	}
	found := false
	for _, membership := range memberships {
		if membership.StudentID == studentID {
			found = true
			break
		}
	}
	if !found {
		return ErrRankedChoiceStudentExcluded
	}
	nonParticipations, err := tx.ListSessionNonParticipations(ctx, schoolYearID, programID, sessionID)
	if err != nil {
		return err
	}
	for _, row := range nonParticipations {
		if row.StudentID == studentID {
			return ErrRankedChoiceStudentExcluded
		}
	}
	return nil
}

func compareIDs(a, b ids.XID) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func mustJSON(value any) json.RawMessage {
	result, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return result
}
