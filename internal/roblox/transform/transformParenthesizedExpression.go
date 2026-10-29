package transform

// Port of TSTransformer/nodes/expressions/transformParenthesizedExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformParenthesizedExpression is TSTransformer/nodes/expressions/transformParenthesizedExpression.ts transformParenthesizedExpression.
func transformParenthesizedExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	expression := transformExpression(state, prereqs, skipDownwards(node.Expression()))
	if luau.IsSimple(expression) {
		return expression
	} else {
		return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
	}
}
