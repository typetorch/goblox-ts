package transform

// Port of TSTransformer/nodes/expressions/transformVoidExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformVoidExpression is TSTransformer/nodes/expressions/transformVoidExpression.ts transformVoidExpression.
func transformVoidExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.NilLiteral {
	prereqs.PushList(transformExpressionStatementInner(state, skipDownwards(node.Expression())))
	return luau.Create(&luau.NilLiteral{})
}
