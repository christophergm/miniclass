package program

import (
	"testing"

	"github.com/chrismott/miniclass/internal/data"
	"github.com/chrismott/miniclass/internal/ids"
)

func TestMatchesAutoAssignment(t *testing.T) {
	grade1, grade2 := ids.XID("grade-1"), ids.XID("grade-2")
	roomA, roomB := ids.XID("room-a"), ids.XID("room-b")

	tests := []struct {
		name    string
		program data.Program
		grade   ids.XID
		room    ids.XID
		want    bool
	}{
		{name: "empty criteria match every student", program: data.Program{}, grade: grade1, room: roomA, want: true},
		{name: "grade-only match", program: data.Program{AutoAssignmentGradeLevelIDs: []ids.XID{grade1}}, grade: grade1, room: roomB, want: true},
		{name: "grade-only mismatch", program: data.Program{AutoAssignmentGradeLevelIDs: []ids.XID{grade1}}, grade: grade2, room: roomA, want: false},
		{name: "homeroom-only match", program: data.Program{AutoAssignmentHomeroomIDs: []ids.XID{roomA}}, grade: grade2, room: roomA, want: true},
		{name: "homeroom-only mismatch", program: data.Program{AutoAssignmentHomeroomIDs: []ids.XID{roomA}}, grade: grade1, room: roomB, want: false},
		{name: "both criteria match", program: data.Program{AutoAssignmentGradeLevelIDs: []ids.XID{grade1}, AutoAssignmentHomeroomIDs: []ids.XID{roomA}}, grade: grade1, room: roomA, want: true},
		{name: "both criteria require grade and homeroom", program: data.Program{AutoAssignmentGradeLevelIDs: []ids.XID{grade1}, AutoAssignmentHomeroomIDs: []ids.XID{roomA}}, grade: grade1, room: roomB, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := matchesAutoAssignment(test.program, test.grade, test.room); got != test.want {
				t.Fatalf("matchesAutoAssignment() = %v, want %v", got, test.want)
			}
		})
	}
}
