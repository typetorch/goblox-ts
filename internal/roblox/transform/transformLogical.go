package transform

// Port of TSTransformer/nodes/transformLogical.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformLogical is TSTransformer/nodes/transformLogical.ts transformLogical.
func transformLogical(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/transformLogical.ts", "transformLogical"))
}
