package problems

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/require"
)

func TestWithCausePreservesProblemAndErrorChain(t *testing.T) {
	cause := errors.New("internal diagnostic")
	model := New(http.StatusInternalServerError, InternalError, "generic public message")
	problem := WithCause(model, cause)
	wrapped := fmt.Errorf("handler: %w", problem)
	require.ErrorIs(t, wrapped, cause)
	require.Same(t, cause, Cause(wrapped))
	var found *huma.ErrorModel
	require.ErrorAs(t, wrapped, &found)
	require.Same(t, model, found)
	var status huma.StatusError
	require.ErrorAs(t, wrapped, &status)
	require.Equal(t, http.StatusInternalServerError, status.GetStatus())
	public, err := json.Marshal(problem)
	require.NoError(t, err)
	expected, err := json.Marshal(model)
	require.NoError(t, err)
	require.JSONEq(t, string(expected), string(public))
	require.NotContains(t, string(public), cause.Error())
	require.Nil(t, Cause(model))
}
