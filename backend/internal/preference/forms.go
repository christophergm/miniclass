package preference

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
)

// FormType identifies the two independent preference instruments. Keeping the
// type on the form prevents a respondent client from treating a session vote
// as an interest-profile answer (or vice versa).
type FormType string

const (
	FormTypeInterestProfile FormType = "interest_profile"
	FormTypeRankedChoice    FormType = "ranked_choice"
)

var (
	ErrPreferenceFormNotAvailable  = errors.New("preference form is not available")
	ErrPreferenceStudentOutOfScope = errors.New("student is outside the respondent scope")
)

// PreferenceForm is deliberately smaller than the administrator survey view.
// It contains only what a respondent needs to complete one instrument and the
// existing effective response. Codes, audience snapshots, and roster
// metadata never cross this boundary.
type PreferenceForm struct {
	Type            FormType
	ID              ids.XID
	SchoolYearID    ids.XID
	ProgramID       ids.XID
	SessionID       ids.XID
	ProgramName     string
	SessionName     string
	Name            string
	Introduction    string
	StudentID       ids.XID
	StudentName     string
	ClosesAt        *time.Time
	RankDepth       int
	Questions       []PreferenceFormQuestion
	ScaleOptions    []PreferenceFormScaleOption
	Offerings       []PreferenceFormOffering
	InterestAnswers []PreferenceFormInterestAnswer
	RankedAnswers   []PreferenceFormRankedAnswer
	SubmittedAt     *time.Time
}

type PreferenceFormQuestion struct {
	InterestAreaID ids.XID
	Label          string
	Ordinal        int
}

type PreferenceFormScaleOption struct {
	Value   string
	Label   string
	Ordinal int
}

type PreferenceFormOffering struct {
	ID                  ids.XID
	Name                string
	Description         string
	MinGradeLevelID     ids.XID
	MaxGradeLevelID     ids.XID
	Location            string
	MeetingPoint        string
	MeetingInstructions string
	MeetingDates        []time.Time
}

type PreferenceFormInterestAnswer struct {
	InterestAreaID ids.XID
	Rating         *data.InterestProfileRating
}

type PreferenceFormRankedAnswer struct {
	OfferingID ids.XID
	Answer     data.RankedChoiceAnswer
	Rank       *int
}

type GuardianPreferenceStudent struct {
	StudentID   ids.XID
	DisplayName string
	Forms       []PreferenceForm
}

type GuardianPreferenceForms struct {
	SchoolYearID ids.XID
	Students     []GuardianPreferenceStudent
}

// GetInterestProfileForm returns one currently open survey for a selected
// student. The caller is responsible for authenticating the principal; this
// method enforces the survey audience and the open window in the transaction.
func (s *Service) GetInterestProfileForm(ctx context.Context, organizationID string, schoolYearID, programID, surveyID, studentID ids.XID) (PreferenceForm, error) {
	if s == nil || s.database == nil {
		return PreferenceForm{}, ErrPreferenceServiceNil
	}
	var result PreferenceForm
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			return err
		}
		program, err := tx.GetProgram(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result, err = interestProfileForm(ctx, tx, survey, program.Name, student)
		return err
	})
	if err != nil {
		return PreferenceForm{}, fmt.Errorf("get interest profile form: %w", err)
	}
	return result, nil
}

// GetInterestProfileFormByCode resolves the one instrument-bound code before
// loading the form. The code is never returned in the form or any response.
func (s *Service) GetInterestProfileFormByCode(ctx context.Context, organizationID string, schoolYearID, programID, surveyID ids.XID, code string) (PreferenceForm, error) {
	if s == nil || s.database == nil {
		return PreferenceForm{}, ErrPreferenceServiceNil
	}
	var result PreferenceForm
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		survey, err := tx.GetInterestProfileSurvey(ctx, schoolYearID, programID, surveyID)
		if err != nil {
			return err
		}
		studentID, err := tx.FindActiveInterestProfileSurveyAccessCode(ctx, schoolYearID, programID, surveyID, surveyCodeHash(code))
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return ErrSurveyCodeInvalid
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return ErrSurveyCodeInvalid
		}
		program, err := tx.GetProgram(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result, err = interestProfileForm(ctx, tx, survey, program.Name, student)
		return err
	})
	if err != nil {
		return PreferenceForm{}, fmt.Errorf("get interest profile form by code: %w", err)
	}
	return result, nil
}

// GetRankedChoiceForm returns one currently open session catalog for a
// selected participating student.
func (s *Service) GetRankedChoiceForm(ctx context.Context, organizationID string, schoolYearID, programID, sessionID, studentID ids.XID) (PreferenceForm, error) {
	if s == nil || s.database == nil {
		return PreferenceForm{}, ErrPreferenceServiceNil
	}
	var result PreferenceForm
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			return err
		}
		program, err := tx.GetProgram(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result, err = rankedChoiceForm(ctx, tx, session, program.Name, student)
		return err
	})
	if err != nil {
		return PreferenceForm{}, fmt.Errorf("get ranked-choice form: %w", err)
	}
	return result, nil
}

// GetRankedChoiceFormByCode resolves the one session-bound code before
// returning the course guide and the student's latest valid response.
func (s *Service) GetRankedChoiceFormByCode(ctx context.Context, organizationID string, schoolYearID, programID, sessionID ids.XID, code string) (PreferenceForm, error) {
	if s == nil || s.database == nil {
		return PreferenceForm{}, ErrPreferenceServiceNil
	}
	var result PreferenceForm
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		session, err := tx.GetSession(ctx, schoolYearID, programID, sessionID)
		if err != nil {
			return err
		}
		studentID, err := tx.FindActiveRankedChoiceAccessCode(ctx, schoolYearID, programID, sessionID, rankedChoiceCodeHash(code))
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return ErrRankedChoiceCodeInvalid
		}
		student, err := tx.GetStudentByID(ctx, schoolYearID, studentID)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			return ErrRankedChoiceCodeInvalid
		}
		program, err := tx.GetProgram(ctx, schoolYearID, programID)
		if err != nil {
			return err
		}
		result, err = rankedChoiceForm(ctx, tx, session, program.Name, student)
		return err
	})
	if err != nil {
		return PreferenceForm{}, fmt.Errorf("get ranked-choice form by code: %w", err)
	}
	return result, nil
}

// ListGuardianPreferenceForms returns all open instruments for the adult's
// live relationship scope. The scope is resolved inside the same tenant read
// transaction, so a stale browser payload cannot broaden the result.
func (s *Service) ListGuardianPreferenceForms(ctx context.Context, organizationID string, schoolYearID, adultID ids.XID) (GuardianPreferenceForms, error) {
	if s == nil || s.database == nil {
		return GuardianPreferenceForms{}, ErrPreferenceServiceNil
	}
	var result GuardianPreferenceForms
	err := s.database.InTenantRead(ctx, organizationID, func(ctx context.Context, tx *data.Tx) error {
		scope, err := tx.ResolveGuardianScope(ctx, schoolYearID, adultID)
		if err != nil {
			return err
		}
		students, err := tx.ListStudents(ctx, schoolYearID, false)
		if err != nil {
			return err
		}
		studentByID := make(map[ids.XID]data.Student, len(students))
		for _, student := range students {
			studentByID[student.ID] = student
		}
		programs, err := tx.ListPrograms(ctx, schoolYearID)
		if err != nil {
			return err
		}
		result = GuardianPreferenceForms{SchoolYearID: schoolYearID, Students: make([]GuardianPreferenceStudent, 0, len(scope.StudentIDs))}
		for _, studentID := range scope.StudentIDs {
			student, ok := studentByID[studentID]
			if !ok {
				continue
			}
			entry := GuardianPreferenceStudent{StudentID: student.ID, DisplayName: studentDisplayName(student), Forms: []PreferenceForm{}}
			for _, program := range programs {
				surveys, err := tx.ListInterestProfileSurveys(ctx, schoolYearID, program.ID)
				if err != nil {
					return err
				}
				for _, survey := range surveys {
					form, err := interestProfileForm(ctx, tx, survey, program.Name, student)
					if err != nil {
						if errors.Is(err, ErrPreferenceFormNotAvailable) {
							continue
						}
						return err
					} else {
						entry.Forms = append(entry.Forms, form)
					}
				}
				sessions, err := tx.ListSessions(ctx, schoolYearID, program.ID)
				if err != nil {
					return err
				}
				for _, session := range sessions {
					form, err := rankedChoiceForm(ctx, tx, session, program.Name, student)
					if err != nil {
						if errors.Is(err, ErrPreferenceFormNotAvailable) {
							continue
						}
						return err
					}
					entry.Forms = append(entry.Forms, form)
				}
			}
			result.Students = append(result.Students, entry)
		}
		sort.Slice(result.Students, func(i, j int) bool { return result.Students[i].DisplayName < result.Students[j].DisplayName })
		return nil
	})
	if err != nil {
		return GuardianPreferenceForms{}, fmt.Errorf("list guardian preference forms: %w", err)
	}
	return result, nil
}

func interestProfileForm(ctx context.Context, tx *data.Tx, survey data.InterestProfileSurvey, programName string, student data.Student) (PreferenceForm, error) {
	now := time.Now().UTC()
	if effectiveSurveyState(survey, now) != data.InterestProfileSurveyOpen || survey.OpensAt == nil || now.Before(*survey.OpensAt) || survey.ClosesAt == nil || !now.Before(*survey.ClosesAt) {
		return PreferenceForm{}, ErrPreferenceFormNotAvailable
	}
	snapshot, err := tx.ListInterestProfileSurveyAudienceSnapshot(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	if !containsStudent(snapshotIDs(snapshot), student.ID) {
		return PreferenceForm{}, ErrPreferenceFormNotAvailable
	}
	questions, err := tx.ListInterestProfileSurveyQuestions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	options, err := tx.ListInterestProfileSurveyScaleOptions(ctx, survey.SchoolYearID, survey.ProgramID, survey.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	effective, err := tx.EffectiveInterestProfile(ctx, survey.SchoolYearID, survey.ProgramID, student.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	effectiveByArea := make(map[ids.XID]data.EffectiveInterestProfileValue, len(effective))
	for _, value := range effective {
		effectiveByArea[value.InterestAreaID] = value
	}
	form := PreferenceForm{Type: FormTypeInterestProfile, ID: survey.ID, SchoolYearID: survey.SchoolYearID, ProgramID: survey.ProgramID, ProgramName: programName, Name: survey.Name, Introduction: survey.Introduction, StudentID: student.ID, StudentName: studentDisplayName(student), ClosesAt: survey.ClosesAt, Questions: make([]PreferenceFormQuestion, 0, len(questions)), ScaleOptions: make([]PreferenceFormScaleOption, 0, len(options)), InterestAnswers: make([]PreferenceFormInterestAnswer, 0, len(questions))}
	for _, question := range questions {
		form.Questions = append(form.Questions, PreferenceFormQuestion{InterestAreaID: question.InterestAreaID, Label: question.Label, Ordinal: question.Ordinal})
		answer := PreferenceFormInterestAnswer{InterestAreaID: question.InterestAreaID}
		if value, ok := effectiveByArea[question.InterestAreaID]; ok {
			rating := value.Rating
			answer.Rating = &rating
			if form.SubmittedAt == nil || value.SubmittedAt.After(*form.SubmittedAt) {
				submittedAt := value.SubmittedAt
				form.SubmittedAt = &submittedAt
			}
		}
		form.InterestAnswers = append(form.InterestAnswers, answer)
	}
	for _, option := range options {
		form.ScaleOptions = append(form.ScaleOptions, PreferenceFormScaleOption{Value: string(option.Value), Label: option.Label, Ordinal: option.Ordinal})
	}
	return form, nil
}

func rankedChoiceForm(ctx context.Context, tx *data.Tx, session data.Session, programName string, student data.Student) (PreferenceForm, error) {
	now := time.Now().UTC()
	if session.RankedChoice == nil || session.State != data.SessionVotingOpen || session.RankedChoice.Deadline == nil || !now.Before(*session.RankedChoice.Deadline) {
		return PreferenceForm{}, ErrPreferenceFormNotAvailable
	}
	if err := ensureRankedChoiceParticipant(ctx, tx, session.SchoolYearID, session.ProgramID, session.ID, student.ID); err != nil {
		if errors.Is(err, ErrRankedChoiceStudentExcluded) {
			return PreferenceForm{}, ErrPreferenceFormNotAvailable
		}
		return PreferenceForm{}, err
	}
	offerings, err := tx.ListOfferings(ctx, session.SchoolYearID, session.ProgramID, session.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	if len(offerings) == 0 {
		return PreferenceForm{}, ErrPreferenceFormNotAvailable
	}
	dates, err := tx.ListMeetingDates(ctx, session.SchoolYearID, session.ProgramID, session.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	form := PreferenceForm{Type: FormTypeRankedChoice, ID: session.ID, SchoolYearID: session.SchoolYearID, ProgramID: session.ProgramID, SessionID: session.ID, ProgramName: programName, SessionName: session.Name, Name: session.Name, StudentID: student.ID, StudentName: studentDisplayName(student), ClosesAt: session.RankedChoice.Deadline, RankDepth: session.RankedChoice.RankDepth, Offerings: make([]PreferenceFormOffering, 0, len(offerings)), RankedAnswers: []PreferenceFormRankedAnswer{}}
	meetingDates := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		meetingDates = append(meetingDates, date.Date)
	}
	for _, offering := range offerings {
		form.Offerings = append(form.Offerings, PreferenceFormOffering{ID: offering.ID, Name: offering.Name, Description: offering.Description, MinGradeLevelID: offering.MinGradeLevelID, MaxGradeLevelID: offering.MaxGradeLevelID, Location: offering.Location, MeetingPoint: offering.MeetingPoint, MeetingInstructions: offering.MeetingInstructions, MeetingDates: meetingDates})
	}
	submission, responses, err := latestRankedChoiceIfPresent(ctx, tx, session, student.ID)
	if err != nil {
		return PreferenceForm{}, err
	}
	if submission.ID != "" {
		form.SubmittedAt = &submission.SubmittedAt
	}
	for _, response := range responses {
		form.RankedAnswers = append(form.RankedAnswers, PreferenceFormRankedAnswer{OfferingID: response.OfferingID, Answer: response.Answer, Rank: response.Rank})
	}
	return form, nil
}

func latestRankedChoiceIfPresent(ctx context.Context, tx *data.Tx, session data.Session, studentID ids.XID) (data.RankedChoiceSubmission, []data.RankedChoiceResponse, error) {
	submission, err := tx.GetLatestRankedChoiceSubmission(ctx, session.SchoolYearID, session.ProgramID, session.ID, studentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return data.RankedChoiceSubmission{}, nil, nil
		}
		return data.RankedChoiceSubmission{}, nil, err
	}
	responses, err := tx.ListRankedChoiceResponses(ctx, session.SchoolYearID, session.ProgramID, session.ID, submission.ID)
	return submission, responses, err
}

func snapshotIDs(values []data.InterestProfileSurveyAudienceSnapshot) []ids.XID {
	result := make([]ids.XID, 0, len(values))
	for _, value := range values {
		result = append(result, value.StudentID)
	}
	return result
}

func containsStudent(values []ids.XID, wanted ids.XID) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func ensureSurveyStudent(ctx context.Context, tx *data.Tx, schoolYearID, programID, surveyID, studentID ids.XID) error {
	snapshot, err := tx.ListInterestProfileSurveyAudienceSnapshot(ctx, schoolYearID, programID, surveyID)
	if err != nil {
		return err
	}
	if !containsStudent(snapshotIDs(snapshot), studentID) {
		return ErrSurveyStudentExcluded
	}
	return nil
}

func ensureGuardianStudent(ctx context.Context, tx *data.Tx, schoolYearID, adultID, studentID ids.XID) error {
	scope, err := tx.ResolveGuardianScope(ctx, schoolYearID, adultID)
	if err != nil {
		return err
	}
	if !containsStudent(scope.StudentIDs, studentID) {
		return ErrPreferenceStudentOutOfScope
	}
	return nil
}

func studentDisplayName(student data.Student) string {
	givenName := student.LegalGivenName
	if student.PreferredGivenName != nil && *student.PreferredGivenName != "" {
		givenName = *student.PreferredGivenName
	}
	return givenName + " " + student.LegalFamilyName
}
