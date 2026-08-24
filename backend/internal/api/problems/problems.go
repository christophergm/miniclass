// Package problems defines the stable RFC 9457 problem types exposed by the
// MiniClass API.
package problems

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/danielgtaylor/huma/v2"
)

// Slug is the stable machine-readable identifier for a problem type. Slugs
// are used as relative URI references in the RFC 9457 type member.
type Slug string

const (
	RouteNotFound       Slug = "route-not-found"
	MethodNotAllowed    Slug = "method-not-allowed"
	InternalError       Slug = "internal-error"
	DatabaseUnavailable Slug = "database-unavailable"
)

// Definition describes one registered problem type.
type Definition struct {
	Slug  Slug
	Title string
}

var registry = map[Slug]Definition{
	RouteNotFound:       {Slug: RouteNotFound, Title: "Route not found"},
	MethodNotAllowed:    {Slug: MethodNotAllowed, Title: "Method not allowed"},
	InternalError:       {Slug: InternalError, Title: "Internal server error"},
	DatabaseUnavailable: {Slug: DatabaseUnavailable, Title: "Database unavailable"},
}

// Definitions returns the registry in stable slug order for contract
// generation and callers that need to enumerate the closed set.
func Definitions() []Definition {
	definitions := make([]Definition, 0, len(registry))
	for _, definition := range registry {
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Slug < definitions[j].Slug
	})
	return definitions
}

// New constructs a registered RFC 9457 problem response.
func New(status int, slug Slug, detail string) *huma.ErrorModel {
	definition, ok := registry[slug]
	if !ok {
		definition = Definition{Slug: slug, Title: http.StatusText(status)}
	}
	return &huma.ErrorModel{
		Type:   string(definition.Slug),
		Title:  definition.Title,
		Status: status,
		Detail: detail,
	}
}

// Write writes a problem response for middleware and router-level failures
// that occur outside a Huma operation.
func Write(w http.ResponseWriter, problem *huma.ErrorModel) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(problem.GetStatus())
	_ = json.NewEncoder(w).Encode(problem)
}

// Schema returns the OpenAPI schema for the registered problem-type slugs.
func Schema() *huma.Schema {
	enum := make([]any, 0, len(registry))
	for _, definition := range Definitions() {
		enum = append(enum, string(definition.Slug))
	}
	return &huma.Schema{
		Type:        "string",
		Description: "Stable RFC 9457 problem type slug.",
		Enum:        enum,
	}
}
