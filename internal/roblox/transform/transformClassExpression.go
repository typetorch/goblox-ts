package transform

// Port of TSTransformer/nodes/expressions/transformClassExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformClassExpression is TSTransformer/nodes/expressions/transformClassExpression.ts transformClassExpression.
func transformClassExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	statements, name := transformClassLikeDeclaration(state, node)
	prereqs.PushList(statements)
	return name
}
