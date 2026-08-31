// Package roster parses the wide roster export used by the community platform.
// It deliberately has no database or HTTP dependencies: the result is the
// canonical input to the later preview/import stages.
package roster

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	AdultExclusionNoName       = "no_name"
	AdultExclusionNoChildren   = "no_children"
	AdultExclusionNoneEnrolled = "none_enrolled"
)

// Field aliases are the explicit mapping SPEC §11.3 requires in place of
// positional reads. The first name in each list is the one the community
// platform actually exports — its documents are Mongo-shaped (`_id`,
// `firstName`, `_class`) — and the rest keep hand-written and
// CSV-derived JSON sources working.
var (
	identifierFields = []string{"_id", "id", "external_id", "external_identifier"}
	givenNameFields  = []string{"firstName", "given_name", "first_name"}
	familyNameFields = []string{"lastName", "family_name", "last_name"}
	classFields      = []string{"_class", "class", "type", "kind", "reference"}
	labelFields      = []string{"name", "label"}
)

// Student is the canonical student record emitted by the roster source. The
// classroom label is display metadata only; it is never interpreted as a grade
// (ADR 0014). The source's own band field — exported as `grade` — is read by
// nothing and deliberately not carried here.
type Student struct {
	SourceExternalIdentifier    string `json:"source_external_identifier"`
	ExternalIdentifier          string `json:"external_identifier"`
	GivenName                   string `json:"given_name"`
	FamilyName                  string `json:"family_name"`
	LegalGivenName              string `json:"legal_given_name"`
	LegalFamilyName             string `json:"legal_family_name"`
	ClassroomExternalIdentifier string `json:"classroom_external_identifier"`
	ClassroomLabel              string `json:"classroom_label"`
}

// Adult is the canonical adult record. Status is retained as source metadata,
// but it does not decide whether the adult has a role in the programme.
type Adult struct {
	SourceExternalIdentifier string `json:"source_external_identifier"`
	ExternalIdentifier       string `json:"external_identifier"`
	Email                    string `json:"email"`
	GivenName                string `json:"given_name"`
	FamilyName               string `json:"family_name"`
	LegalGivenName           string `json:"legal_given_name"`
	LegalFamilyName          string `json:"legal_family_name"`
	Status                   string `json:"status"`
}

// GuardianRelationship is the canonical edge between source records. Person
// names are intentionally absent: opaque source identifiers are the join keys.
type GuardianRelationship struct {
	AdultExternalIdentifier   string `json:"adult_external_identifier"`
	StudentExternalIdentifier string `json:"student_external_identifier"`
	RelationshipType          string `json:"relationship_type"`
}

type ChildExclusion struct {
	SourceExternalIdentifier string `json:"source_external_identifier"`
	GivenName                string `json:"given_name"`
	FamilyName               string `json:"family_name"`
	Reason                   string `json:"reason"`
}

type AdultExclusion struct {
	SourceExternalIdentifier string `json:"source_external_identifier"`
	GivenName                string `json:"given_name"`
	FamilyName               string `json:"family_name"`
	Reason                   string `json:"reason"`
}

// Warning is a non-blocking source issue. Unknown relationship labels are
// warnings because §5.2 requires the organiser to be told without blocking
// otherwise usable roster rows.
type Warning struct {
	Code                      string `json:"code"`
	Message                   string `json:"message"`
	AdultExternalIdentifier   string `json:"adult_external_identifier,omitempty"`
	StudentExternalIdentifier string `json:"student_external_identifier,omitempty"`
	SourceValue               string `json:"source_value,omitempty"`
}

// Result contains canonical records and every non-fatal filter observation.
// The slices preserve first-seen source order; duplicate child records and
// duplicate edges are emitted once.
type Result struct {
	Students              []Student              `json:"students"`
	Adults                []Adult                `json:"adults"`
	GuardianRelationships []GuardianRelationship `json:"guardian_relationships"`
	ExcludedChildren      []ChildExclusion       `json:"excluded_children"`
	ExcludedAdults        []AdultExclusion       `json:"excluded_adults"`
	Warnings              []Warning              `json:"warnings"`

	sourceRows []SourceRow
}

// SourceRow preserves the wide source shape for preview. A row can contain
// an adult, several students, and several guardian edges, so the envelope can
// report outcomes as a tree instead of flattening the import into rows.
type SourceRow struct {
	Number                int                    `json:"number"`
	Adult                 Adult                  `json:"adult"`
	Students              []Student              `json:"students"`
	GuardianRelationships []GuardianRelationship `json:"guardian_relationships"`
}

// Document is the parsed roster plus the source-row association needed by the
// preview phase. Result remains available for callers that only need the
// canonical records.
type Document struct {
	Result Result
	Rows   []SourceRow
}

// ParseJSON parses one wide roster export. It returns an error for malformed
// or empty documents and for contradictory names for one child identifier.
func ParseJSON(document []byte) (Result, error) {
	parsed, err := ParseDocument(document)
	if err != nil {
		return Result{}, err
	}
	return parsed.Result, nil
}

// ParseDocument parses one wide roster export and retains the association
// between each canonical record and its source row.
func ParseDocument(document []byte) (Document, error) {
	if len(bytes.TrimSpace(document)) == 0 {
		return Document{}, errors.New("roster: empty JSON document")
	}
	var records []json.RawMessage
	if err := json.Unmarshal(document, &records); err != nil {
		return Document{}, fmt.Errorf("roster: decode JSON: %w", err)
	}
	if len(records) == 0 {
		return Document{}, errors.New("roster: empty JSON array")
	}

	result, err := parseRecords(records)
	if err != nil {
		return Document{}, err
	}
	return Document{Result: result, Rows: result.sourceRows}, nil
}

// Parse is an alias for ParseJSON for callers that select a source parser by
// package rather than by format-specific function name.
func Parse(document []byte) (Result, error) { return ParseJSON(document) }

// ParseReader parses a complete JSON document from a reader.
func ParseReader(reader io.Reader) (Result, error) {
	if reader == nil {
		return Result{}, errors.New("roster: nil reader")
	}
	document, err := io.ReadAll(reader)
	if err != nil {
		return Result{}, fmt.Errorf("roster: read JSON: %w", err)
	}
	return ParseJSON(document)
}

type childRecord struct {
	Student
	seenClassroom bool
}

type adultRecord struct {
	Adult
	relationships []GuardianRelationship
	childCount    int
	enrolledCount int
}

type sourceRowReference struct {
	Number                int
	AdultID               string
	ChildIDs              []string
	GuardianRelationships []GuardianRelationship
}

func parseRecords(records []json.RawMessage) (Result, error) {
	result := Result{
		Students:              make([]Student, 0),
		Adults:                make([]Adult, 0),
		GuardianRelationships: make([]GuardianRelationship, 0),
		ExcludedChildren:      make([]ChildExclusion, 0),
		ExcludedAdults:        make([]AdultExclusion, 0),
		Warnings:              make([]Warning, 0),
	}
	children := make(map[string]*childRecord)
	childOrder := make([]string, 0)
	adults := make(map[string]*adultRecord)
	adultOrder := make([]string, 0)
	seenEdges := make(map[string]bool)
	rowReferences := make([]sourceRowReference, 0, len(records))

	for recordIndex, raw := range records {
		object, err := objectValue(raw)
		if err != nil {
			return Result{}, fmt.Errorf("roster: adult record %d: %w", recordIndex+1, err)
		}
		adultID := stringField(object, identifierFields...)
		if adultID == "" {
			return Result{}, fmt.Errorf("roster: adult record %d has no opaque id", recordIndex+1)
		}
		adult := &adultRecord{Adult: Adult{
			SourceExternalIdentifier: adultID,
			ExternalIdentifier:       adultID,
			Email:                    stringField(object, "email"),
			GivenName:                stringField(object, givenNameFields...),
			FamilyName:               stringField(object, familyNameFields...),
			Status:                   stringField(object, "status"),
		}}
		adult.LegalGivenName, adult.LegalFamilyName = adult.GivenName, adult.FamilyName
		if existing, ok := adults[adultID]; ok {
			if normalName(existing.GivenName) != normalName(adult.GivenName) || normalName(existing.FamilyName) != normalName(adult.FamilyName) {
				return Result{}, fmt.Errorf("roster: adult %q has contradictory names: %q %q and %q %q", adultID, existing.GivenName, existing.FamilyName, adult.GivenName, adult.FamilyName)
			}
			adult = existing
		} else {
			adults[adultID] = adult
			adultOrder = append(adultOrder, adultID)
		}

		row := sourceRowReference{Number: recordIndex + 1, AdultID: adultID}
		relationships := rawList(object, "relationships")
		for relationshipIndex, rawRelationship := range relationships {
			relationshipObject, err := objectValue(rawRelationship)
			if err != nil {
				return Result{}, fmt.Errorf("roster: adult %q relationship %d: %w", adultID, relationshipIndex+1, err)
			}
			childObject, err := objectField(relationshipObject, "child", "student")
			if err != nil {
				return Result{}, fmt.Errorf("roster: adult %q relationship %d: %w", adultID, relationshipIndex+1, err)
			}
			childID := stringField(childObject, identifierFields...)
			if childID == "" {
				return Result{}, fmt.Errorf("roster: adult %q relationship %d child has no opaque id", adultID, relationshipIndex+1)
			}
			givenName := stringField(childObject, givenNameFields...)
			familyName := stringField(childObject, familyNameFields...)
			classroom, hasClassroom := classroomFromGroups(rawList(childObject, "groups"))
			if existing, ok := children[childID]; ok {
				if normalName(existing.GivenName) != normalName(givenName) || normalName(existing.FamilyName) != normalName(familyName) {
					return Result{}, fmt.Errorf("roster: child %q has contradictory names: %q %q and %q %q", childID, existing.GivenName, existing.FamilyName, givenName, familyName)
				}
				if !existing.seenClassroom && hasClassroom {
					existing.ClassroomExternalIdentifier = classroom.ID
					existing.ClassroomLabel = classroom.Label
					existing.seenClassroom = true
				}
			} else {
				child := &childRecord{Student: Student{
					SourceExternalIdentifier: childID,
					ExternalIdentifier:       childID,
					GivenName:                givenName,
					FamilyName:               familyName,
				}}
				child.LegalGivenName, child.LegalFamilyName = givenName, familyName
				if hasClassroom {
					child.ClassroomExternalIdentifier = classroom.ID
					child.ClassroomLabel = classroom.Label
					child.seenClassroom = true
				}
				children[childID] = child
				childOrder = append(childOrder, childID)
			}

			child := children[childID]
			row.ChildIDs = append(row.ChildIDs, childID)
			adult.childCount++
			if child.seenClassroom {
				adult.enrolledCount++
			}
			typeValue := stringField(relationshipObject, "relationship", "relationship_type", "type", "label")
			mapped, known := relationshipType(typeValue)
			if !known {
				result.Warnings = append(result.Warnings, Warning{
					Code: "unrecognized_relationship", Message: fmt.Sprintf("relationship %q mapped to other", typeValue),
					AdultExternalIdentifier: adultID, StudentExternalIdentifier: childID, SourceValue: typeValue,
				})
			}
			edge := GuardianRelationship{AdultExternalIdentifier: adultID, StudentExternalIdentifier: childID, RelationshipType: mapped}
			row.GuardianRelationships = append(row.GuardianRelationships, edge)
			edgeKey := adultID + "\x00" + childID
			if !seenEdges[edgeKey] {
				seenEdges[edgeKey] = true
				if child.seenClassroom {
					adult.relationships = append(adult.relationships, edge)
				}
			}
		}
		rowReferences = append(rowReferences, row)
	}

	for _, childID := range childOrder {
		child := children[childID]
		if !child.seenClassroom {
			result.ExcludedChildren = append(result.ExcludedChildren, ChildExclusion{
				SourceExternalIdentifier: child.SourceExternalIdentifier, GivenName: child.GivenName, FamilyName: child.FamilyName,
				Reason: "no_classroom",
			})
			continue
		}
		result.Students = append(result.Students, child.Student)
	}
	for _, adultID := range adultOrder {
		adult := adults[adultID]
		switch {
		case normalName(adult.GivenName) == "" || normalName(adult.FamilyName) == "":
			result.ExcludedAdults = append(result.ExcludedAdults, AdultExclusion{
				SourceExternalIdentifier: adult.SourceExternalIdentifier, GivenName: adult.GivenName, FamilyName: adult.FamilyName, Reason: AdultExclusionNoName,
			})
		case adult.childCount == 0:
			result.ExcludedAdults = append(result.ExcludedAdults, AdultExclusion{
				SourceExternalIdentifier: adult.SourceExternalIdentifier, GivenName: adult.GivenName, FamilyName: adult.FamilyName, Reason: AdultExclusionNoChildren,
			})
		case adult.enrolledCount == 0:
			result.ExcludedAdults = append(result.ExcludedAdults, AdultExclusion{
				SourceExternalIdentifier: adult.SourceExternalIdentifier, GivenName: adult.GivenName, FamilyName: adult.FamilyName, Reason: AdultExclusionNoneEnrolled,
			})
		default:
			result.Adults = append(result.Adults, adult.Adult)
			result.GuardianRelationships = append(result.GuardianRelationships, adult.relationships...)
		}
	}
	for _, reference := range rowReferences {
		row := SourceRow{Number: reference.Number, Adult: adults[reference.AdultID].Adult}
		seenChildren := make(map[string]bool)
		for _, childID := range reference.ChildIDs {
			if seenChildren[childID] {
				continue
			}
			seenChildren[childID] = true
			if child := children[childID]; child != nil && child.seenClassroom {
				row.Students = append(row.Students, child.Student)
			}
		}
		seenRowEdges := make(map[string]bool)
		for _, edge := range reference.GuardianRelationships {
			key := edge.AdultExternalIdentifier + "\x00" + edge.StudentExternalIdentifier
			if seenRowEdges[key] {
				continue
			}
			seenRowEdges[key] = true
			if child := children[edge.StudentExternalIdentifier]; child != nil && child.seenClassroom {
				row.GuardianRelationships = append(row.GuardianRelationships, edge)
			}
		}
		result.sourceRows = append(result.sourceRows, row)
	}
	return result, nil
}

type classroom struct{ ID, Label string }

func classroomFromGroups(groups []json.RawMessage) (classroom, bool) {
	for _, raw := range groups {
		object, err := objectValue(raw)
		if err != nil {
			continue
		}
		isReference := false
		id := stringField(object, identifierFields...)
		for _, field := range classFields {
			classRaw, ok := object[field]
			if !ok || classRaw == nil {
				continue
			}
			if value, ok := stringValue(classRaw); ok {
				isReference = isClassroomReference(value)
			} else if classObject, ok := objectValueOK(classRaw); ok {
				isReference = isClassroomReference(stringField(classObject, "type", "kind", "reference", "name"))
				if value := stringField(classObject, identifierFields...); value != "" {
					id = value
				}
			}
			if isReference {
				break
			}
		}
		if value, ok := boolValue(object["classroom"]); ok {
			isReference = isReference || value
		}
		if isReference {
			return classroom{ID: id, Label: stringField(object, labelFields...)}, true
		}
	}
	return classroom{}, false
}

// isClassroomReference recognizes both a bare "classroom" discriminator and
// the platform's fully qualified reference class names, such as
// "models.embedded.reference.ClassroomReference". A child's group list also
// carries SchoolReference and CasualGroupReference entries, so the trailing
// segment is the only part that decides enrolment.
func isClassroomReference(value string) bool {
	value = strings.TrimSpace(value)
	if index := strings.LastIndex(value, "."); index >= 0 {
		value = value[index+1:]
	}
	return strings.EqualFold(strings.TrimSuffix(value, "Reference"), "classroom")
}

func relationshipType(value string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MOM", "DAD", "PARENT":
		return "parent", true
	case "GRANDMA":
		return "grandparent", true
	case "GUARDIAN":
		return "guardian", true
	case "UNCLE":
		return "other", true
	default:
		return "other", false
	}
}

func normalName(value string) string { return strings.Join(strings.Fields(value), " ") }

func objectValue(raw json.RawMessage) (map[string]json.RawMessage, error) {
	value, ok := objectValueOK(raw)
	if !ok {
		return nil, errors.New("expected JSON object")
	}
	return value, nil
}

func objectValueOK(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return nil, false
	}
	return value, true
}

func objectField(object map[string]json.RawMessage, names ...string) (map[string]json.RawMessage, error) {
	for _, name := range names {
		if raw, ok := object[name]; ok {
			value, valid := objectValueOK(raw)
			if !valid {
				return nil, fmt.Errorf("field %q is not an object", name)
			}
			return value, nil
		}
	}
	return nil, errors.New("relationship has no child object")
}

func rawList(object map[string]json.RawMessage, name string) []json.RawMessage {
	raw, ok := object[name]
	if !ok {
		return nil
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil
	}
	return values
}

func stringField(object map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		if raw, ok := object[name]; ok {
			if value, valid := stringValue(raw); valid {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func stringValue(raw json.RawMessage) (string, bool) {
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func boolValue(raw json.RawMessage) (bool, bool) {
	if raw == nil {
		return false, false
	}
	var value bool
	if json.Unmarshal(raw, &value) != nil {
		return false, false
	}
	return value, true
}
