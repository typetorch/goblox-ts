package transform

// Port of TSTransformer/util/assignment.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getSimpleAssignmentOperator is TSTransformer/util/assignment.ts getSimpleAssignmentOperator.
func getSimpleAssignmentOperator(leftType *checker.Type, operatorKind ast.Kind, rightType *checker.Type) luau.AssignmentOperator {
	panic(todo("util/assignment.ts", "getSimpleAssignmentOperator"))
}

// createAssignmentStatement is TSTransformer/util/assignment.ts createAssignmentStatement.
func createAssignmentStatement(writable luau.WritableExpression, operator luau.AssignmentOperator, value luau.Expression, readable luau.WritableExpression) *luau.Assignment {
	panic(todo("util/assignment.ts", "createAssignmentStatement"))
}

// createAssignmentExpression is TSTransformer/util/assignment.ts createAssignmentExpression.
func createAssignmentExpression(prereqs *Prereqs, writable luau.WritableExpression, operator luau.AssignmentOperator, value luau.Expression, readable luau.WritableExpression) luau.WritableExpression {
	panic(todo("util/assignment.ts", "createAssignmentExpression"))
}

// createCompoundAssignmentStatement is TSTransformer/util/assignment.ts createCompoundAssignmentStatement.
func createCompoundAssignmentStatement(prereqs *Prereqs, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) *luau.Assignment {
	panic(todo("util/assignment.ts", "createCompoundAssignmentStatement"))
}

// createCompoundAssignmentExpression is TSTransformer/util/assignment.ts createCompoundAssignmentExpression.
func createCompoundAssignmentExpression(prereqs *Prereqs, writable luau.WritableExpression, writableType *checker.Type, readable luau.WritableExpression, operator ast.Kind, value luau.Expression, valueType *checker.Type) luau.WritableExpression {
	panic(todo("util/assignment.ts", "createCompoundAssignmentExpression"))
}
