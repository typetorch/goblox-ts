package transform

// Port of TSTransformer/nodes/expressions/transformUnaryExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPostfixUnaryExpression is TSTransformer/nodes/expressions/transformUnaryExpression.ts transformPostfixUnaryExpression.
func transformPostfixUnaryExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.TemporaryIdentifier {
	panic(todo("nodes/expressions/transformUnaryExpression.ts", "transformPostfixUnaryExpression"))
}

// transformPrefixUnaryExpression is TSTransformer/nodes/expressions/transformUnaryExpression.ts transformPrefixUnaryExpression.
func transformPrefixUnaryExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	panic(todo("nodes/expressions/transformUnaryExpression.ts", "transformPrefixUnaryExpression"))
}
