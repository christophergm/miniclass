// Package seed owns the deterministic development corpus and its loader.
package seed

import (
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/data"
)

const (
	StudentCount   = 139
	AdultCount     = 102
	HouseholdCount = 90
)

// StudentSpec is the deterministic, database-independent shape of one seed
// student. IDs are assigned only after insertion and are never used as keys.
type StudentSpec struct {
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	Grade              int
	Homeroom           int
	ExternalIdentifier *string
}

type AdultSpec struct {
	LegalGivenName      string
	LegalFamilyName     string
	PreferredGivenName  *string
	Email               *string
	Phone               *string
	ExternalIdentifier  *string
	ParticipationIntent data.AdultParticipationIntent
}

type HouseholdSpec struct {
	DisplayName string
}

type StudentMembershipSpec struct {
	StudentIndex   int
	HouseholdIndex int
}

type AdultMembershipSpec struct {
	AdultIndex     int
	HouseholdIndex int
}

type GuardianSpec struct {
	AdultIndex       int
	StudentIndex     int
	RelationshipType data.GuardianRelationshipType
}

// Corpus is the complete deterministic input graph. It contains indexes, not
// names, in relationships so duplicate family names and two-word surnames
// cannot affect joins.
type Corpus struct {
	Grades             []string
	Homerooms          []string
	Students           []StudentSpec
	Adults             []AdultSpec
	Households         []HouseholdSpec
	StudentMemberships []StudentMembershipSpec
	AdultMemberships   []AdultMembershipSpec
	Guardians          []GuardianSpec
}

// Generate returns the same synthetic corpus on every call.
func Generate() Corpus {
	corpus := Corpus{
		Grades:     []string{"1", "2", "3", "4", "5", "6"},
		Homerooms:  []string{"Red", "Blue", "Green", "Gold", "Purple", "Silver"},
		Students:   make([]StudentSpec, 0, StudentCount),
		Adults:     make([]AdultSpec, 0, AdultCount),
		Households: make([]HouseholdSpec, 0, HouseholdCount),
	}

	gradeCounts := [...]int{20, 27, 22, 21, 30, 19}
	for grade, count := range gradeCounts {
		for offset := 0; offset < count; offset++ {
			index := len(corpus.Students)
			given := fmt.Sprintf("Synthetic Given %03d", index+1)
			family := fmt.Sprintf("Synthetic Family %02d", (index%30)+1)
			if index == 2 {
				family = "Synthetic De La Sample"
			}
			var preferred *string
			if index%17 == 0 {
				value := fmt.Sprintf("Preferred %03d", index+1)
				preferred = &value
			}
			var external *string
			if index != 0 {
				value := fmt.Sprintf("synthetic-student-%03d", index+1)
				external = &value
			}
			corpus.Students = append(corpus.Students, StudentSpec{
				LegalGivenName: given, LegalFamilyName: family, PreferredGivenName: preferred,
				Grade: grade + 1, Homeroom: index % len(corpus.Homerooms), ExternalIdentifier: external,
			})
		}
	}

	for index := 0; index < AdultCount; index++ {
		intent := data.AdultParticipationUnavailable
		if index < 13 {
			intent = data.AdultParticipationLead
		} else if index < 58 {
			intent = data.AdultParticipationHelp
		}
		var email *string
		if index < 100 {
			value := fmt.Sprintf("synthetic-adult-%03d@example.test", index+1)
			email = &value
		}
		var external *string
		if index != 101 {
			value := fmt.Sprintf("synthetic-adult-%03d", index+1)
			external = &value
		}
		family := fmt.Sprintf("Synthetic Family %02d", (index%30)+1)
		corpus.Adults = append(corpus.Adults, AdultSpec{
			LegalGivenName: fmt.Sprintf("Synthetic Adult %03d", index+1), LegalFamilyName: family,
			Email: email, ExternalIdentifier: external, ParticipationIntent: intent,
		})
	}

	for index := 0; index < HouseholdCount; index++ {
		corpus.Households = append(corpus.Households, HouseholdSpec{DisplayName: fmt.Sprintf("Synthetic Household %03d", index+1)})
		corpus.StudentMemberships = append(corpus.StudentMemberships, StudentMembershipSpec{StudentIndex: index, HouseholdIndex: index})
		corpus.AdultMemberships = append(corpus.AdultMemberships, AdultMembershipSpec{AdultIndex: index, HouseholdIndex: index})
	}
	for index := 90; index < 137; index++ {
		corpus.StudentMemberships = append(corpus.StudentMemberships, StudentMembershipSpec{StudentIndex: index, HouseholdIndex: index - 90})
	}
	corpus.StudentMemberships = append(corpus.StudentMemberships,
		StudentMembershipSpec{StudentIndex: 0, HouseholdIndex: 1},
		StudentMembershipSpec{StudentIndex: 1, HouseholdIndex: 2},
	)
	for index := 90; index < 100; index++ {
		corpus.AdultMemberships = append(corpus.AdultMemberships, AdultMembershipSpec{AdultIndex: index, HouseholdIndex: index - 90})
	}
	corpus.AdultMemberships = append(corpus.AdultMemberships, AdultMembershipSpec{AdultIndex: 0, HouseholdIndex: 1})

	relationshipTypes := [...]data.GuardianRelationshipType{
		data.GuardianRelationshipParent, data.GuardianRelationshipGuardian,
		data.GuardianRelationshipGrandparent, data.GuardianRelationshipOther,
	}
	for index := 0; index < 100; index++ {
		corpus.Guardians = append(corpus.Guardians, GuardianSpec{
			AdultIndex: index, StudentIndex: index % 137, RelationshipType: relationshipTypes[index%len(relationshipTypes)],
		})
	}
	return corpus
}

// Validate checks the corpus invariants without requiring PostgreSQL.
func (c Corpus) Validate() error {
	if len(c.Grades) != 6 || len(c.Homerooms) != 6 || len(c.Students) != StudentCount || len(c.Adults) != AdultCount || len(c.Households) != HouseholdCount {
		return fmt.Errorf("corpus dimensions are grades=%d homerooms=%d students=%d adults=%d households=%d", len(c.Grades), len(c.Homerooms), len(c.Students), len(c.Adults), len(c.Households))
	}
	gradeCounts := make([]int, len(c.Grades))
	for index, student := range c.Students {
		if student.Grade < 1 || student.Grade > len(c.Grades) || student.Homeroom < 0 || student.Homeroom >= len(c.Homerooms) {
			return fmt.Errorf("student %d has invalid grade or homeroom", index)
		}
		gradeCounts[student.Grade-1]++
	}
	wantGrades := []int{20, 27, 22, 21, 30, 19}
	for index, want := range wantGrades {
		if gradeCounts[index] != want {
			return fmt.Errorf("grade %d has %d students, want %d", index+1, gradeCounts[index], want)
		}
	}
	participation := map[data.AdultParticipationIntent]int{}
	for _, adult := range c.Adults {
		participation[adult.ParticipationIntent]++
	}
	if participation[data.AdultParticipationLead] != 13 || participation[data.AdultParticipationHelp] != 45 || participation[data.AdultParticipationUnavailable] != 44 {
		return fmt.Errorf("adult participation split is lead=%d help=%d unavailable=%d", participation[data.AdultParticipationLead], participation[data.AdultParticipationHelp], participation[data.AdultParticipationUnavailable])
	}
	if c.Students[0].ExternalIdentifier != nil || c.Students[2].LegalFamilyName != "Synthetic De La Sample" {
		return fmt.Errorf("student edge cases are missing")
	}
	if strings.TrimSpace(c.Students[0].LegalFamilyName) == "" {
		return fmt.Errorf("student family name is empty")
	}
	familyNames := map[string]int{}
	homerooms := map[int]bool{}
	for _, student := range c.Students {
		familyNames[student.LegalFamilyName]++
		homerooms[student.Homeroom] = true
	}
	if familyNames["Synthetic Family 01"] < 2 || len(homerooms) != 6 {
		return fmt.Errorf("student name or homeroom edge cases are missing")
	}
	studentMemberships := map[int]int{}
	for _, membership := range c.StudentMemberships {
		studentMemberships[membership.StudentIndex]++
	}
	if studentMemberships[0] < 2 || studentMemberships[1] < 2 || studentMemberships[137] != 0 || studentMemberships[138] != 0 {
		return fmt.Errorf("student household edge cases are missing")
	}
	adultMemberships := map[int]int{}
	for _, membership := range c.AdultMemberships {
		adultMemberships[membership.AdultIndex]++
	}
	guardianRelationships := map[int]int{}
	for _, relationship := range c.Guardians {
		guardianRelationships[relationship.AdultIndex]++
	}
	if adultMemberships[0] < 2 || adultMemberships[100] != 0 || adultMemberships[101] != 0 || guardianRelationships[100] != 0 || guardianRelationships[101] != 0 {
		return fmt.Errorf("adult relationship edge cases are missing")
	}
	return nil
}
