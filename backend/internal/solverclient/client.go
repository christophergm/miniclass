// Package solverclient calls the stateless solver sidecar over its versioned contract.
package solverclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chrismott/miniclass/internal/solvercontract"
)

var (
	ErrUnavailable     = errors.New("solver sidecar is unavailable")
	ErrInvalidResponse = errors.New("solver sidecar returned an invalid response")
)

// Client is the boundary used by application services. It deliberately accepts
// only the self-contained solver contract and has no database dependency.
type Client interface {
	Solve(context.Context, solvercontract.Request) (solvercontract.Response, error)
}

type HTTPClient struct {
	baseURL string
	client  *http.Client
}

func New(baseURL string, timeout time.Duration) (*HTTPClient, error) {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("solver base URL must be an absolute URL")
	}
	if timeout <= 0 {
		return nil, errors.New("solver request timeout must be positive")
	}
	return &HTTPClient{baseURL: strings.TrimRight(parsed.String(), "/"), client: &http.Client{Timeout: timeout}}, nil
}

func (c *HTTPClient) Solve(ctx context.Context, request solvercontract.Request) (solvercontract.Response, error) {
	if c == nil || c.client == nil {
		return solvercontract.Response{}, fmt.Errorf("%w: client is nil", ErrUnavailable)
	}
	payload, err := solvercontract.CanonicalJSON(request)
	if err != nil {
		return solvercontract.Response{}, fmt.Errorf("encode solver request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/solve", bytes.NewReader(payload))
	if err != nil {
		return solvercontract.Response{}, fmt.Errorf("build solver request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.client.Do(httpRequest)
	if err != nil {
		return solvercontract.Response{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return solvercontract.Response{}, fmt.Errorf("read solver response: %w", err)
	}
	if response.StatusCode >= http.StatusInternalServerError {
		return solvercontract.Response{}, fmt.Errorf("%w: status %d", ErrUnavailable, response.StatusCode)
	}
	if response.StatusCode != http.StatusOK {
		return solvercontract.Response{}, fmt.Errorf("%w: status %d", ErrInvalidResponse, response.StatusCode)
	}
	var result solvercontract.Response
	if err := json.Unmarshal(body, &result); err != nil {
		return solvercontract.Response{}, fmt.Errorf("%w: decode JSON: %v", ErrInvalidResponse, err)
	}
	if result.Seed != request.Seed {
		return solvercontract.Response{}, fmt.Errorf("%w: response seed does not match request", ErrInvalidResponse)
	}
	if _, err := solvercontract.CanonicalResponseJSON(result); err != nil {
		return solvercontract.Response{}, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	return result, nil
}
