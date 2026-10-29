package transform

// Port of TSTransformer/nodes/statements/transformReturnStatement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformReturnStatementInner is TSTransformer/nodes/statements/transformReturnStatement.ts transformReturnStatementInner.
func transformReturnStatementInner(state *TransformState, returnExp *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformReturnStatement.ts", "transformReturnStatementInner"))
}

// transformReturnStatement is TSTransformer/nodes/statements/transformReturnStatement.ts transformReturnStatement.
func transformReturnStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformReturnStatement.ts", "transformReturnStatement"))
}
