package transform

// Port of TSTransformer/nodes/statements/transformForOfStatement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformForOfRangeMacro is TSTransformer/nodes/statements/transformForOfStatement.ts transformForOfRangeMacro.
func transformForOfRangeMacro(state *TransformState, node *ast.Node, macroCall *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformForOfStatement.ts", "transformForOfRangeMacro"))
}

// transformForOfStatement is TSTransformer/nodes/statements/transformForOfStatement.ts transformForOfStatement.
func transformForOfStatement(state *TransformState, node *ast.Node) *luau.List {
	panic(todo("nodes/statements/transformForOfStatement.ts", "transformForOfStatement"))
}
