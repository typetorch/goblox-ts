package transform

// Port of TSTransformer/nodes/class/transformDecorators.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformDecorators is TSTransformer/nodes/class/transformDecorators.ts transformDecorators.
func transformDecorators(state *TransformState, node *ast.Node, classId luau.AnyIdentifier) *luau.List {
	panic(todo("nodes/class/transformDecorators.ts", "transformDecorators"))
}
