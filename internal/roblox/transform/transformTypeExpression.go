package transform

// Port of TSTransformer/nodes/expressions/transformTypeExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTypeExpression is TSTransformer/nodes/expressions/transformTypeExpression.ts transformTypeExpression.
func transformTypeExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	return transformExpression(state, prereqs, node.Expression())
}
