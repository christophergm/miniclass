// Package seed owns the deterministic development corpus and its loader.
package seed

import (
	"fmt"
	"strings"

	"github.com/chrismott/miniclass/internal/data"
)

const (
	StudentCount = 139
	AdultCount   = 102
)

// StudentSpec is the deterministic, database-independent shape of one seed
// student. IDs are assigned only after insertion and are never used as keys.
type StudentSpec struct {
	LegalGivenName     string
	LegalFamilyName    string
	PreferredGivenName *string
	Grade              *int
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
	ParticipationIntent *data.AdultParticipationIntent
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
	Grades    []string
	Homerooms []string
	Students  []StudentSpec
	Adults    []AdultSpec
	Guardians []GuardianSpec
}

// Generate returns the same synthetic corpus on every call.
func Generate() Corpus {
	corpus := Corpus{
		Grades:    []string{"1", "2", "3", "4", "5", "6"},
		Homerooms: []string{"Red", "Blue", "Green", "Gold", "Purple", "Silver"},
		Students:  make([]StudentSpec, 0, StudentCount),
		Adults:    make([]AdultSpec, 0, AdultCount),
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
				Grade: gradePointer(grade+1, index != 0), Homeroom: index % len(corpus.Homerooms), ExternalIdentifier: external,
			})
		}
	}

	for index := 0; index < AdultCount; index++ {
		var intent *data.AdultParticipationIntent
		if index != 101 {
			value := data.AdultParticipationUnavailable
			intent = &value
		}
		if index < 13 {
			value := data.AdultParticipationLead
			intent = &value
		} else if index < 58 {
			value := data.AdultParticipationHelp
			intent = &value
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

	// The guardian edge is the only family construct (SPEC §8.2), so the shape of
	// this graph is the only thing that decides what a family looks like in a
	// development database. Left uniform it would be one adult per child, and
	// every surface that derives "the students this adult guards" would be
	// exercised only in the case where the derivation is uninteresting. The bulk
	// loops below give the ordinary case; the named block after them carries the
	// shapes Validate refuses to lose.
	relationshipTypes := [...]data.GuardianRelationshipType{
		data.GuardianRelationshipParent, data.GuardianRelationshipGuardian,
		data.GuardianRelationshipGrandparent, data.GuardianRelationshipOther,
	}
	for index := 0; index < 100; index++ {
		corpus.Guardians = append(corpus.Guardians, GuardianSpec{
			AdultIndex: index, StudentIndex: index, RelationshipType: relationshipTypes[index%len(relationshipTypes)],
		})
	}
	// Siblings: adults 0 to 36 guard a second child, so the derived scope is a
	// set rather than a singleton for a good share of the roster.
	for index := 100; index < 137; index++ {
		corpus.Guardians = append(corpus.Guardians, GuardianSpec{
			AdultIndex: index - 100, StudentIndex: index, RelationshipType: relationshipTypes[index%len(relationshipTypes)],
		})
	}
	corpus.Guardians = append(corpus.Guardians,
		// A separated family. Student 137's two guardians have no other child
		// between them, so nothing in the data links those two adults — which is
		// the point: after ADR 0012 nothing can. SPEC §8.2 records that the
		// reference program ran a whole second survey for these families.
		GuardianSpec{AdultIndex: 98, StudentIndex: 137, RelationshipType: data.GuardianRelationshipParent},
		GuardianSpec{AdultIndex: 99, StudentIndex: 137, RelationshipType: data.GuardianRelationshipParent},
		// An adult across two families. Adult 97 guards students 95 and 96, whose
		// other guardians are different adults, so this adult's derived scope
		// spans a boundary a stored grouping would have had to choose one side of.
		GuardianSpec{AdultIndex: 97, StudentIndex: 95, RelationshipType: data.GuardianRelationshipGrandparent},
		GuardianSpec{AdultIndex: 97, StudentIndex: 96, RelationshipType: data.GuardianRelationshipGrandparent},
	)
	// Student 138 is left with no guardian at all, and adults 100 and 101 with no
	// student. Both absences are deliberate; see Validate.
	return corpus
}

// Validate checks the corpus invariants without requiring PostgreSQL.
func (c Corpus) Validate() error {
	if len(c.Grades) != 6 || len(c.Homerooms) != 6 || len(c.Students) != StudentCount || len(c.Adults) != AdultCount {
		return fmt.Errorf("corpus dimensions are grades=%d homerooms=%d students=%d adults=%d", len(c.Grades), len(c.Homerooms), len(c.Students), len(c.Adults))
	}
	gradeCounts := make([]int, len(c.Grades))
	ungraded := 0
	for index, student := range c.Students {
		if student.Grade == nil {
			ungraded++
		} else if *student.Grade < 1 || *student.Grade > len(c.Grades) {
			return fmt.Errorf("student %d has invalid grade or homeroom", index)
		}
		if student.Homeroom < 0 || student.Homeroom >= len(c.Homerooms) {
			return fmt.Errorf("student %d has invalid grade or homeroom", index)
		}
		if student.Grade != nil {
			gradeCounts[*student.Grade-1]++
		}
	}
	wantGrades := []int{19, 27, 22, 21, 30, 19}
	for index, want := range wantGrades {
		if gradeCounts[index] != want {
			return fmt.Errorf("grade %d has %d students, want %d", index+1, gradeCounts[index], want)
		}
	}
	if ungraded != 1 {
		return fmt.Errorf("want one ungraded student, got %d", ungraded)
	}
	participation := map[data.AdultParticipationIntent]int{}
	undeclared := 0
	for _, adult := range c.Adults {
		if adult.ParticipationIntent == nil {
			undeclared++
			participation[""]++
		} else {
			participation[*adult.ParticipationIntent]++
		}
	}
	if participation[data.AdultParticipationLead] != 13 || participation[data.AdultParticipationHelp] != 45 || participation[data.AdultParticipationUnavailable] != 43 {
		return fmt.Errorf("adult participation split is lead=%d help=%d unavailable=%d", participation[data.AdultParticipationLead], participation[data.AdultParticipationHelp], participation[data.AdultParticipationUnavailable])
	}
	if undeclared != 1 {
		return fmt.Errorf("want one adult without a participation intent, got %d", undeclared)
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
	// The guardian graph is the whole of the family model now, so its deliberate
	// shapes are invariants rather than incidental data. Each check below is
	// stated as "some such case exists" rather than by index, so that moving a
	// case is allowed and losing one is not.
	guardedStudents := make([]map[int]bool, len(c.Adults))
	studentGuardians := make([]map[int]bool, len(c.Students))
	for index, relationship := range c.Guardians {
		if relationship.AdultIndex < 0 || relationship.AdultIndex >= len(c.Adults) || relationship.StudentIndex < 0 || relationship.StudentIndex >= len(c.Students) {
			return fmt.Errorf("guardian relationship %d references an unknown adult or student", index)
		}
		if guardedStudents[relationship.AdultIndex] == nil {
			guardedStudents[relationship.AdultIndex] = map[int]bool{}
		}
		if studentGuardians[relationship.StudentIndex] == nil {
			studentGuardians[relationship.StudentIndex] = map[int]bool{}
		}
		guardedStudents[relationship.AdultIndex][relationship.StudentIndex] = true
		studentGuardians[relationship.StudentIndex][relationship.AdultIndex] = true
	}
	if len(guardedStudents[100]) != 0 || len(guardedStudents[101]) != 0 {
		return fmt.Errorf("adults 100 and 101 must guard no student: they are the non-guardian volunteers")
	}
	if !hasUnguardedStudent(studentGuardians) {
		return fmt.Errorf("no student without a guardian: SPEC \u00a78.2's warning has nothing to fire on")
	}
	if !hasSeparatedFamily(studentGuardians, guardedStudents) {
		return fmt.Errorf("no student with two guardians who share no other student")
	}
	if !hasAdultAcrossFamilies(guardedStudents, studentGuardians) {
		return fmt.Errorf("no adult guarding co-guarded students that share no other guardian")
	}
	return nil
}

func gradePointer(value int, present bool) *int {
	if !present {
		return nil
	}
	return &value
}

// hasUnguardedStudent reports the case SPEC §8.2 requires a warning for, and
// never a block. A warning with no data behind it is a warning nobody has seen,
// so the development corpus carries one on purpose.
func hasUnguardedStudent(studentGuardians []map[int]bool) bool {
	for _, guardians := range studentGuardians {
		if len(guardians) == 0 {
			return true
		}
	}
	return false
}

// hasSeparatedFamily reports one student with two guardians who guard no other
// child in common. Nothing else in the data relates those two adults, because
// after ADR 0012 there is nothing else that could.
func hasSeparatedFamily(studentGuardians, guardedStudents []map[int]bool) bool {
	for student, guardians := range studentGuardians {
		for left := range guardians {
			for right := range guardians {
				if left < right && !sharesOther(guardedStudents[left], guardedStudents[right], student) {
					return true
				}
			}
		}
	}
	return false
}

// hasAdultAcrossFamilies reports an adult guarding two students that each have
// another guardian, and not the same other guardian. Requiring both students to
// be co-guarded is what separates this from a pair of siblings, who share every
// guardian they have; it is also what keeps the check from passing on a corpus
// where every child has exactly one adult.
func hasAdultAcrossFamilies(guardedStudents, studentGuardians []map[int]bool) bool {
	for adult, students := range guardedStudents {
		for left := range students {
			for right := range students {
				if left >= right || len(studentGuardians[left]) < 2 || len(studentGuardians[right]) < 2 {
					continue
				}
				if !sharesOther(studentGuardians[left], studentGuardians[right], adult) {
					return true
				}
			}
		}
	}
	return false
}

// sharesOther reports whether two index sets have a member in common other than
// the one they are already known to share.
func sharesOther(left, right map[int]bool, except int) bool {
	for index := range left {
		if index != except && right[index] {
			return true
		}
	}
	return false
}
