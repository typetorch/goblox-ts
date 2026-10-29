package transform

// Port of TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformCoalescingAssignmentExpression(
	state *TransformState,
	prereqs *Prereqs,
	left *ast.Node,
	right *ast.Node,
) luau.WritableExpression {
	writable := transformWritableExpression(state, prereqs, left, true)
	valuePrereqs := NewPrereqs()
	value := transformExpression(state, valuePrereqs, right)

	ifStatements := luau.MakeList()
	ifStatements.PushList(valuePrereqs.Statements)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: "=",
		Right:    value,
	}))

	prereqs.Push(luau.Create(&luau.IfStatement{
		Condition:  luau.Binary(writable, "==", luau.Nil()),
		Statements: ifStatements,
		ElseBody:   luau.MakeList(),
	}))

	return writable
}

func transformLogicalAndAssignmentExpression(
	state *TransformState,
	prereqs *Prereqs,
	left *ast.Node,
	right *ast.Node,
) luau.WritableExpression {
	writable := transformWritableExpression(state, prereqs, left, true)
	valuePrereqs := NewPrereqs()
	value := transformExpression(state, valuePrereqs, right)

	conditionId := prereqs.PushToVar(writable, "condition")

	ifStatements := luau.MakeList()
	ifStatements.PushList(valuePrereqs.Statements)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     conditionId,
		Operator: "=",
		Right:    value,
	}))

	prereqs.Push(luau.Create(&luau.IfStatement{
		Condition:  createTruthinessChecks(state, prereqs, writable, left),
		Statements: ifStatements,
		ElseBody:   luau.MakeList(),
	}))

	prereqs.Push(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: "=",
		Right:    conditionId,
	}))

	return writable
}

func transformLogicalOrAssignmentExpression(
	state *TransformState,
	prereqs *Prereqs,
	left *ast.Node,
	right *ast.Node,
) luau.WritableExpression {
	writable := transformWritableExpression(state, prereqs, left, true)
	valuePrereqs := NewPrereqs()
	value := transformExpression(state, valuePrereqs, right)

	conditionId := prereqs.PushToVar(writable, "condition")

	ifStatements := luau.MakeList()
	ifStatements.PushList(valuePrereqs.Statements)
	ifStatements.Push(luau.Create(&luau.Assignment{
		Left:     conditionId,
		Operator: "=",
		Right:    value,
	}))

	prereqs.Push(luau.Create(&luau.IfStatement{
		Condition:  luau.Unary("not", createTruthinessChecks(state, prereqs, writable, left)),
		Statements: ifStatements,
		ElseBody:   luau.MakeList(),
	}))

	prereqs.Push(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: "=",
		Right:    conditionId,
	}))

	return writable
}

// transformLogicalOrCoalescingAssignmentExpression is TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts transformLogicalOrCoalescingAssignmentExpression.
func transformLogicalOrCoalescingAssignmentExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.WritableExpression {
	binary := node.AsBinaryExpression()
	operator := binary.OperatorToken.Kind
	if operator == ast.KindQuestionQuestionEqualsToken {
		return transformCoalescingAssignmentExpression(state, prereqs, binary.Left, binary.Right)
	} else if operator == ast.KindAmpersandAmpersandEqualsToken {
		return transformLogicalAndAssignmentExpression(state, prereqs, binary.Left, binary.Right)
	} else {
		return transformLogicalOrAssignmentExpression(state, prereqs, binary.Left, binary.Right)
	}
}

// transformLogicalOrCoalescingAssignmentExpressionStatement is TSTransformer/nodes/transformLogicalOrCoalescingAssignmentExpression.ts transformLogicalOrCoalescingAssignmentExpressionStatement.
func transformLogicalOrCoalescingAssignmentExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	prereqs := NewPrereqs()
	transformLogicalOrCoalescingAssignmentExpression(state, prereqs, node)
	return prereqs.Statements
}
