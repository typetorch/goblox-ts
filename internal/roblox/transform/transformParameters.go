package transform

// Port of TSTransformer/nodes/transformParameters.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformParameters is TSTransformer/nodes/transformParameters.ts transformParameters.
func transformParameters(state *TransformState, node *ast.Node) (parameters *luau.List, statements *luau.List, hasDotDotDot bool) {
	panic(todo("nodes/transformParameters.ts", "transformParameters"))
}
