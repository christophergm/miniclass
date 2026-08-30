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

// Student is the canonical student record emitted by the roster source. The
// classroom label and band are display metadata only; neither is interpreted
// as a grade (ADR 0014).
type Student struct {
	SourceExternalIdentifier    string `json:"source_external_identifier"`
	ExternalIdentifier          string `json:"external_identifier"`
	GivenName                   string `json:"given_name"`
	FamilyName                  string `json:"family_name"`
	LegalGivenName              string `json:"legal_given_name"`
	LegalFamilyName             string `json:"legal_family_name"`
	ClassroomExternalIdentifier string `json:"classroom_external_identifier"`
	ClassroomID                 string `json:"classroom_id"`
	ClassroomLabel              string `json:"classroom_label"`
	ClassroomBand               string `json:"classroom_band"`
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
}

// ParseJSON parses one wide roster export. It returns an error for malformed
// or empty documents and for contradictory names for one child identifier.
func ParseJSON(document []byte) (Result, error) {
	if len(bytes.TrimSpace(document)) == 0 {
		return Result{}, errors.New("roster: empty JSON document")
	}
	var records []json.RawMessage
	if err := json.Unmarshal(document, &records); err != nil {
		return Result{}, fmt.Errorf("roster: decode JSON: %w", err)
	}
	if len(records) == 0 {
		return Result{}, errors.New("roster: empty JSON array")
	}

	return parseRecords(records)
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

	for recordIndex, raw := range records {
		object, err := objectValue(raw)
		if err != nil {
			return Result{}, fmt.Errorf("roster: adult record %d: %w", recordIndex+1, err)
		}
		adultID := stringField(object, "id", "external_id", "external_identifier")
		if adultID == "" {
			return Result{}, fmt.Errorf("roster: adult record %d has no opaque id", recordIndex+1)
		}
		adult := &adultRecord{Adult: Adult{
			SourceExternalIdentifier: adultID,
			ExternalIdentifier:       adultID,
			Email:                    stringField(object, "email"),
			GivenName:                stringField(object, "given_name", "first_name"),
			FamilyName:               stringField(object, "family_name", "last_name"),
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
			childID := stringField(childObject, "id", "external_id", "external_identifier")
			if childID == "" {
				return Result{}, fmt.Errorf("roster: adult %q relationship %d child has no opaque id", adultID, relationshipIndex+1)
			}
			givenName := stringField(childObject, "given_name", "first_name")
			familyName := stringField(childObject, "family_name", "last_name")
			classroom, hasClassroom := classroomFromGroups(rawList(childObject, "groups"))
			if existing, ok := children[childID]; ok {
				if normalName(existing.GivenName) != normalName(givenName) || normalName(existing.FamilyName) != normalName(familyName) {
					return Result{}, fmt.Errorf("roster: child %q has contradictory names: %q %q and %q %q", childID, existing.GivenName, existing.FamilyName, givenName, familyName)
				}
				if !existing.seenClassroom && hasClassroom {
					existing.ClassroomExternalIdentifier = classroom.ID
					existing.ClassroomID = classroom.ID
					existing.ClassroomLabel = classroom.Label
					existing.ClassroomBand = classroom.Band
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
					child.ClassroomID = classroom.ID
					child.ClassroomLabel = classroom.Label
					child.ClassroomBand = classroom.Band
					child.seenClassroom = true
				}
				children[childID] = child
				childOrder = append(childOrder, childID)
			}

			child := children[childID]
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
			edgeKey := adultID + "\x00" + childID
			if !seenEdges[edgeKey] {
				seenEdges[edgeKey] = true
				if child.seenClassroom {
					adult.relationships = append(adult.relationships, edge)
				}
			}
		}
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
	return result, nil
}

type classroom struct{ ID, Label, Band string }

func classroomFromGroups(groups []json.RawMessage) (classroom, bool) {
	for _, raw := range groups {
		object, err := objectValue(raw)
		if err != nil {
			continue
		}
		classRaw := object["class"]
		isReference := false
		id := stringField(object, "id", "external_id", "external_identifier")
		if classRaw != nil {
			if value, ok := stringValue(classRaw); ok {
				isReference = strings.EqualFold(strings.TrimSpace(value), "classroom")
			} else if classObject, ok := objectValueOK(classRaw); ok {
				kind := stringField(classObject, "type", "kind", "reference", "name")
				isReference = strings.EqualFold(strings.TrimSpace(kind), "classroom")
				if value := stringField(classObject, "id", "external_id", "external_identifier"); value != "" {
					id = value
				}
			}
		}
		if !isReference {
			kind := stringField(object, "type", "kind", "reference")
			isReference = strings.EqualFold(strings.TrimSpace(kind), "classroom")
		}
		if value, ok := boolValue(object["classroom"]); ok {
			isReference = isReference || value
		}
		if isReference {
			return classroom{ID: id, Label: stringField(object, "label", "name"), Band: stringField(object, "band", "band_string")}, true
		}
	}
	return classroom{}, false
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
