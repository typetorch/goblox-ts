package transform

// Port of TSTransformer/nodes/class/transformPropertyDeclaration.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformPropertyDeclaration is TSTransformer/nodes/class/transformPropertyDeclaration.ts transformPropertyDeclaration.
func transformPropertyDeclaration(state *TransformState, prereqs *Prereqs, node *ast.Node, name luau.AnyIdentifier) *luau.List {
	panic(todo("nodes/class/transformPropertyDeclaration.ts", "transformPropertyDeclaration"))
}
