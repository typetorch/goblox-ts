package transform

// Port of TSTransformer/nodes/expressions/transformAwaitExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformAwaitExpression is TSTransformer/nodes/expressions/transformAwaitExpression.ts transformAwaitExpression.
func transformAwaitExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	panic(todo("nodes/expressions/transformAwaitExpression.ts", "transformAwaitExpression"))
}
