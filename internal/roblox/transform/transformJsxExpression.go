package transform

// Port of TSTransformer/nodes/expressions/transformJsxExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxExpression is TSTransformer/nodes/expressions/transformJsxExpression.ts transformJsxExpression.
func transformJsxExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	jsxExpression := node.AsJsxExpression()
	if jsxExpression.Expression != nil {
		expression := transformExpression(state, prereqs, jsxExpression.Expression)
		if jsxExpression.DotDotDotToken != nil {
			return luau.Call(luau.Globals.Unpack, expression)
		}
		return expression
	}
	return luau.NewNone()
}
