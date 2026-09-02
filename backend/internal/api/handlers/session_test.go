package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/chrismott/miniclass/internal/api/problems"
	programservice "github.com/chrismott/miniclass/internal/program"
	"github.com/danielgtaylor/huma/v2"
)

func TestSessionProblemUsesSpecificTransitionGateDetail(t *testing.T) {
	cause := fmt.Errorf(
		"preview session transition: %w",
		fmt.Errorf(
			"%w: voting cannot open until the catalog has at least one offering",
			programservice.ErrSessionTransitionGate,
		),
	)

	problem := sessionProblem(cause)
	var model *huma.ErrorModel
	if !errors.As(problem, &model) {
		t.Fatalf("problem = %T, want *huma.ErrorModel", problem)
	}
	if model.Type != string(problems.SessionTransitionGate) || model.Status != http.StatusConflict {
		t.Fatalf("problem type/status = %q/%d", model.Type, model.Status)
	}
	if model.Detail != "voting cannot open until the catalog has at least one offering" {
		t.Fatalf("problem detail = %q", model.Detail)
	}
}
