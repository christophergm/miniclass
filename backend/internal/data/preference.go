package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// PreferenceSubmissionChannel identifies the narrow access path used for a
// response. It is stored with each retained submission, not inferred later.
type PreferenceSubmissionChannel string

const (
	PreferenceChannelGuardian              PreferenceSubmissionChannel = "guardian"
	PreferenceChannelStudentCode           PreferenceSubmissionChannel = "student_code"
	PreferenceChannelAdministratorOnBehalf PreferenceSubmissionChannel = "administrator_on_behalf"
)

// InterestProfileRating includes the explicit unrated state. No response is
// represented by the absence of a submission, rather than by another value.
type InterestProfileRating string

const (
	InterestProfileVeryInterested InterestProfileRating = "very_interested"
	InterestProfileInterested     InterestProfileRating = "interested"
	InterestProfileNotInterested  InterestProfileRating = "not_interested"
	InterestProfileUnrated        InterestProfileRating = "unrated"
)

// RankedChoiceAnswer is one response to one offering in a submitted catalog.
type RankedChoiceAnswer string

const (
	RankedChoiceRanked        RankedChoiceAnswer = "ranked"
	RankedChoiceInterested    RankedChoiceAnswer = "interested"
	RankedChoiceNotInterested RankedChoiceAnswer = "not_interested"
	RankedChoiceNoResponse    RankedChoiceAnswer = "no_response"
)

// InterestProfileAnswer is intentionally keyed by the opaque area ID.
type InterestProfileAnswer struct {
	InterestAreaID ids.XID
	Rating         InterestProfileRating
}

// RankedChoiceResponse is intentionally keyed by the opaque offering ID.
type RankedChoiceResponseInput struct {
	OfferingID ids.XID
	Answer     RankedChoiceAnswer
	Rank       *int
}

type InterestProfileSubmission struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	StudentID      ids.XID
	Channel        PreferenceSubmissionChannel
	ActorType      audit.ActorType
	ActorUserID    *ids.XID
	ActorAdultID   *ids.XID
	ActorLabel     string
	SubmittedAt    time.Time
	CreatedAt      time.Time
}

type InterestProfileResponse struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SubmissionID   ids.XID
	InterestAreaID ids.XID
	Rating         InterestProfileRating
	CreatedAt      time.Time
}

type EffectiveInterestProfileValue struct {
	InterestAreaID ids.XID
	Rating         InterestProfileRating
	SubmissionID   ids.XID
	SubmittedAt    time.Time
}

type RankedChoiceSubmission struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	StudentID      ids.XID
	Channel        PreferenceSubmissionChannel
	ActorType      audit.ActorType
	ActorUserID    *ids.XID
	ActorAdultID   *ids.XID
	ActorLabel     string
	SubmittedAt    time.Time
	CreatedAt      time.Time
}

type RankedChoiceResponse struct {
	ID             ids.XID
	OrganizationID ids.XID
	SchoolYearID   ids.XID
	ProgramID      ids.XID
	SessionID      ids.XID
	SubmissionID   ids.XID
	OfferingID     ids.XID
	Answer         RankedChoiceAnswer
	Rank           *int
	CreatedAt      time.Time
}

func (tx *Tx) CreateInterestProfileSubmission(ctx context.Context, schoolYearID, programID, studentID ids.XID, channel PreferenceSubmissionChannel, actorAdultID *ids.XID, answers []InterestProfileAnswer) (InterestProfileSubmission, []InterestProfileResponse, error) {
	if err := validateSubmissionAttribution(tx.actor, channel, actorAdultID); err != nil {
		return InterestProfileSubmission{}, nil, err
	}
	if len(answers) == 0 {
		return InterestProfileSubmission{}, nil, errors.New("create interest profile submission: at least one area response is required")
	}
	if err := validateInterestProfileAnswers(answers); err != nil {
		return InterestProfileSubmission{}, nil, err
	}
	row, err := tx.queries.CreateInterestProfileSubmission(ctx, db.CreateInterestProfileSubmissionParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, StudentID: studentID,
		Channel: db.PreferenceSubmissionChannel(channel), ActorType: db.AuditActorType(tx.actor.Type), ActorUserID: tx.actor.UserID,
		ActorAdultID: actorAdultID, ActorLabel: strings.TrimSpace(tx.actor.Label),
	})
	if err != nil {
		return InterestProfileSubmission{}, nil, fmt.Errorf("create interest profile submission: %w", err)
	}
	submission, err := interestProfileSubmission(row)
	if err != nil {
		return InterestProfileSubmission{}, nil, err
	}
	responses := make([]InterestProfileResponse, 0, len(answers))
	for _, answer := range answers {
		created, err := tx.queries.CreateInterestProfileResponse(ctx, db.CreateInterestProfileResponseParams{
			OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID,
			SubmissionID: submission.ID, InterestAreaID: answer.InterestAreaID, Response: db.InterestProfileRating(answer.Rating),
		})
		if err != nil {
			return InterestProfileSubmission{}, nil, fmt.Errorf("create interest profile response: %w", err)
		}
		value, err := interestProfileResponse(created)
		if err != nil {
			return InterestProfileSubmission{}, nil, err
		}
		responses = append(responses, value)
	}
	return submission, responses, nil
}

func (tx *Tx) ListInterestProfileSubmissions(ctx context.Context, schoolYearID, programID, studentID ids.XID) ([]InterestProfileSubmission, error) {
	rows, err := tx.queries.ListInterestProfileSubmissions(ctx, db.ListInterestProfileSubmissionsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, StudentID: studentID,
	})
	if err != nil {
		return nil, fmt.Errorf("list interest profile submissions: %w", err)
	}
	result := make([]InterestProfileSubmission, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSubmission(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) ListInterestProfileResponses(ctx context.Context, schoolYearID, programID, submissionID ids.XID) ([]InterestProfileResponse, error) {
	rows, err := tx.queries.ListInterestProfileResponses(ctx, db.ListInterestProfileResponsesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SubmissionID: submissionID,
	})
	if err != nil {
		return nil, fmt.Errorf("list interest profile responses: %w", err)
	}
	result := make([]InterestProfileResponse, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileResponse(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) EffectiveInterestProfile(ctx context.Context, schoolYearID, programID, studentID ids.XID) ([]EffectiveInterestProfileValue, error) {
	rows, err := tx.queries.GetEffectiveInterestProfile(ctx, db.GetEffectiveInterestProfileParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, StudentID: studentID,
	})
	if err != nil {
		return nil, fmt.Errorf("get effective interest profile: %w", err)
	}
	result := make([]EffectiveInterestProfileValue, 0, len(rows))
	for _, row := range rows {
		submittedAt, err := programTime(row.SubmittedAt, "submitted_at")
		if err != nil {
			return nil, err
		}
		result = append(result, EffectiveInterestProfileValue{
			InterestAreaID: row.InterestAreaID, Rating: InterestProfileRating(row.Response), SubmissionID: row.SubmissionID, SubmittedAt: submittedAt,
		})
	}
	return result, nil
}

func (tx *Tx) CreateRankedChoiceSubmission(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID, channel PreferenceSubmissionChannel, actorAdultID *ids.XID, responses []RankedChoiceResponseInput) (RankedChoiceSubmission, []RankedChoiceResponse, error) {
	if err := validateSubmissionAttribution(tx.actor, channel, actorAdultID); err != nil {
		return RankedChoiceSubmission{}, nil, err
	}
	if err := validateRankedChoiceResponses(responses); err != nil {
		return RankedChoiceSubmission{}, nil, err
	}
	row, err := tx.queries.CreateRankedChoiceSubmission(ctx, db.CreateRankedChoiceSubmissionParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, StudentID: studentID,
		Channel: db.PreferenceSubmissionChannel(channel), ActorType: db.AuditActorType(tx.actor.Type), ActorUserID: tx.actor.UserID,
		ActorAdultID: actorAdultID, ActorLabel: strings.TrimSpace(tx.actor.Label),
	})
	if err != nil {
		return RankedChoiceSubmission{}, nil, fmt.Errorf("create ranked choice submission: %w", err)
	}
	submission, err := rankedChoiceSubmission(row)
	if err != nil {
		return RankedChoiceSubmission{}, nil, err
	}

	createdResponses := make([]RankedChoiceResponse, 0, len(responses))
	for _, response := range responses {
		created, err := tx.queries.CreateRankedChoiceResponse(ctx, db.CreateRankedChoiceResponseParams{
			OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID,
			SubmissionID: submission.ID, OfferingID: response.OfferingID, Response: db.RankedChoiceAnswer(response.Answer), Rank: nullableRank(response.Rank),
		})
		if err != nil {
			return RankedChoiceSubmission{}, nil, fmt.Errorf("create ranked choice response: %w", err)
		}
		value, err := rankedChoiceResponse(created)
		if err != nil {
			return RankedChoiceSubmission{}, nil, err
		}
		createdResponses = append(createdResponses, value)
	}
	return submission, createdResponses, nil
}

func (tx *Tx) ListRankedChoiceSubmissions(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID) ([]RankedChoiceSubmission, error) {
	rows, err := tx.queries.ListRankedChoiceSubmissions(ctx, db.ListRankedChoiceSubmissionsParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, StudentID: studentID,
	})
	if err != nil {
		return nil, fmt.Errorf("list ranked choice submissions: %w", err)
	}
	result := make([]RankedChoiceSubmission, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceSubmission(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) GetLatestRankedChoiceSubmission(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID) (RankedChoiceSubmission, error) {
	row, err := tx.queries.GetLatestRankedChoiceSubmission(ctx, db.GetLatestRankedChoiceSubmissionParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, StudentID: studentID,
	})
	if err != nil {
		return RankedChoiceSubmission{}, fmt.Errorf("get latest ranked choice submission: %w", err)
	}
	return rankedChoiceSubmission(row)
}

func (tx *Tx) ListRankedChoiceResponses(ctx context.Context, schoolYearID, programID, sessionID, submissionID ids.XID) ([]RankedChoiceResponse, error) {
	rows, err := tx.queries.ListRankedChoiceResponses(ctx, db.ListRankedChoiceResponsesParams{
		OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, ProgramID: programID, SessionID: sessionID, SubmissionID: submissionID,
	})
	if err != nil {
		return nil, fmt.Errorf("list ranked choice responses: %w", err)
	}
	result := make([]RankedChoiceResponse, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceResponse(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func validateSubmissionAttribution(actor audit.Actor, channel PreferenceSubmissionChannel, actorAdultID *ids.XID) error {
	switch channel {
	case PreferenceChannelGuardian:
		if actor.Type != audit.ActorTypeLink || actorAdultID == nil {
			return errors.New("preference submission: guardian channel requires a guardian link and adult id")
		}
	case PreferenceChannelStudentCode:
		if actor.Type != audit.ActorTypeLink || actorAdultID != nil {
			return errors.New("preference submission: student-code channel requires a student-code link")
		}
	case PreferenceChannelAdministratorOnBehalf:
		if actor.Type != audit.ActorTypeUser || actor.UserID == nil || actorAdultID != nil {
			return errors.New("preference submission: administrator-on-behalf channel requires an administrator user")
		}
	default:
		return fmt.Errorf("preference submission: invalid channel %q", channel)
	}
	return nil
}

func validateInterestProfileAnswers(answers []InterestProfileAnswer) error {
	seen := make(map[ids.XID]struct{}, len(answers))
	for _, answer := range answers {
		if answer.InterestAreaID == "" {
			return errors.New("interest profile response: interest area id is required")
		}
		if _, ok := seen[answer.InterestAreaID]; ok {
			return fmt.Errorf("interest profile response: interest area %q is repeated", answer.InterestAreaID)
		}
		seen[answer.InterestAreaID] = struct{}{}
		switch answer.Rating {
		case InterestProfileVeryInterested, InterestProfileInterested, InterestProfileNotInterested, InterestProfileUnrated:
		default:
			return fmt.Errorf("interest profile response: rating %q is invalid", answer.Rating)
		}
	}
	return nil
}

func validateRankedChoiceResponses(responses []RankedChoiceResponseInput) error {
	if len(responses) == 0 {
		return errors.New("ranked choice submission: at least one offering response is required")
	}
	seenOfferings := make(map[ids.XID]struct{}, len(responses))
	seenRanks := make(map[int]struct{}, len(responses))
	for _, response := range responses {
		if response.OfferingID == "" {
			return errors.New("ranked choice response: offering id is required")
		}
		if _, ok := seenOfferings[response.OfferingID]; ok {
			return fmt.Errorf("ranked choice response: offering %q is repeated", response.OfferingID)
		}
		seenOfferings[response.OfferingID] = struct{}{}
		switch response.Answer {
		case RankedChoiceRanked:
			if response.Rank == nil || *response.Rank < 1 {
				return errors.New("ranked choice response: ranked answer requires a positive rank")
			}
			if _, ok := seenRanks[*response.Rank]; ok {
				return fmt.Errorf("ranked choice response: rank %d is repeated", *response.Rank)
			}
			seenRanks[*response.Rank] = struct{}{}
		case RankedChoiceInterested, RankedChoiceNotInterested, RankedChoiceNoResponse:
			if response.Rank != nil {
				return fmt.Errorf("ranked choice response: %q cannot include a rank", response.Answer)
			}
		default:
			return fmt.Errorf("ranked choice response: answer %q is invalid", response.Answer)
		}
	}
	return nil
}

func nullableRank(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func interestProfileSubmission(row db.InterestProfileSubmission) (InterestProfileSubmission, error) {
	submittedAt, err := programTime(row.SubmittedAt, "submitted_at")
	if err != nil {
		return InterestProfileSubmission{}, err
	}
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileSubmission{}, err
	}
	return InterestProfileSubmission{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, StudentID: row.StudentID, Channel: PreferenceSubmissionChannel(row.Channel), ActorType: audit.ActorType(row.ActorType), ActorUserID: row.ActorUserID, ActorAdultID: row.ActorAdultID, ActorLabel: row.ActorLabel, SubmittedAt: submittedAt, CreatedAt: createdAt}, nil
}

func interestProfileResponse(row db.InterestProfileResponse) (InterestProfileResponse, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return InterestProfileResponse{}, err
	}
	return InterestProfileResponse{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SubmissionID: row.SubmissionID, InterestAreaID: row.InterestAreaID, Rating: InterestProfileRating(row.Response), CreatedAt: createdAt}, nil
}

func rankedChoiceSubmission(row db.RankedChoiceSubmission) (RankedChoiceSubmission, error) {
	submittedAt, err := programTime(row.SubmittedAt, "submitted_at")
	if err != nil {
		return RankedChoiceSubmission{}, err
	}
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return RankedChoiceSubmission{}, err
	}
	return RankedChoiceSubmission{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, StudentID: row.StudentID, Channel: PreferenceSubmissionChannel(row.Channel), ActorType: audit.ActorType(row.ActorType), ActorUserID: row.ActorUserID, ActorAdultID: row.ActorAdultID, ActorLabel: row.ActorLabel, SubmittedAt: submittedAt, CreatedAt: createdAt}, nil
}

func rankedChoiceResponse(row db.RankedChoiceResponse) (RankedChoiceResponse, error) {
	createdAt, err := programTime(row.CreatedAt, "created_at")
	if err != nil {
		return RankedChoiceResponse{}, err
	}
	var rank *int
	if row.Rank.Valid {
		value := int(row.Rank.Int32)
		rank = &value
	}
	return RankedChoiceResponse{ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID, ProgramID: row.ProgramID, SessionID: row.SessionID, SubmissionID: row.SubmissionID, OfferingID: row.OfferingID, Answer: RankedChoiceAnswer(row.Response), Rank: rank, CreatedAt: createdAt}, nil
}

func (tx *Tx) ListAllInterestProfileSubmissionsForRegistry(ctx context.Context) ([]InterestProfileSubmission, error) {
	rows, err := tx.queries.ListAllInterestProfileSubmissionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list interest profile submissions for registry: %w", err)
	}
	result := make([]InterestProfileSubmission, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileSubmission(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileSubmissionForRegistry(ctx context.Context, id ids.XID) (InterestProfileSubmission, error) {
	row, err := tx.queries.FindInterestProfileSubmissionForRegistry(ctx, db.FindInterestProfileSubmissionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileSubmission{}, nil
	}
	if err != nil {
		return InterestProfileSubmission{}, fmt.Errorf("find interest profile submission for registry: %w", err)
	}
	return interestProfileSubmission(row)
}

func (tx *Tx) ListAllInterestProfileResponsesForRegistry(ctx context.Context) ([]InterestProfileResponse, error) {
	rows, err := tx.queries.ListAllInterestProfileResponsesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list interest profile responses for registry: %w", err)
	}
	result := make([]InterestProfileResponse, 0, len(rows))
	for _, row := range rows {
		value, err := interestProfileResponse(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindInterestProfileResponseForRegistry(ctx context.Context, id ids.XID) (InterestProfileResponse, error) {
	row, err := tx.queries.FindInterestProfileResponseForRegistry(ctx, db.FindInterestProfileResponseForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return InterestProfileResponse{}, nil
	}
	if err != nil {
		return InterestProfileResponse{}, fmt.Errorf("find interest profile response for registry: %w", err)
	}
	return interestProfileResponse(row)
}

func (tx *Tx) ListAllRankedChoiceSubmissionsForRegistry(ctx context.Context) ([]RankedChoiceSubmission, error) {
	rows, err := tx.queries.ListAllRankedChoiceSubmissionsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list ranked choice submissions for registry: %w", err)
	}
	result := make([]RankedChoiceSubmission, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceSubmission(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindRankedChoiceSubmissionForRegistry(ctx context.Context, id ids.XID) (RankedChoiceSubmission, error) {
	row, err := tx.queries.FindRankedChoiceSubmissionForRegistry(ctx, db.FindRankedChoiceSubmissionForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return RankedChoiceSubmission{}, nil
	}
	if err != nil {
		return RankedChoiceSubmission{}, fmt.Errorf("find ranked choice submission for registry: %w", err)
	}
	return rankedChoiceSubmission(row)
}

func (tx *Tx) ListAllRankedChoiceResponsesForRegistry(ctx context.Context) ([]RankedChoiceResponse, error) {
	rows, err := tx.queries.ListAllRankedChoiceResponsesForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list ranked choice responses for registry: %w", err)
	}
	result := make([]RankedChoiceResponse, 0, len(rows))
	for _, row := range rows {
		value, err := rankedChoiceResponse(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

func (tx *Tx) FindRankedChoiceResponseForRegistry(ctx context.Context, id ids.XID) (RankedChoiceResponse, error) {
	row, err := tx.queries.FindRankedChoiceResponseForRegistry(ctx, db.FindRankedChoiceResponseForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if errors.Is(err, pgx.ErrNoRows) {
		return RankedChoiceResponse{}, nil
	}
	if err != nil {
		return RankedChoiceResponse{}, fmt.Errorf("find ranked choice response for registry: %w", err)
	}
	return rankedChoiceResponse(row)
}
