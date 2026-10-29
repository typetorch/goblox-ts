package transform

// Port of TSTransformer/util/expressionMightMutate.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// expressionMightMutate is TSTransformer/util/expressionMightMutate.ts expressionMightMutate.
func expressionMightMutate(state *TransformState, expression luau.Expression, node *ast.Node) bool {
	panic(todo("util/expressionMightMutate.ts", "expressionMightMutate"))
}
