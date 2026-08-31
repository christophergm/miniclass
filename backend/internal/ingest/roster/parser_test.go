package roster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSyntheticCorpusMatchesObservedShape(t *testing.T) {
	first := GenerateSyntheticJSON()
	second := GenerateSyntheticJSON()
	require.Equal(t, first, second)

	result, err := ParseJSON(first)
	require.NoError(t, err)
	require.Len(t, result.Students, 185)
	require.Len(t, result.Adults, 226)
	require.Len(t, result.GuardianRelationships, 303)
	require.Len(t, result.ExcludedChildren, 62)
	require.Len(t, result.ExcludedAdults, 98)

	adultReasons := map[string]int{}
	for _, exclusion := range result.ExcludedAdults {
		adultReasons[exclusion.Reason]++
	}
	require.Equal(t, map[string]int{
		AdultExclusionNoName: 70, AdultExclusionNoChildren: 12, AdultExclusionNoneEnrolled: 16,
	}, adultReasons)
	for _, student := range result.Students {
		require.NotEmpty(t, student.ClassroomExternalIdentifier)
	}
	require.NotEmpty(t, result.Students[2].ClassroomBand)
	require.Equal(t, "De La Sample", result.Students[2].FamilyName)
	classroomCounts := map[string]int{}
	for _, student := range result.Students {
		classroomCounts[student.ClassroomExternalIdentifier]++
	}
	require.Equal(t, []int{25, 25, 25, 25, 24, 21, 21, 19}, []int{
		classroomCounts["synthetic-classroom-01"], classroomCounts["synthetic-classroom-02"],
		classroomCounts["synthetic-classroom-03"], classroomCounts["synthetic-classroom-04"],
		classroomCounts["synthetic-classroom-05"], classroomCounts["synthetic-classroom-06"],
		classroomCounts["synthetic-classroom-07"], classroomCounts["synthetic-classroom-08"],
	})

	nameCounts := map[string]int{}
	for _, student := range result.Students {
		nameCounts[student.GivenName+"\x00"+student.FamilyName]++
	}
	require.Contains(t, nameCounts, "Given017\x00Family17")
	require.Equal(t, 2, nameCounts["Given017\x00Family17"])
}

func TestGoldenFilesAreGeneratedAndParseable(t *testing.T) {
	root := "testdata"
	jsonGolden, err := os.ReadFile(filepath.Join(root, "synthetic_roster.json"))
	require.NoError(t, err)
	gradesGolden, err := os.ReadFile(filepath.Join(root, "synthetic_grades.csv"))
	require.NoError(t, err)
	edgeGolden, err := os.ReadFile(filepath.Join(root, "synthetic_edge_cases.json"))
	require.NoError(t, err)
	require.Equal(t, GenerateSyntheticJSON(), jsonGolden)
	require.Equal(t, GenerateSyntheticGradesCSV(), gradesGolden)
	require.Equal(t, GenerateSyntheticEdgeCasesJSON(), edgeGolden)
	result, err := ParseJSON(jsonGolden)
	require.NoError(t, err)
	require.Len(t, result.Students, SyntheticEnrolled)
	grades, err := ParseGradesCSV(strings.NewReader(string(gradesGolden)))
	require.NoError(t, err)
	require.Len(t, grades, SyntheticChildCount)
	_, err = ParseJSON(edgeGolden)
	require.Error(t, err, "the contradiction fixture must remain a failing source")
}

func TestParseResolvesFieldsByNameAndIgnoresUnknownFields(t *testing.T) {
	document := `[
      {"unread": {"anything": true}, "relationships": [{
        "child": {"groups": [{"name":"Room 9", "band":"3rd-4th Grade", "class":"classroom", "id":"room-9"}], "family_name":"Example", "id":"child-1", "given_name":"Casey", "new_field":"ignored"},
        "relationship": " dad "
      }], "status":"active", "family_name":"Adult", "id":"adult-1", "given_name":"Alex", "email":"alex@example.test"}
    ]`
	result, err := ParseJSON([]byte(document))
	require.NoError(t, err)
	require.Equal(t, "Casey", result.Students[0].GivenName)
	require.Equal(t, "Example", result.Students[0].FamilyName)
	require.Equal(t, "3rd-4th Grade", result.Students[0].ClassroomBand)
	require.Equal(t, "parent", result.GuardianRelationships[0].RelationshipType)
	require.Empty(t, result.Warnings)
}

// The community platform exports Mongo-shaped documents: `_id`, `firstName`,
// and group membership discriminated by a fully qualified `_class`. This is
// the shape the importer exists to consume, so it is pinned without needing
// the real export on disk (SPEC §11.3).
func TestParseReadsPlatformExportShape(t *testing.T) {
	document := `[
      {
        "_id": "5f6b82f7dafa6f6540fd48e5", "email": "adult@example.test",
        "firstName": "Given", "lastName": "Family", "status": "ACTIVE",
        "isStaff": false, "thumbnailSequence": 100,
        "roles": [{"_class": "models.ClassroomRole", "roleName": "MEMBER", "entity": {"_class": "models.embedded.reference.ClassroomReference", "_id": "room-1"}}],
        "relationships": [
          {"relationship": "MOM", "child": {"_id": "66ccb3df1b5ca469692d1b1b", "firstName": "Child", "lastName": "OfMine", "groups": [
            {"_class": "models.embedded.reference.SchoolReference", "_id": "school-1", "name": "Program", "loc": [0.0, 0.0]},
            {"_class": "models.embedded.reference.ClassroomReference", "_id": "room-1", "name": "Serena", "grade": "3rd-4th Grade"}
          ]}},
          {"relationship": "DAD", "child": {"_id": "unenrolled-1", "firstName": "Casual", "lastName": "Member", "groups": [
            {"_class": "models.embedded.reference.SchoolReference", "_id": "school-1", "name": "Program"},
            {"_class": "models.embedded.reference.CasualGroupReference", "_id": "news-1", "name": "All School News"}
          ]}}
        ]
      }
    ]`
	result, err := ParseJSON([]byte(document))
	require.NoError(t, err)

	require.Len(t, result.Adults, 1)
	require.Equal(t, "5f6b82f7dafa6f6540fd48e5", result.Adults[0].SourceExternalIdentifier)
	require.Equal(t, "Given", result.Adults[0].GivenName)
	require.Equal(t, "Family", result.Adults[0].FamilyName)

	require.Len(t, result.Students, 1, "only the child in a ClassroomReference group is enrolled")
	require.Equal(t, "66ccb3df1b5ca469692d1b1b", result.Students[0].SourceExternalIdentifier)
	require.Equal(t, "Child", result.Students[0].GivenName)
	require.Equal(t, "room-1", result.Students[0].ClassroomExternalIdentifier)
	require.Equal(t, "Serena", result.Students[0].ClassroomLabel)
	require.Equal(t, "3rd-4th Grade", result.Students[0].ClassroomBand)

	require.Len(t, result.GuardianRelationships, 1)
	require.Equal(t, "parent", result.GuardianRelationships[0].RelationshipType)
	require.Len(t, result.ExcludedChildren, 1)
	require.Equal(t, "unenrolled-1", result.ExcludedChildren[0].SourceExternalIdentifier)
	require.Equal(t, "no_classroom", result.ExcludedChildren[0].Reason)
	require.Empty(t, result.Warnings)
}

func TestParseFiltersAndReportsWithoutBlocking(t *testing.T) {
	document := `[
      {"id":"named-no-children", "given_name":"No", "family_name":"Children", "relationships":[]},
      {"id":"nameless", "email":"invite@example.test", "relationships":[{"child":{"id":"enrolled","given_name":"En","family_name":"Rolled","groups":[{"id":"room","class":"classroom","name":"Room","band":"unknown"}]},"relationship":"MOM"}]},
      {"id":"none-enrolled", "given_name":"None", "family_name":"Enrolled", "relationships":[{"child":{"id":"not-enrolled","given_name":"Not","family_name":"Enrolled","groups":[]},"relationship":"mystery"}]},
      {"id":"kept", "given_name":"Has", "family_name":"Role", "relationships":[{"child":{"id":"enrolled","given_name":"En","family_name":"Rolled","groups":[{"id":"room","class":"classroom","name":"Room","band":"unknown"}]},"relationship":"PARENT"},{"child":{"id":"not-enrolled","given_name":"Not","family_name":"Enrolled","groups":[]},"relationship":"UNCLE"}]}
    ]`
	result, err := ParseJSON([]byte(document))
	require.NoError(t, err)
	require.Len(t, result.Students, 1)
	require.Len(t, result.Adults, 1)
	require.Equal(t, "kept", result.Adults[0].SourceExternalIdentifier)
	require.Len(t, result.ExcludedChildren, 1)
	require.Equal(t, "not-enrolled", result.ExcludedChildren[0].SourceExternalIdentifier)
	require.Equal(t, []string{AdultExclusionNoChildren, AdultExclusionNoName, AdultExclusionNoneEnrolled}, []string{
		result.ExcludedAdults[0].Reason, result.ExcludedAdults[1].Reason, result.ExcludedAdults[2].Reason,
	})
	require.Len(t, result.GuardianRelationships, 1)
	require.Equal(t, "parent", result.GuardianRelationships[0].RelationshipType)
	require.Len(t, result.Warnings, 1)
	require.Equal(t, "unrecognized_relationship", result.Warnings[0].Code)
}

func TestParseMapsRelationshipLabelsAndDeduplicatesEdges(t *testing.T) {
	labels := []string{"MOM", "DAD", "PARENT", "GRANDMA", "GUARDIAN", "UNCLE", "", "cousin"}
	for index, label := range labels {
		document := `[{"id":"adult","given_name":"A","family_name":"B","relationships":[{"child":{"id":"child","given_name":"C","family_name":"D","groups":[{"id":"room","type":"classroom"}]},"relationship":"` + label + `"},{"child":{"id":"child","given_name":"C","family_name":"D","groups":[{"id":"room","type":"classroom"}]},"relationship":"` + label + `"}]}]`
		result, err := ParseJSON([]byte(document))
		require.NoError(t, err)
		require.Len(t, result.GuardianRelationships, 1)
		want := "other"
		if index < 3 {
			want = "parent"
		} else if index == 3 {
			want = "grandparent"
		} else if index == 4 {
			want = "guardian"
		}
		require.Equal(t, want, result.GuardianRelationships[0].RelationshipType)
		if index < 6 {
			require.Empty(t, result.Warnings)
		} else {
			require.Len(t, result.Warnings, 2)
		}
	}
}

func TestParseContradictoryChildNamesIsAnError(t *testing.T) {
	document := `[{"id":"adult-1","given_name":"A","family_name":"B","relationships":[{"child":{"id":"child-1","given_name":"One","family_name":"Name","groups":[{"id":"room","class":"classroom"}]}},{"child":{"id":"child-1","given_name":"Other","family_name":"Name","groups":[{"id":"room","class":"classroom"}]}}]}]`
	_, err := ParseJSON([]byte(document))
	require.Error(t, err)
	require.Contains(t, err.Error(), `child "child-1"`)
	require.Contains(t, err.Error(), `"One" "Name"`)
	require.Contains(t, err.Error(), `"Other" "Name"`)
}

func TestParseRejectsMalformedAndEmptyDocuments(t *testing.T) {
	for _, document := range []string{"", "   ", "[]", "{not-json}", "[null]"} {
		_, err := ParseJSON([]byte(document))
		require.Error(t, err, document)
	}
}

func TestParseGradesCSVUsesNamedColumns(t *testing.T) {
	rows, err := ParseGradesCSV(strings.NewReader("extra,grade,student_name\nignored,4,Family De La Sample\n"))
	require.NoError(t, err)
	require.Equal(t, []GradeRecord{{StudentName: "Family De La Sample", Grade: "4"}}, rows)
}

func TestOptInRealExportConformance(t *testing.T) {
	path := os.Getenv("MINICLASS_ROSTER_JSON_PATH")
	if path == "" {
		t.Skip("set MINICLASS_ROSTER_JSON_PATH to run the opt-in real-export check")
	}
	document, err := os.ReadFile(path)
	require.NoError(t, err)
	result, err := ParseJSON(document)
	require.NoError(t, err)
	require.Len(t, result.Students, 185)
	require.Len(t, result.Adults, 226)
	require.Len(t, result.GuardianRelationships, 303)
	for _, student := range result.Students {
		found := false
		for _, edge := range result.GuardianRelationships {
			if edge.StudentExternalIdentifier == student.SourceExternalIdentifier {
				found = true
				break
			}
		}
		require.True(t, found, "student %s has no guardian", student.SourceExternalIdentifier)
	}
}
