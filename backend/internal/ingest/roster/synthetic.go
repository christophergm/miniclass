package roster

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
)

const (
	SyntheticAdultCount   = 324
	SyntheticChildCount   = 247
	SyntheticEnrolled     = 185
	SyntheticClassroomCnt = 8
)

// GenerateSyntheticJSON returns the deterministic, entirely synthetic wide
// export used by the parser's golden tests. It mirrors the observed source
// dimensions without copying historical names or contact details.
func GenerateSyntheticJSON() []byte {
	type group struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Class string `json:"class"`
		Band  string `json:"band"`
	}
	type child struct {
		ID         string  `json:"id"`
		GivenName  string  `json:"given_name"`
		FamilyName string  `json:"family_name"`
		Status     string  `json:"status"`
		Groups     []group `json:"groups"`
	}
	type relationship struct {
		Child        child  `json:"child"`
		Relationship string `json:"relationship"`
	}
	type adult struct {
		ID            string         `json:"id"`
		Email         string         `json:"email,omitempty"`
		GivenName     string         `json:"given_name"`
		FamilyName    string         `json:"family_name"`
		Status        string         `json:"status"`
		Relationships []relationship `json:"relationships"`
		Unread        map[string]any `json:"profile_metadata,omitempty"`
	}

	children := make([]child, SyntheticChildCount)
	for index := range children {
		given, family := fmt.Sprintf("Given%03d", index+1), fmt.Sprintf("Family%02d", index%31+1)
		if index == 2 {
			family = "De La Sample"
		}
		if index == 17 {
			given, family = children[16].GivenName, children[16].FamilyName
		}
		children[index] = child{
			ID: fmt.Sprintf("synthetic-child-%03d", index+1), GivenName: given, FamilyName: family,
			Status: "active",
		}
		if index < SyntheticEnrolled {
			room := syntheticClassroomFor(index)
			bands := [...]string{"K", "1-2", "3", "4-5", "6"}
			children[index].Groups = []group{{ID: fmt.Sprintf("synthetic-classroom-%02d", room+1), Name: fmt.Sprintf("Room %02d", room+1), Class: "classroom", Band: bands[room%len(bands)]}}
		} else if index%2 == 0 {
			children[index].Groups = []group{{ID: fmt.Sprintf("synthetic-activity-%02d", index%7+1), Name: "Activity group", Class: "activity", Band: "not-a-grade"}}
		}
	}

	// The observed enrolled-child distribution is 108 with two guardians, 72
	// with one, and 5 with three: 303 edges in total. Assign one edge to every
	// retained adult first, then fill each student's target count without ever
	// repeating an adult/student edge.
	targets := make([]int, SyntheticEnrolled)
	for index := range targets {
		switch {
		case index < 108:
			targets[index] = 2
		case index < 180:
			targets[index] = 1
		default:
			targets[index] = 3
		}
	}
	edgesByStudent := make([][]int, SyntheticEnrolled)
	used := make(map[string]bool)
	for adultIndex := 0; adultIndex < 226; adultIndex++ {
		studentIndex := adultIndex % SyntheticEnrolled
		assignSyntheticEdge(edgesByStudent, used, studentIndex, adultIndex)
	}
	for studentIndex, target := range targets {
		for len(edgesByStudent[studentIndex]) < target {
			for offset := 0; ; offset++ {
				adultIndex := (studentIndex*29 + offset) % 226
				if assignSyntheticEdge(edgesByStudent, used, studentIndex, adultIndex) {
					break
				}
			}
		}
	}

	adults := make([]adult, SyntheticAdultCount)
	for index := range adults {
		adults[index] = adult{
			ID: fmt.Sprintf("synthetic-adult-%03d", index+1), Status: "active",
			Email:     fmt.Sprintf("synthetic-%03d@example.test", index+1),
			GivenName: fmt.Sprintf("Adult%03d", index+1), FamilyName: fmt.Sprintf("Contact%02d", index%17+1),
			Unread: map[string]any{"source_flag": index%9 == 0, "import_note": "synthetic"},
		}
		if index >= 226 && index < 296 { // invited accounts have no names
			adults[index].GivenName, adults[index].FamilyName = "", ""
			adults[index].Email = fmt.Sprintf("invited-%03d@example.test", index+1)
		}
		if index >= 296 && index < 308 { // named accounts with no relationships
			adults[index].GivenName = fmt.Sprintf("Unlinked%03d", index+1)
		}
		if index >= 308 { // named accounts whose only children are unenrolled
			adults[index].GivenName = fmt.Sprintf("Unenrolled%03d", index+1)
		}
	}

	labels := [...]string{"MOM", "DAD", "PARENT", "GRANDMA", "GUARDIAN", "UNCLE"}
	for studentIndex, adultIndexes := range edgesByStudent {
		for edgeIndex, adultIndex := range adultIndexes {
			childValue := children[studentIndex]
			adults[adultIndex].Relationships = append(adults[adultIndex].Relationships, relationship{
				Child: childValue, Relationship: labels[(studentIndex+edgeIndex)%len(labels)],
			})
		}
	}
	for index := 308; index < SyntheticAdultCount; index++ {
		childIndex := SyntheticEnrolled + (index - 308)
		childValue := children[childIndex]
		adults[index].Relationships = []relationship{{Child: childValue, Relationship: "PARENT"}}
	}
	for index := 226; index < 296; index++ {
		childValue := children[SyntheticEnrolled+(index-226)%(SyntheticChildCount-SyntheticEnrolled)]
		adults[index].Relationships = []relationship{{Child: childValue, Relationship: "PARENT"}}
	}

	encoded, err := json.MarshalIndent(adults, "", "  ")
	if err != nil {
		panic(fmt.Sprintf("roster: encode synthetic corpus: %v", err))
	}
	return append(encoded, '\n')
}

func syntheticClassroomFor(index int) int {
	counts := [...]int{25, 25, 25, 25, 24, 21, 21, 19}
	for room, count := range counts {
		if index < count {
			return room
		}
		index -= count
	}
	panic("roster: synthetic classroom index out of range")
}

func assignSyntheticEdge(edgesByStudent [][]int, used map[string]bool, studentIndex, adultIndex int) bool {
	key := strconv.Itoa(studentIndex) + ":" + strconv.Itoa(adultIndex)
	if used[key] {
		return false
	}
	used[key] = true
	edgesByStudent[studentIndex] = append(edgesByStudent[studentIndex], adultIndex)
	return true
}

// GenerateSyntheticGradesCSV returns a deterministic two-column companion
// source for the grades_csv parser. It is intentionally separate from the
// roster JSON because the JSON source carries no grade authority.
func GenerateSyntheticGradesCSV() []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"student_name", "grade"})
	for index := 0; index < SyntheticChildCount; index++ {
		grade := strconv.Itoa(index%6 + 1)
		if index >= SyntheticEnrolled {
			grade = ""
		}
		given, family := fmt.Sprintf("Given%03d", index+1), fmt.Sprintf("Family%02d", index%31+1)
		if index == 2 {
			family = "De La Sample"
		}
		if index == 17 {
			given, family = "Given017", "Family17"
		}
		_ = writer.Write([]string{given + " " + family, grade})
	}
	writer.Flush()
	return buffer.Bytes()
}

// GenerateSyntheticEdgeCasesJSON is a small golden source for cases that did
// not occur in the historical export but are required to keep the parser
// honest: contradictory source names, duplicate normalized names, and a
// two-word family name.
func GenerateSyntheticEdgeCasesJSON() []byte {
	return []byte(`[
  {"id":"synthetic-edge-adult-1","given_name":"Edge","family_name":"Adult","relationships":[
    {"relationship":"PARENT","child":{"id":"synthetic-edge-child-1","given_name":"Alex","family_name":"De La Cruz","groups":[{"id":"synthetic-edge-room","name":"Edge Room","class":"classroom","band":"display only"}]}},
    {"relationship":"MOM","child":{"id":"synthetic-edge-child-2","given_name":"Same","family_name":"Name","groups":[{"id":"synthetic-edge-room","name":"Edge Room","class":"classroom","band":"display only"}]}}
  ]},
  {"id":"synthetic-edge-adult-2","given_name":"Second","family_name":"Adult","relationships":[
    {"relationship":"DAD","child":{"id":"synthetic-edge-child-2","given_name":"Same","family_name":"Name","groups":[{"id":"synthetic-edge-room","name":"Edge Room","class":"classroom","band":"display only"}]}}
  ]},
  {"id":"synthetic-edge-adult-3","given_name":"Contradiction","family_name":"Adult","relationships":[
    {"relationship":"PARENT","child":{"id":"synthetic-edge-contradiction","given_name":"First","family_name":"Version","groups":[{"id":"synthetic-edge-room","class":"classroom"}]}},
    {"relationship":"PARENT","child":{"id":"synthetic-edge-contradiction","given_name":"Second","family_name":"Version","groups":[{"id":"synthetic-edge-room","class":"classroom"}]}}
  ]}
]
`)
}
