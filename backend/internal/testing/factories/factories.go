// Package factories contains small, valid roster builders for database-backed
// tests and the development seed loader. It deliberately creates one entity
// at a time so isolation tests do not depend on the full corpus.
package factories

import (
	"context"
	"errors"
	"time"

	"github.com/chrismott/miniclass/internal/audit"
	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/chrismott/miniclass/internal/people"
	"github.com/chrismott/miniclass/internal/preference"
	"github.com/chrismott/miniclass/internal/program"
	"github.com/chrismott/miniclass/internal/schoolyear"
	"github.com/chrismott/miniclass/internal/vocabulary"
)

// Factory is scoped to one organization. Every write delegates to the normal
// audited application service, preserving the same tenancy checks as the API.
type Factory struct {
	organizationID string
	actor          audit.Actor
	people         *people.Service
	schoolYears    *schoolyear.Service
	vocabulary     *vocabulary.Service
	programs       *program.Service
	preferences    *preference.Service
}

// New returns builders for one organization. A zero actor is replaced with a
// system actor so the helpers are convenient in focused tests.
func New(database *data.DB, organizationID string, actor audit.Actor) *Factory {
	if actor.Type == "" {
		actor = audit.Actor{Type: audit.ActorTypeSystem, Label: "test factory"}
	}
	return &Factory{
		organizationID: organizationID,
		actor:          actor,
		people:         people.New(database),
		schoolYears:    schoolyear.New(database),
		vocabulary:     vocabulary.New(database),
		programs:       program.New(database),
		preferences:    preference.New(database),
	}
}

func (f *Factory) CreateProgram(ctx context.Context, schoolYearID ids.XID, name string) (data.Program, error) {
	if err := f.validate(); err != nil {
		return data.Program{}, err
	}
	return f.programs.Create(ctx, f.organizationID, f.actor, schoolYearID, name)
}

// CreateInterestArea creates one programme-owned interest-area vocabulary entry.
func (f *Factory) CreateInterestArea(ctx context.Context, schoolYearID, programID ids.XID, label string) (data.InterestArea, error) {
	if err := f.validate(); err != nil {
		return data.InterestArea{}, err
	}
	return f.programs.CreateInterestArea(ctx, f.organizationID, f.actor, schoolYearID, programID, label)
}

// CreateSession creates a planning session with its required meeting dates.
func (f *Factory) CreateSession(ctx context.Context, schoolYearID, programID ids.XID, name string, dates []time.Time) (data.Session, error) {
	if err := f.validate(); err != nil {
		return data.Session{}, err
	}
	return f.programs.CreateSession(ctx, f.organizationID, f.actor, schoolYearID, programID, name, dates)
}

// ConfigureRankedChoice enables ranked-choice collection for a planning or
// catalog-published session.
func (f *Factory) ConfigureRankedChoice(ctx context.Context, schoolYearID, programID, sessionID ids.XID, rankDepth int, deadline time.Time) (data.Session, error) {
	if err := f.validate(); err != nil {
		return data.Session{}, err
	}
	return f.programs.UpdateSession(ctx, f.organizationID, f.actor, schoolYearID, programID, sessionID, program.SessionUpdate{
		RankedChoice: &data.RankedChoiceConfiguration{RankDepth: rankDepth, Deadline: &deadline},
	})
}

// TransitionSession applies a lifecycle transition through the normal
// audited service path.
func (f *Factory) TransitionSession(ctx context.Context, schoolYearID, programID, sessionID ids.XID, nextState data.SessionState, confirm bool, reason string, votingDeadline *time.Time) (program.SessionTransitionResult, error) {
	if err := f.validate(); err != nil {
		return program.SessionTransitionResult{}, err
	}
	return f.programs.TransitionSession(ctx, f.organizationID, f.actor, schoolYearID, programID, sessionID, program.SessionTransitionInput{
		NextState: nextState, Confirm: confirm, Reason: reason, VotingDeadline: votingDeadline,
	})
}

// CreateMeetingDate adds one date to an existing session.
func (f *Factory) CreateMeetingDate(ctx context.Context, schoolYearID, programID, sessionID ids.XID, date time.Time) (data.MeetingDate, error) {
	if err := f.validate(); err != nil {
		return data.MeetingDate{}, err
	}
	return f.programs.CreateMeetingDate(ctx, f.organizationID, f.actor, schoolYearID, programID, sessionID, date)
}

// CreateOffering creates a catalog offering in an existing session.
func (f *Factory) CreateOffering(ctx context.Context, schoolYearID, programID, sessionID ids.XID, name, description string, minimumViableEnrollment *int, capacity int, minGradeLevelID, maxGradeLevelID ids.XID, location, meetingPoint, meetingInstructions string, interestAreaID *ids.XID) (data.Offering, error) {
	if err := f.validate(); err != nil {
		return data.Offering{}, err
	}
	return f.programs.CreateOffering(ctx, f.organizationID, f.actor, schoolYearID, programID, sessionID, name, description, minimumViableEnrollment, capacity, minGradeLevelID, maxGradeLevelID, location, meetingPoint, meetingInstructions, interestAreaID)
}

// CreateSessionNonParticipation records a required reason for excluding a
// programme member from one session.
func (f *Factory) CreateSessionNonParticipation(ctx context.Context, schoolYearID, programID, sessionID, studentID ids.XID, reason string) (data.SessionNonParticipation, error) {
	if err := f.validate(); err != nil {
		return data.SessionNonParticipation{}, err
	}
	return f.programs.CreateSessionNonParticipation(ctx, f.organizationID, f.actor, schoolYearID, programID, sessionID, studentID, reason)
}

func (f *Factory) AddProgramMembership(ctx context.Context, schoolYearID, programID, studentID ids.XID) (data.ProgramMembership, error) {
	if err := f.validate(); err != nil {
		return data.ProgramMembership{}, err
	}
	return f.programs.AddMembership(ctx, f.organizationID, f.actor, schoolYearID, programID, studentID)
}

// SubmitInterestProfile records a synthetic preference through the normal
// audited service path.
func (f *Factory) SubmitInterestProfile(ctx context.Context, input preference.InterestProfileSubmissionInput) (data.InterestProfileSubmission, error) {
	if err := f.validate(); err != nil {
		return data.InterestProfileSubmission{}, err
	}
	return f.preferences.SubmitInterestProfile(ctx, f.organizationID, f.actor, input)
}

// SubmitRankedChoices records a synthetic catalog response through the normal
// audited service path.
func (f *Factory) SubmitRankedChoices(ctx context.Context, input preference.RankedChoiceSubmissionInput) (data.RankedChoiceSubmission, error) {
	if err := f.validate(); err != nil {
		return data.RankedChoiceSubmission{}, err
	}
	return f.preferences.SubmitRankedChoices(ctx, f.organizationID, f.actor, input)
}

func (f *Factory) validate() error {
	if f == nil || f.people == nil || f.schoolYears == nil || f.vocabulary == nil || f.programs == nil || f.preferences == nil {
		return errors.New("factory is nil")
	}
	if f.organizationID == "" {
		return errors.New("factory organization id is empty")
	}
	return nil
}

// CreateSchoolYear creates a minimal setup school year.
func (f *Factory) CreateSchoolYear(ctx context.Context, label string) (data.SchoolYear, error) {
	if err := f.validate(); err != nil {
		return data.SchoolYear{}, err
	}
	return f.schoolYears.Create(ctx, f.organizationID, f.actor, label)
}

// CreateGradeLevel creates one school-year-scoped grade vocabulary entry.
func (f *Factory) CreateGradeLevel(ctx context.Context, schoolYearID ids.XID, code, label string) (data.GradeLevel, error) {
	if err := f.validate(); err != nil {
		return data.GradeLevel{}, err
	}
	return f.vocabulary.CreateGrade(ctx, f.organizationID, schoolYearID, f.actor, code, label)
}

// CreateHomeroom creates one school-year-scoped homeroom vocabulary entry.
func (f *Factory) CreateHomeroom(ctx context.Context, schoolYearID ids.XID, name string) (data.Homeroom, error) {
	if err := f.validate(); err != nil {
		return data.Homeroom{}, err
	}
	return f.vocabulary.CreateHomeroom(ctx, f.organizationID, schoolYearID, f.actor, name, nil)
}

// CreateStudent creates one minimal-valid student in a school year.
func (f *Factory) CreateStudent(ctx context.Context, schoolYearID ids.XID, input people.StudentCreateInput) (data.Student, error) {
	if err := f.validate(); err != nil {
		return data.Student{}, err
	}
	return f.people.CreateStudent(ctx, f.organizationID, schoolYearID, f.actor, input)
}

// CreateUngradedStudent creates the setup-state case where a student's grade
// has not yet been supplied. Homeroom remains required for dismissal lists.
func (f *Factory) CreateUngradedStudent(ctx context.Context, schoolYearID, homeroomID ids.XID, givenName, familyName string) (data.Student, error) {
	return f.CreateStudent(ctx, schoolYearID, people.StudentCreateInput{LegalGivenName: givenName, LegalFamilyName: familyName, HomeroomID: homeroomID})
}

// CreateAdult creates one minimal-valid adult in a school year.
func (f *Factory) CreateAdult(ctx context.Context, schoolYearID ids.XID, input people.AdultCreateInput) (data.Adult, error) {
	if err := f.validate(); err != nil {
		return data.Adult{}, err
	}
	return f.people.Create(ctx, f.organizationID, schoolYearID, f.actor, input)
}

// CreateUndeclaredAdult creates an adult whose participation survey answer is
// still absent rather than fabricating an unavailable declaration.
func (f *Factory) CreateUndeclaredAdult(ctx context.Context, schoolYearID ids.XID, givenName, familyName string) (data.Adult, error) {
	return f.CreateAdult(ctx, schoolYearID, people.AdultCreateInput{LegalGivenName: givenName, LegalFamilyName: familyName})
}

// CreateGuardianRelationship creates one adult-to-student relationship.
func (f *Factory) CreateGuardianRelationship(ctx context.Context, schoolYearID ids.XID, input people.GuardianRelationshipCreateInput) (data.GuardianRelationship, error) {
	if err := f.validate(); err != nil {
		return data.GuardianRelationship{}, err
	}
	return f.people.CreateGuardianRelationship(ctx, f.organizationID, schoolYearID, f.actor, input)
}
