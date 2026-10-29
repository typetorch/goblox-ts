package transform

// Port of TSTransformer/nodes/statements/transformBlock.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformBlock is TSTransformer/nodes/statements/transformBlock.ts transformBlock.
func transformBlock(state *TransformState, node *ast.Node) *luau.List {
	return luau.MakeList(
		luau.Create(&luau.DoStatement{
			Statements: transformStatementList(state, node, node.AsBlock().Statements.Nodes, nil),
		}),
	)
}
