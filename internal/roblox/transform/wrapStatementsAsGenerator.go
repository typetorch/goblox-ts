package transform

// Port of TSTransformer/util/wrapStatementsAsGenerator.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// wrapStatementsAsGenerator is TSTransformer/util/wrapStatementsAsGenerator.ts wrapStatementsAsGenerator.
func wrapStatementsAsGenerator(state *TransformState, node *ast.Node, statements *luau.List) *luau.List {
	return luau.MakeList(
		luau.Create(&luau.ReturnStatement{
			Expression: luau.Call(state.TS(node, "generator"),
				luau.Create(&luau.FunctionExpression{
					HasDotDotDot: false,
					Parameters:   luau.MakeList(),
					Statements:   statements,
				}),
			),
		}),
	)
}
