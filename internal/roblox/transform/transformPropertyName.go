package transform

// Port of TSTransformer/nodes/transformPropertyName.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyName is TSTransformer/nodes/transformPropertyName.ts transformPropertyName.
func transformPropertyName(state *TransformState, prereqs *Prereqs, name *ast.Node) luau.Expression {
	panic(todo("nodes/transformPropertyName.ts", "transformPropertyName"))
}
