package transform

// Port of TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyAccessExpressionInner is TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts transformPropertyAccessExpressionInner.
func transformPropertyAccessExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, name string) luau.Expression {
	// a in a.b
	validateNotAnyType(state, node.Expression())

	memberType := state.TypeChecker.GetNonOptionalType(state.GetType(node))
	receiverType := state.GetType(node.Expression())
	addIndexDiagnostics(state, node, memberType, receiverType)

	if ast.IsDeleteExpression(skipUpwards(node).Parent) {
		prereqs.Push(luau.Create(&luau.Assignment{
			Left:     luau.Property(convertToIndexableExpression(expression), name),
			Operator: "=",
			Right:    luau.Nil(),
		}))
		return luau.NewNone()
	}

	property := luau.Property(convertToIndexableExpression(expression), name)
	tryMarkBuiltinMember(state, node, property)
	return property
}

// transformPropertyAccessExpression is TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts transformPropertyAccessExpression.
func transformPropertyAccessExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	if constantValue := getConstantValueLiteral(state, node); constantValue != nil {
		return constantValue
	}

	return transformOptionalChain(state, prereqs, node)
}
