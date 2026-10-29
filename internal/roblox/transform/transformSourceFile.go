package transform

// Port of TSTransformer/nodes/transformSourceFile.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformSourceFile is TSTransformer/nodes/transformSourceFile.ts transformSourceFile.
func transformSourceFile(state *TransformState, node *ast.SourceFile) *luau.List {
	panic(todo("nodes/transformSourceFile.ts", "transformSourceFile"))
}
