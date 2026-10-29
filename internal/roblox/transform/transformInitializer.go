package transform

// Port of TSTransformer/nodes/transformInitializer.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformInitializer is TSTransformer/nodes/transformInitializer.ts transformInitializer.
func transformInitializer(state *TransformState, id luau.WritableExpression, initializer *ast.Node) *luau.IfStatement {
	statements := luau.MakeList()
	initializerPrereqs := NewPrereqs()
	value := transformExpression(state, initializerPrereqs, initializer)
	statements.PushList(initializerPrereqs.Statements)
	statements.Push(luau.Create(&luau.Assignment{
		Left:     id,
		Operator: "=",
		Right:    value,
	}))

	return luau.Create(&luau.IfStatement{
		Condition:  luau.Binary(id, "==", luau.Nil()),
		ElseBody:   luau.MakeList(),
		Statements: statements,
	})
}
