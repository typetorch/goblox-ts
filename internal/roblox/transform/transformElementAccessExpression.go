package transform

// Port of TSTransformer/nodes/expressions/transformElementAccessExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformElementAccessExpressionInner is TSTransformer/nodes/expressions/transformElementAccessExpression.ts transformElementAccessExpressionInner.
func transformElementAccessExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, argumentExpression *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformElementAccessExpression.ts", "transformElementAccessExpressionInner"))
}

// transformElementAccessExpression is TSTransformer/nodes/expressions/transformElementAccessExpression.ts transformElementAccessExpression.
func transformElementAccessExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformElementAccessExpression.ts", "transformElementAccessExpression"))
}
