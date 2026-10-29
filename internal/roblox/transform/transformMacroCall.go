package transform

// Port of TSTransformer/nodes/transformMacroCall.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformMacroCall is TSTransformer/nodes/transformMacroCall.ts transformMacroCall.
func transformMacroCall(macro CallMacro, state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, nodeArguments []*ast.Node) luau.Expression {
	panic(todo("nodes/transformMacroCall.ts", "transformMacroCall"))
}
