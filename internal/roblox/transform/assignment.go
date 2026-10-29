package transform

// Port of TSTransformer/util/assignment.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

var compoundOperatorMap = map[ast.Kind]luau.AssignmentOperator{
	// compound assignment
	ast.KindMinusEqualsToken:            "-=",
	ast.KindAsteriskEqualsToken:         "*=",
	ast.KindSlashEqualsToken:            "/=",
	ast.KindAsteriskAsteriskEqualsToken: "^=",
	ast.KindPercentEqualsToken:          "%=",

	// unary
	ast.KindPlusPlusToken:   "+=",
	ast.KindMinusMinusToken: "-=",

	// normal assignment
	ast.KindEqualsToken: "=",
}

// getSimpleAssignmentOperator is TSTransformer/util/assignment.ts getSimpleAssignmentOperator. It returns
// "" where TypeScript returns undefined.
func getSimpleAssignmentOperator(leftType *checker.Type, operatorKind ast.Kind, rightType *checker.Type) luau.AssignmentOperator {
	// plus
	if operatorKind == ast.KindPlusEqualsToken {
		if isDefinitelyType(leftType, isStringType) || isDefinitelyType(rightType, isStringType) {
			return "..="
		}
		return "+="
	}

	return compoundOperatorMap[operatorKind]
}

// createAssignmentStatement is TSTransformer/util/assignment.ts createAssignmentStatement.
func createAssignmentStatement(writable luau.WritableExpression, operator luau.AssignmentOperator, value luau.Expression, readable luau.WritableExpression) *luau.Assignment {
	operatorField := "="
	right := value
	if readable == writable {
		operatorField = operator
	} else {
		right = luau.Binary(readable, operator[:len(operator)-1], value)
	}
	return luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: operatorField,
		Right:    right,
	})
}

// createAssignmentExpression is TSTransformer/util/assignment.ts createAssignmentExpression. A nil
// readable is the TypeScript default (writable).
func createAssignmentExpression(prereqs *Prereqs, writable luau.WritableExpression, operator luau.AssignmentOperator, value luau.Expression, readable luau.WritableExpression) luau.WritableExpression {
	if readable == nil {
		readable = writable
	}
	prereqs.Push(createAssignmentStatement(writable, operator, value, readable))
	return writable
}

// createCompoundAssignmentStatement is TSTransformer/util/assignment.ts createCompoundAssignmentStatement.
func createCompoundAssignmentStatement(prereqs *Prereqs, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) *luau.Assignment {
	return luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: "=",
		Right:    createBinaryFromOperator(prereqs, readable, writableType, operator, value, valueType),
	})
}

// createCompoundAssignmentExpression is TSTransformer/util/assignment.ts createCompoundAssignmentExpression.
func createCompoundAssignmentExpression(prereqs *Prereqs, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) luau.WritableExpression {
	return createAssignmentExpression(
		prereqs,
		writable,
		"=",
		createBinaryFromOperator(prereqs, readable, writableType, operator, value, valueType),
		nil,
	)
}
