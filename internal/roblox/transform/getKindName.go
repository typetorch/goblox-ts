package transform

// Port of TSTransformer/util/getKindName.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

// getKindName is the TypeScript name of a syntax kind. typescript-go's marker kinds (FirstX, LastX)
// are aliases, so the stringer already returns the real kind's name.
func getKindName(kind ast.Kind) string {
	return strings.TrimPrefix(kind.String(), "Kind")
}
