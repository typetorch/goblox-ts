package transform

// Port of TSTransformer/nodes/expressions/transformNewExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformNewExpression is TSTransformer/nodes/expressions/transformNewExpression.ts transformNewExpression.
func transformNewExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	validateNotAnyType(state, node.Expression())

	symbol := getFirstConstructSymbol(state, node.Expression())
	if symbol != nil {
		macro := state.Services.MacroManager.GetConstructorMacro(symbol)
		if macro != nil {
			return macro(state, prereqs, node)
		}
	}

	expression := convertToIndexableExpression(transformExpression(state, prereqs, node.Expression()))
	var args []luau.Expression
	if node.AsNewExpression().Arguments != nil {
		args = ensureTransformOrder(state, prereqs, node.Arguments(), nil)
	}
	return luau.Call(luau.Property(expression, "new"), args...)
}
