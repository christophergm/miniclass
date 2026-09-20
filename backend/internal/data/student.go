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
	GradeLevelID       *ids.XID
	HomeroomID         ids.XID
	ExternalIdentifier *string
	IsPlaceholder      bool
	Provenance         string
	DeletedAt          *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// CreateStudent inserts a student under the transaction tenant and year.
func (tx *Tx) CreateStudent(ctx context.Context, schoolYearID ids.XID, gradeLevelID *ids.XID, homeroomID ids.XID, legalGivenName, legalFamilyName string, preferredGivenName, externalIdentifier *string) (Student, error) {
	return tx.CreateStudentWithMetadata(ctx, schoolYearID, gradeLevelID, homeroomID, legalGivenName, legalFamilyName, preferredGivenName, externalIdentifier, false, "legacy")
}

// CreateStudentWithMetadata inserts a roster record while preserving the
// provenance and placeholder state that later corrections must not overwrite.
func (tx *Tx) CreateStudentWithMetadata(ctx context.Context, schoolYearID ids.XID, gradeLevelID *ids.XID, homeroomID ids.XID, legalGivenName, legalFamilyName string, preferredGivenName, externalIdentifier *string, isPlaceholder bool, provenance string) (Student, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	provenance = strings.TrimSpace(provenance)
	if legalGivenName == "" || legalFamilyName == "" {
		return Student{}, errors.New("create student: legal names are required")
	}
	if strings.TrimSpace(string(schoolYearID)) == "" || strings.TrimSpace(string(homeroomID)) == "" {
		return Student{}, errors.New("create student: school year and homeroom are required")
	}
	if provenance == "" {
		return Student{}, errors.New("create student: provenance is required")
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
		IsPlaceholder:      isPlaceholder,
		Provenance:         provenance,
	})
	if err != nil {
		return Student{}, wrapStudentMutationError("create student", err)
	}
	return studentFromCreateRow(row)
}

// ListStudents lists students for one year. Deleted rows are opt-in.
func (tx *Tx) ListStudents(ctx context.Context, schoolYearID ids.XID, includeDeleted bool) ([]Student, error) {
	rows, err := tx.queries.ListStudents(ctx, db.ListStudentsParams{OrganizationID: tx.organizationID, SchoolYearID: schoolYearID, Column3: includeDeleted})
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	result := make([]Student, 0, len(rows))
	for _, row := range rows {
		value, err := studentFromListStudentsRow(row)
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
	return studentFromGetStudentByIDRow(row)
}

// GetStudentByIDIncludingDeleted is reserved for restore and other explicit
// historical operations. Ordinary reads use GetStudentByID so deleted rows
// remain invisible by default.
func (tx *Tx) GetStudentByIDIncludingDeleted(ctx context.Context, schoolYearID, id ids.XID) (Student, error) {
	if strings.TrimSpace(string(id)) == "" || strings.TrimSpace(string(schoolYearID)) == "" {
		return Student{}, errors.New("get student: ids are required")
	}
	row, err := tx.queries.GetStudentByIDIncludingDeleted(ctx, db.GetStudentByIDIncludingDeletedParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Student{}, fmt.Errorf("get student including deleted: %w", err)
	}
	return studentFromGetStudentByIDIncludingDeletedRow(row)
}

// UpdateStudent replaces the editable fields of one active student.
func (tx *Tx) UpdateStudent(ctx context.Context, schoolYearID, id ids.XID, legalGivenName, legalFamilyName string, preferredGivenName *string, gradeLevelID *ids.XID, homeroomID ids.XID, externalIdentifier *string) (Student, error) {
	legalGivenName = strings.TrimSpace(legalGivenName)
	legalFamilyName = strings.TrimSpace(legalFamilyName)
	if legalGivenName == "" || legalFamilyName == "" {
		return Student{}, errors.New("update student: legal names are required")
	}
	if strings.TrimSpace(string(homeroomID)) == "" {
		return Student{}, errors.New("update student: homeroom is required")
	}
	row, err := tx.queries.UpdateStudent(ctx, db.UpdateStudentParams{
		ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID,
		LegalGivenName: legalGivenName, LegalFamilyName: legalFamilyName,
		PreferredGivenName: nullableStudentText(preferredGivenName), GradeLevelID: gradeLevelID,
		HomeroomID: homeroomID, ExternalIdentifier: nullableStudentText(externalIdentifier),
	})
	if err != nil {
		return Student{}, wrapStudentMutationError("update student", err)
	}
	return studentFromUpdateRow(row)
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

// DeidentifyStudent preserves grade and homeroom history while removing the
// identifying fields from a retained student row.
func (tx *Tx) DeidentifyStudent(ctx context.Context, schoolYearID, id ids.XID) (Student, error) {
	row, err := tx.queries.DeidentifyStudent(ctx, db.DeidentifyStudentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Student{}, wrapStudentMutationError("de-identify student", err)
	}
	return studentFromDeidentifyRow(row)
}

func (tx *Tx) CountStudentAssociatedData(ctx context.Context, schoolYearID, id ids.XID) (int64, error) {
	return tx.queries.CountStudentAssociatedData(ctx, db.CountStudentAssociatedDataParams{StudentID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
}

func (tx *Tx) HardDeleteStudent(ctx context.Context, schoolYearID, id ids.XID) error {
	params := db.HardDeleteStudentRankedAccessCodesParams{StudentID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID}
	if err := tx.queries.HardDeleteStudentRankedAccessCodes(ctx, params); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentSurveyAccessCodes(ctx, db.HardDeleteStudentSurveyAccessCodesParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentSurveySnapshots(ctx, db.HardDeleteStudentSurveySnapshotsParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentSurveyAudience(ctx, db.HardDeleteStudentSurveyAudienceParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentRankedSubmissions(ctx, db.HardDeleteStudentRankedSubmissionsParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentSurveySubmissions(ctx, db.HardDeleteStudentSurveySubmissionsParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentNonParticipations(ctx, db.HardDeleteStudentNonParticipationsParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentMemberships(ctx, db.HardDeleteStudentMembershipsParams(params)); err != nil {
		return err
	}
	if err := tx.queries.HardDeleteStudentRelationships(ctx, db.HardDeleteStudentRelationshipsParams(params)); err != nil {
		return err
	}
	return tx.queries.HardDeleteStudent(ctx, db.HardDeleteStudentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
}

func (tx *Tx) RestoreStudent(ctx context.Context, schoolYearID, id ids.XID) (Student, error) {
	row, err := tx.queries.RestoreStudent(ctx, db.RestoreStudentParams{ID: id, OrganizationID: tx.organizationID, SchoolYearID: schoolYearID})
	if err != nil {
		return Student{}, wrapStudentMutationError("restore student", err)
	}
	return studentFromRestoreRow(row)
}

// ListAllActiveStudentsForRegistry is used only by the isolation registry.
func (tx *Tx) ListAllActiveStudentsForRegistry(ctx context.Context) ([]Student, error) {
	rows, err := tx.queries.ListAllActiveStudentsForRegistry(ctx, tx.organizationID)
	if err != nil {
		return nil, fmt.Errorf("list students for registry: %w", err)
	}
	result := make([]Student, 0, len(rows))
	for _, row := range rows {
		value, err := studentFromListAllActiveStudentsForRegistryRow(row)
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
	value, err := studentFromFindStudentForRegistryRow(row)
	return value, value.SchoolYearID, err
}

func studentFromParts(id, organizationID, schoolYearID ids.XID, legalGivenName, legalFamilyName string, preferredGivenName pgtype.Text, gradeLevelID *ids.XID, homeroomID ids.XID, externalIdentifier pgtype.Text, isPlaceholder bool, provenance string, deletedAt, createdAtValue, updatedAtValue pgtype.Timestamptz) (Student, error) {
	createdAt, err := studentTime(createdAtValue, "created_at")
	if err != nil {
		return Student{}, err
	}
	updatedAt, err := studentTime(updatedAtValue, "updated_at")
	if err != nil {
		return Student{}, err
	}
	return Student{
		ID: id, OrganizationID: organizationID,
		SchoolYearID:       schoolYearID,
		LegalGivenName:     legalGivenName,
		LegalFamilyName:    legalFamilyName,
		PreferredGivenName: nullableStudentString(preferredGivenName),
		GradeLevelID:       gradeLevelID,
		HomeroomID:         homeroomID,
		ExternalIdentifier: nullableStudentString(externalIdentifier),
		IsPlaceholder:      isPlaceholder,
		Provenance:         provenance,
		DeletedAt:          nullableStudentTime(deletedAt),
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
	}, nil
}

func studentFromCreateRow(row db.CreateStudentRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromListStudentsRow(row db.ListStudentsRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromGetStudentByIDRow(row db.GetStudentByIDRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromGetStudentByIDIncludingDeletedRow(row db.GetStudentByIDIncludingDeletedRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromUpdateRow(row db.UpdateStudentRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromRestoreRow(row db.RestoreStudentRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromDeidentifyRow(row db.DeidentifyStudentRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromListAllActiveStudentsForRegistryRow(row db.ListAllActiveStudentsForRegistryRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
}

func studentFromFindStudentForRegistryRow(row db.FindStudentForRegistryRow) (Student, error) {
	return studentFromParts(row.ID, row.OrganizationID, row.SchoolYearID, row.LegalGivenName, row.LegalFamilyName, row.PreferredGivenName, row.GradeLevelID, row.HomeroomID, row.ExternalIdentifier, row.IsPlaceholder, row.Provenance, row.DeletedAt, row.CreatedAt, row.UpdatedAt)
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
