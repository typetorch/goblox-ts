package transform

// Port of TSTransformer/nodes/expressions/transformSpreadElement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformSpreadElement is TSTransformer/nodes/expressions/transformSpreadElement.ts transformSpreadElement.
func transformSpreadElement(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	panic(todo("nodes/expressions/transformSpreadElement.ts", "transformSpreadElement"))
}
