package transform

// Port of TSTransformer/nodes/transformInitializer.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformInitializer is TSTransformer/nodes/transformInitializer.ts transformInitializer.
func transformInitializer(state *TransformState, id luau.WritableExpression, initializer *ast.Node) *luau.IfStatement {
	panic(todo("nodes/transformInitializer.ts", "transformInitializer"))
}
