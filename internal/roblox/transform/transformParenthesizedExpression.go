package transform

// Port of TSTransformer/nodes/expressions/transformParenthesizedExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformParenthesizedExpression is TSTransformer/nodes/expressions/transformParenthesizedExpression.ts transformParenthesizedExpression.
func transformParenthesizedExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformParenthesizedExpression.ts", "transformParenthesizedExpression"))
}
