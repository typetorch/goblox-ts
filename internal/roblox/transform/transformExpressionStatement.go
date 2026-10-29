package transform

// Port of TSTransformer/nodes/statements/transformExpressionStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformUnaryExpressionStatement(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.Assignment {
	var operand *ast.Node
	var operatorKind ast.Kind
	if ast.IsPrefixUnaryExpression(node) {
		operand = node.AsPrefixUnaryExpression().Operand
		operatorKind = node.AsPrefixUnaryExpression().Operator
	} else {
		operand = node.AsPostfixUnaryExpression().Operand
		operatorKind = node.AsPostfixUnaryExpression().Operator
	}
	writable := transformWritableExpression(state, prereqs, operand, false)
	var operator luau.AssignmentOperator = "-="
	if operatorKind == ast.KindPlusPlusToken {
		operator = "+="
	}
	return luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: operator,
		Right:    luau.Number(1),
	})
}

// transformExpressionStatementInner is TSTransformer/nodes/statements/transformExpressionStatement.ts transformExpressionStatementInner.
func transformExpressionStatementInner(state *TransformState, expression *ast.Node) *luau.List {
	statements := luau.MakeList()
	prereqs := NewPrereqs()

	if ast.IsBinaryExpression(expression) {
		binary := expression.AsBinaryExpression()
		operatorKind := binary.OperatorToken.Kind
		if ast.IsLogicalOrCoalescingAssignmentExpression(expression) {
			return transformLogicalOrCoalescingAssignmentExpressionStatement(state, expression)
		} else if ast.IsAssignmentOperator(operatorKind) &&
			!ast.IsArrayLiteralExpression(binary.Left) &&
			!ast.IsObjectLiteralExpression(binary.Left) {
			writableType := state.GetType(binary.Left)
			valueType := state.GetType(binary.Right)
			operator := getSimpleAssignmentOperator(writableType, operatorKind, valueType)
			writable, readable, value := transformWritableAssignment(
				state,
				prereqs,
				binary.Left,
				binary.Right,
				operator == "",
				operator != "=",
			)

			var assignment *luau.Assignment
			if operator != "" {
				assignment = createAssignmentStatement(
					writable,
					operator,
					getAssignableValue(operator, value, valueType),
					readable,
				)
			} else {
				assignment = createCompoundAssignmentStatement(
					prereqs,
					writable,
					writableType,
					readable,
					operatorKind,
					value,
					valueType,
				)
			}

			statements.PushList(prereqs.Statements)
			statements.Push(assignment)
			return statements
		}
	} else if ast.IsPrefixUnaryExpression(expression) && isUnaryAssignmentOperator(expression.AsPrefixUnaryExpression().Operator) ||
		ast.IsPostfixUnaryExpression(expression) && isUnaryAssignmentOperator(expression.AsPostfixUnaryExpression().Operator) {
		assignment := transformUnaryExpressionStatement(state, prereqs, expression)
		statements.PushList(prereqs.Statements)
		statements.Push(assignment)
		return statements
	}

	expressionStatements := wrapExpressionStatement(transformExpression(state, prereqs, expression))
	statements.PushList(prereqs.Statements)
	statements.PushList(expressionStatements)
	return statements
}

// transformExpressionStatement is TSTransformer/nodes/statements/transformExpressionStatement.ts transformExpressionStatement.
func transformExpressionStatement(state *TransformState, node *ast.Node) *luau.List {
	expression := skipDownwards(node.AsExpressionStatement().Expression)
	return transformExpressionStatementInner(state, expression)
}
