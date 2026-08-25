package data

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "github.com/chrismott/miniclass/internal/db/gen"
	"github.com/chrismott/miniclass/internal/ids"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// Student is a tenant-safe, school-year-scoped roster record.
type Student struct {
	ID                 ids.XID
	OrganizationID     ids.XID
	SchoolYearID       ids.XID
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	GradeLevelID       ids.XID
	HomeroomID         ids.XID
	ExternalIdentifier *string
	PriorYearStudentID *ids.XID
	DeletedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CreateStudent inserts a student under the transaction tenant and year.
func (tx *Tx) CreateStudent(ctx context.Context, schoolYearID, gradeLevelID, homeroomID ids.XID, legalGivenName, legalFamilyName string, preferredGivenName, externalIdentifier *string, priorYearStudentID *ids.XID) (Student, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	if legalGivenName == "" || legalFamilyName == "" {
		return Student{}, errors.New("create student: legal names are required")
	}
	if strings.TrimSpace(string(schoolYearID)) == "" || strings.TrimSpace(string(gradeLevelID)) == "" || strings.TrimSpace(string(homeroomID)) == "" {
		return Student{}, errors.New("create student: school year, grade, and homeroom are required")
	}
	row, err := tx.queries.CreateStudent(ctx, db.CreateStudentParams{
		OrganizationID:     tx.organizationID,
		SchoolYearID:       schoolYearID,
		LegalGivenName:     legalGivenName,
		LegalFamilyName:    legalFamilyName,
		PreferredGivenName: nullableStudentText(preferredGivenName),
		GradeLevelID:       gradeLevelID,
		HomeroomID:         homeroomID,
		ExternalIdentifier: nullableStudentText(externalIdentifier),
		PriorYearStudentID: priorYearStudentID,
	})
	if err != nil {
		return Student{}, wrapStudentMutationError("create student", err)
	}
	return student(row)
}

// ListStudents lists active students for one year. The soft-delete predicate
// is explicit here and in every other student query by design.
func (tx *Tx) ListStudents(ctx context.Context, schoolYearID ids.XID) ([]Student, error) {
	rows, err := tx.queries.ListStudents(ctx, db.ListStudentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	result := make([]Student, 0, len(rows))
	for _, row := range rows {
		value, err := student(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// GetStudentByID fetches one active student in the tenant and year.
func (tx *Tx) GetStudentByID(ctx context.Context, schoolYearID, id ids.XID) (Student, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Student{}, errors.New("get student: ids are required")
	}
	row, err := tx.queries.GetStudentByID(ctx, db.GetStudentByIDParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Student{}, fmt.Errorf("get student: %w", err)
	}
	return student(row)
}

// UpdateStudent replaces the editable fields of one active student.
func (tx *Tx) UpdateStudent(ctx context.Context, schoolYearID, id ids.XID, legalGivenName, legalFamilyName string, preferredGivenName *string, gradeLevelID, homeroomID ids.XID, externalIdentifier *string, priorYearStudentID *ids.XID) (Student, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	if legalGivenName == "" || legalFamilyName == "" {
		return Student{}, errors.New("update student: legal names are required")
	}
	if strings.TrimSpace(string(gradeLevelID)) == "" || strings.TrimSpace(string(homeroomID)) == "" {
		return Student{}, errors.New("update student: grade and homeroom are required")
	}
	row, err := tx.queries.UpdateStudent(ctx, db.UpdateStudentParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID,
		LegalGivenName: legalGivenName, LegalFamilyName: legalFamilyName,
		PreferredGivenName: nullableStudentText(preferredGivenName), GradeLevelID: gradeLevelID,
		HomeroomID: homeroomID, ExternalIdentifier: nullableStudentText(externalIdentifier),
		PriorYearStudentID: priorYearStudentID,
	})
	if err != nil {
		return Student{}, wrapStudentMutationError("update student", err)
	}
	return student(row)
}

// SoftDeleteStudent hides an active student while preserving audit/history
// and allowing a future import to reuse its external identifier.
func (tx *Tx) SoftDeleteStudent(ctx context.Context, schoolYearID, id ids.XID) (bool, error) {
	rows, err := tx.queries.SoftDeleteStudent(ctx, db.SoftDeleteStudentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return false, wrapStudentMutationError("delete student", err)
	}
	return rows == 1, nil
}

// ListAllActiveStudentsForRegistry is used only by the isolation registry.
func (tx *Tx) ListAllActiveStudentsForRegistry(ctx context.Context) ([]Student, error) {
	rows, err := tx.queries.ListAllActiveStudentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list students for registry: %w", err)
	}
	result := make([]Student, 0, len(rows))
	for _, row := range rows {
		value, err := student(row)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, nil
}

// FindStudentForRegistry returns one active student and its year for a
// generic isolation operation.
func (tx *Tx) FindStudentForRegistry(ctx context.Context, id ids.XID) (Student, ids.XID, error) {
	row, err := tx.queries.FindStudentForRegistry(ctx, db.FindStudentForRegistryParams{ID: id, OrganizationID: tx.organizationID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Student{}, "", nil
		}
		return Student{}, "", fmt.Errorf("find student for registry: %w", err)
	}
	value, err := student(row)
	return value, value.SchoolYearID, err
}

func student(row db.Student) (Student, error) {
	createdAt, err := studentTime(row.CreatedAt, "created_at")
	if err != nil {
		return Student{}, err
	}
	updatedAt, err := studentTime(row.UpdatedAt, "updated_at")
	if err != nil {
		return Student{}, err
	}
	return Student{
		ID: row.ID, OrganizationID: row.OrganizationID, SchoolYearID: row.SchoolYearID,
		LegalGivenName: row.LegalGivenName, LegalFamilyName: row.LegalFamilyName,
		PreferredGivenName: nullableStudentString(row.PreferredGivenName), GradeLevelID: row.GradeLevelID,
		HomeroomID: row.HomeroomID, ExternalIdentifier: nullableStudentString(row.ExternalIdentifier),
		PriorYearStudentID: row.PriorYearStudentID, DeletedAt: nullableStudentTime(row.DeletedAt),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}, nil
}

func nullableStudentText(value *string) pgtype.Text {
	if value == nil || strings.TrimSpace(*value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: strings.TrimSpace(*value), Valid: true}
}

func nullableStudentString(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func nullableStudentTime(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}

func studentTime(value pgtype.Timestamptz, name string) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, fmt.Errorf("student row: %s is null", name)
	}
	return value.Time, nil
}

func wrapStudentMutationError(operation string, err error) error {
	if isClosedYearDatabaseError(err) {
		return fmt.Errorf("%w: %v", ErrSchoolYearClosed, err)
	}
	return fmt.Errorf("%s: %w", operation, err)
}
