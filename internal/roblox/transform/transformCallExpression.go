package transform

// Port of TSTransformer/nodes/expressions/transformCallExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformCallExpressionInner.
func transformCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformCallExpression.ts", "transformCallExpressionInner"))
}

// transformPropertyCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformPropertyCallExpressionInner.
func transformPropertyCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, name string, nodeArguments []*ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformCallExpression.ts", "transformPropertyCallExpressionInner"))
}

// transformElementCallExpressionInner is TSTransformer/nodes/expressions/transformCallExpression.ts transformElementCallExpressionInner.
func transformElementCallExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression *ast.Node, baseExpression luau.Expression, argumentExpression *ast.Node, nodeArguments []*ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformCallExpression.ts", "transformElementCallExpressionInner"))
}

// transformCallExpression is TSTransformer/nodes/expressions/transformCallExpression.ts transformCallExpression.
func transformCallExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformCallExpression.ts", "transformCallExpression"))
}
