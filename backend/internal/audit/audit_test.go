package audit

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/require"
)

// The published OpenAPI enum is derived from Actions, so an action constant
// that never reaches the slice is an endpoint returning a value its own
// contract rejects. That is how school_year_state_transition escaped: the enum
// was a hand-copied struct tag. Go cannot enumerate its own constants, so this
// test reads the source that declares them.
func TestActionsCoversEveryDeclaredActionConstant(t *testing.T) {
	declared, listed := parseActionVocabulary(t)

	require.NotEmpty(t, declared, "no Action constants were found; have they moved out of audit.go?")
	require.ElementsMatch(t, declared, listed,
		"the Action constants and the actions slice have diverged; add the new constant to the slice")
}

func TestActionsHasNoDuplicatesAndReturnsACopy(t *testing.T) {
	first := Actions()
	seen := map[Action]bool{}
	for _, action := range first {
		require.False(t, seen[action], "%s appears in the vocabulary twice", action)
		seen[action] = true
	}

	first[0] = "mutated"
	require.NotEqual(t, Action("mutated"), Actions()[0], "Actions exposes its backing array")
}

// parseActionVocabulary returns the names of every Action-typed constant in
// audit.go and the names referenced by its actions slice literal.
func parseActionVocabulary(t *testing.T) (declared, listed []string) {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "audit.go", nil, 0)
	require.NoError(t, err)

	for _, decl := range file.Decls {
		generic, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range generic.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			switch generic.Tok {
			case token.CONST:
				if named, ok := value.Type.(*ast.Ident); !ok || named.Name != "Action" {
					continue
				}
				for _, name := range value.Names {
					declared = append(declared, name.Name)
				}
			case token.VAR:
				if len(value.Names) != 1 || value.Names[0].Name != "actions" || len(value.Values) != 1 {
					continue
				}
				literal, ok := value.Values[0].(*ast.CompositeLit)
				require.True(t, ok, "the actions slice is no longer a composite literal")
				for _, element := range literal.Elts {
					name, ok := element.(*ast.Ident)
					require.True(t, ok, "the actions slice must reference constants, not string literals")
					listed = append(listed, name.Name)
				}
			}
		}
	}
	return declared, listed
}
