package transform

// Port of TSTransformer/nodes/statements/transformIfStatement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformIfStatementInner is TSTransformer/nodes/statements/transformIfStatement.ts transformIfStatementInner.
func transformIfStatementInner(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.IfStatement {
	panic(todo("nodes/statements/transformIfStatement.ts", "transformIfStatementInner"))
}

// transformIfStatement is TSTransformer/nodes/statements/transformIfStatement.ts transformIfStatement.
func transformIfStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformIfStatement.ts", "transformIfStatement"))
}
