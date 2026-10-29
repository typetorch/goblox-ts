package transform

// Port of TSTransformer/util/wrapReturnIfLuaTuple.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// wrapReturnIfLuaTuple is TSTransformer/util/wrapReturnIfLuaTuple.ts wrapReturnIfLuaTuple.
func wrapReturnIfLuaTuple(state *TransformState, node *ast.Node, exp luau.Expression) luau.Expression {
	panic(todo("util/wrapReturnIfLuaTuple.ts", "wrapReturnIfLuaTuple"))
}
