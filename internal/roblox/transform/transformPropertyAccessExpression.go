package transform

// Port of TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyAccessExpressionInner is TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts transformPropertyAccessExpressionInner.
func transformPropertyAccessExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, name string) luau.Expression {
	panic(todo("nodes/expressions/transformPropertyAccessExpression.ts", "transformPropertyAccessExpressionInner"))
}

// transformPropertyAccessExpression is TSTransformer/nodes/expressions/transformPropertyAccessExpression.ts transformPropertyAccessExpression.
func transformPropertyAccessExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformPropertyAccessExpression.ts", "transformPropertyAccessExpression"))
}
