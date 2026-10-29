package transform

// Port of TSTransformer/nodes/binding/transformBindingName.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformBindingName is TSTransformer/nodes/binding/transformBindingName.ts transformBindingName.
func transformBindingName(state *TransformState, name *ast.Node, initializers *luau.List) luau.AnyIdentifier {
	panic(todo("nodes/binding/transformBindingName.ts", "transformBindingName"))
}
