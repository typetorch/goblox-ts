package transform

// Port of TSTransformer/nodes/transformMethodDeclaration.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformMethodDeclaration is TSTransformer/nodes/transformMethodDeclaration.ts transformMethodDeclaration.
func transformMethodDeclaration(state *TransformState, prereqs *Prereqs, node *ast.Node, ptr *Pointer) *luau.List {
	panic(todo("nodes/transformMethodDeclaration.ts", "transformMethodDeclaration"))
}
