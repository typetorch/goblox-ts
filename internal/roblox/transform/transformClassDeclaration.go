package transform

// Port of TSTransformer/nodes/statements/transformClassDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformClassDeclaration is TSTransformer/nodes/statements/transformClassDeclaration.ts transformClassDeclaration.
func transformClassDeclaration(state *TransformState, node *ast.Node) *luau.List {
	statements, _ := transformClassLikeDeclaration(state, node)
	return statements
}
