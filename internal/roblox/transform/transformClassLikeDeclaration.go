package transform

// Port of TSTransformer/nodes/class/transformClassLikeDeclaration.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformClassLikeDeclaration is TSTransformer/nodes/class/transformClassLikeDeclaration.ts transformClassLikeDeclaration.
func transformClassLikeDeclaration(state *TransformState, node *ast.Node) (statements *luau.List, name luau.AnyIdentifier) {
	panic(todo("nodes/class/transformClassLikeDeclaration.ts", "transformClassLikeDeclaration"))
}
