package transform

// Port of TSTransformer/nodes/statements/transformIfStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformIfStatementInner is TSTransformer/nodes/statements/transformIfStatement.ts transformIfStatementInner.
func transformIfStatementInner(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.IfStatement {
	ifStatement := node.AsIfStatement()
	condition := createTruthinessChecks(
		state,
		prereqs,
		transformExpression(state, prereqs, ifStatement.Expression),
		ifStatement.Expression,
	)

	statements := transformStatementList(state, ifStatement.ThenStatement, getStatements(ifStatement.ThenStatement), nil)

	elseStatement := ifStatement.ElseStatement

	var elseBody luau.ListOrNode
	if elseStatement == nil {
		elseBody = luau.MakeList()
	} else if ast.IsIfStatement(elseStatement) {
		elseIfPrereqs := NewPrereqs()
		elseIf := transformIfStatementInner(state, elseIfPrereqs, elseStatement)
		if elseIfPrereqs.Statements.IsEmpty() {
			elseBody = elseIf
		} else {
			elseIfStatements := luau.MakeList()
			elseIfStatements.PushList(elseIfPrereqs.Statements)
			elseIfStatements.Push(elseIf)
			elseBody = elseIfStatements
		}
	} else {
		elseBody = transformStatementList(state, elseStatement, getStatements(elseStatement), nil)
	}

	return luau.Create(&luau.IfStatement{
		Condition:  condition,
		Statements: statements,
		ElseBody:   elseBody,
	})
}

// transformIfStatement is TSTransformer/nodes/statements/transformIfStatement.ts transformIfStatement.
func transformIfStatement(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()
	prereqs := NewPrereqs()
	statement := transformIfStatementInner(state, prereqs, node)
	statements.PushList(prereqs.Statements)
	statements.Push(statement)
	return statements
}
