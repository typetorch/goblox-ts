package transform

// Port of TSTransformer/nodes/statements/transformVariableStatement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformVariable is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariable.
func transformVariable(state *TransformState, prereqs *Prereqs, identifier *ast.Node, right luau.Expression) luau.Expression {
	panic(todo("nodes/statements/transformVariableStatement.ts", "transformVariable"))
}

// transformVariableDeclaration is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableDeclaration.
func transformVariableDeclaration(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformVariableStatement.ts", "transformVariableDeclaration"))
}

// isVarDeclaration is TSTransformer/nodes/statements/transformVariableStatement.ts isVarDeclaration.
func isVarDeclaration(node *ast.Node) bool {
	panic(todo("nodes/statements/transformVariableStatement.ts", "isVarDeclaration"))
}

// transformVariableDeclarationList is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableDeclarationList.
func transformVariableDeclarationList(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformVariableStatement.ts", "transformVariableDeclarationList"))
}

// transformVariableStatement is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableStatement.
func transformVariableStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformVariableStatement.ts", "transformVariableStatement"))
}
