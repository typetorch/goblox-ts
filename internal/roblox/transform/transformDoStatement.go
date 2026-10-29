package transform

// Port of TSTransformer/nodes/statements/transformDoStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformDoStatement is TSTransformer/nodes/statements/transformDoStatement.ts transformDoStatement.
func transformDoStatement(state *TransformState, node *ast.Node) *luau.List {
	n := node.AsDoStatement()
	expression, statement := n.Expression, n.Statement
	statements := transformStatementList(state, statement, getStatements(statement), nil)

	conditionIsInvertedInLuau := true
	if ast.IsPrefixUnaryExpression(expression) && expression.AsPrefixUnaryExpression().Operator == ast.KindExclamationToken {
		expression = expression.AsPrefixUnaryExpression().Operand
		conditionIsInvertedInLuau = false
	}

	conditionPrereqs := NewPrereqs()
	condition := createTruthinessChecks(
		state,
		conditionPrereqs,
		transformExpression(state, conditionPrereqs, expression),
		expression,
	)

	repeatStatements := luau.MakeList()
	repeatStatements.Push(luau.Create(&luau.DoStatement{
		Statements: statements,
	}))
	repeatStatements.PushList(conditionPrereqs.Statements)

	var finalCondition luau.Expression = condition
	if conditionIsInvertedInLuau {
		finalCondition = luau.Unary("not", condition)
	}
	return luau.MakeList(
		luau.Create(&luau.RepeatStatement{
			Statements: repeatStatements,
			Condition:  finalCondition,
		}),
	)
}
