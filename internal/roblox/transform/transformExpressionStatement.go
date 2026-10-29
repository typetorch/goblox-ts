package transform

// Port of TSTransformer/nodes/statements/transformExpressionStatement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformExpressionStatementInner is TSTransformer/nodes/statements/transformExpressionStatement.ts transformExpressionStatementInner.
func transformExpressionStatementInner(state *TransformState, expression *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformExpressionStatement.ts", "transformExpressionStatementInner"))
}

// transformExpressionStatement is TSTransformer/nodes/statements/transformExpressionStatement.ts transformExpressionStatement.
func transformExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformExpressionStatement.ts", "transformExpressionStatement"))
}
