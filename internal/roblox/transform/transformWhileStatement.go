package transform

// Port of TSTransformer/nodes/statements/transformWhileStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformWhileStatement is TSTransformer/nodes/statements/transformWhileStatement.ts transformWhileStatement.
func transformWhileStatement(state *TransformState, node *ast.Node) *luau.List {
	n := node.AsWhileStatement()
	whileStatements := luau.MakeList()

	conditionPrereqs := NewPrereqs()
	conditionExp := createTruthinessChecks(
		state,
		conditionPrereqs,
		transformExpression(state, conditionPrereqs, n.Expression),
		n.Expression,
	)

	if !conditionPrereqs.Statements.IsEmpty() {
		whileStatements.PushList(conditionPrereqs.Statements)
		whileStatements.Push(luau.Create(&luau.IfStatement{
			Condition:  luau.Unary("not", conditionExp),
			Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
			ElseBody:   luau.MakeList(),
		}))
		conditionExp = luau.Bool(true)
	}

	whileStatements.PushList(transformStatementList(state, n.Statement, getStatements(n.Statement), nil))

	return luau.MakeList(
		luau.Create(&luau.WhileStatement{
			Condition:  conditionExp,
			Statements: whileStatements,
		}),
	)
}
