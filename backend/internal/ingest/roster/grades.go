package roster

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
)

// GradeRecord is the narrow canonical record emitted by grades_csv. The name
// remains a single source value because this source has no authority to split
// a person's name into given and family components.
type GradeRecord struct {
	StudentName string
	Grade       string
}

// ParseGradesCSV reads a headered student_name/grade CSV by header name. Extra
// columns are ignored, and rows with a blank grade are retained as a nullable
// source value for the later matching stage.
func ParseGradesCSV(reader io.Reader) ([]GradeRecord, error) {
	if reader == nil {
		return nil, errors.New("roster: nil CSV reader")
	}
	rows, err := csv.NewReader(reader).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("roster: decode grades CSV: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("roster: grades CSV is empty")
	}
	columns := make(map[string]int, len(rows[0]))
	for index, name := range rows[0] {
		columns[strings.ToLower(strings.TrimSpace(name))] = index
	}
	studentColumn, ok := columns["student_name"]
	if !ok {
		studentColumn, ok = columns["name"]
	}
	gradeColumn, gradeOK := columns["grade"]
	if !ok || !gradeOK {
		return nil, errors.New("roster: grades CSV must name student_name and grade columns")
	}
	result := make([]GradeRecord, 0, len(rows)-1)
	for rowIndex, row := range rows[1:] {
		if studentColumn >= len(row) || gradeColumn >= len(row) {
			return nil, fmt.Errorf("roster: grades CSV row %d has too few columns", rowIndex+2)
		}
		name := strings.TrimSpace(row[studentColumn])
		if name == "" {
			return nil, fmt.Errorf("roster: grades CSV row %d has no student name", rowIndex+2)
		}
		result = append(result, GradeRecord{StudentName: name, Grade: strings.TrimSpace(row[gradeColumn])})
	}
	return result, nil
}
