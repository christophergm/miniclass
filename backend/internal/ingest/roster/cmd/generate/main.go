package main

import (
	"os"
	"path/filepath"

	"github.com/chrismott/miniclass/internal/ingest/roster"
)

func main() {
	directory := "testdata"
	if err := os.MkdirAll(directory, 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "synthetic_roster.json"), roster.GenerateSyntheticJSON(), 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "synthetic_grades.csv"), roster.GenerateSyntheticGradesCSV(), 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "synthetic_edge_cases.json"), roster.GenerateSyntheticEdgeCasesJSON(), 0o644); err != nil {
		panic(err)
	}
}
