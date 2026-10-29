package transform

// Port of TSTransformer/nodes/expressions/transformDeleteExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformDeleteExpression is TSTransformer/nodes/expressions/transformDeleteExpression.ts transformDeleteExpression.
func transformDeleteExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	// we just want the prereqs, deleting is done in the index expression transforms
	transformExpression(state, prereqs, node.Expression())
	if !isUsedAsStatement(node) {
		return luau.Bool(true)
	}
	return luau.NewNone()
}
