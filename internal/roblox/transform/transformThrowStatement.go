package transform

// Port of TSTransformer/nodes/statements/transformThrowStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformThrowStatement is TSTransformer/nodes/statements/transformThrowStatement.ts transformThrowStatement.
func transformThrowStatement(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()
	prereqs := NewPrereqs()
	args := []luau.Expression{transformExpression(state, prereqs, node.AsThrowStatement().Expression)}

	statements.PushList(prereqs.Statements)
	statements.Push(luau.Create(&luau.CallStatement{
		Expression: luau.Call(luau.Globals.Error, args...),
	}))
	return statements
}
