package transform

// Port of TSTransformer/util/assertNever.ts.

import (
	"fmt"

	"github.com/microsoft/typescript-go/internal/ast"
)

// assertNever panics: value was not handled by an exhaustive switch.
func assertNever(value any, message string) {
	description := fmt.Sprintf("%v", value)
	if node, ok := value.(*ast.Node); ok && node != nil {
		description = "a TS node of kind " + getKindName(node.Kind)
	}
	panic("Exhaustive assertion failed! " + message + ", value was " + description +
		"\nThis is usually caused by a TypeScript version mismatch.")
}
