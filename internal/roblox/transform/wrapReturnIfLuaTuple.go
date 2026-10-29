package transform

// Port of TSTransformer/util/wrapReturnIfLuaTuple.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func shouldWrapLuaTuple(state *TransformState, node *ast.Node, exp luau.Expression) bool {
	if !luau.IsCall(exp) {
		return true
	}

	child := skipUpwards(node)
	parent := child.Parent
	questionDotToken := node.AsCallExpression().QuestionDotToken

	// `foo();`
	if ast.IsExpressionStatement(parent) {
		return false
	}

	// if part of for statement definition, except if used as the condition
	if ast.IsForStatement(parent) && parent.AsForStatement().Condition != child {
		return false
	}

	// `const [a] = foo()`
	if ast.IsVariableDeclaration(parent) &&
		ast.IsArrayBindingPattern(parent.Name()) &&
		!arrayBindingPatternContainsHoists(state, parent.Name()) &&
		!arrayLikeExpressionContainsSpread(parent.Name()) &&
		questionDotToken == nil {
		return false
	}

	// `[a] = foo()`
	if ast.IsAssignmentExpression(parent, false) &&
		ast.IsArrayLiteralExpression(parent.AsBinaryExpression().Left) &&
		!arrayLikeExpressionContainsSpread(parent.AsBinaryExpression().Left) &&
		questionDotToken == nil {
		return false
	}

	// `foo()[n]`
	if ast.IsElementAccessExpression(parent) && parent.AsElementAccessExpression().QuestionDotToken == nil {
		return false
	}

	// `return foo()`
	if ast.IsReturnStatement(parent) {
		return false
	}

	// `void foo()`
	if ast.IsVoidExpression(parent) {
		return false
	}

	return true
}

// wrapReturnIfLuaTuple is TSTransformer/util/wrapReturnIfLuaTuple.ts wrapReturnIfLuaTuple.
func wrapReturnIfLuaTuple(state *TransformState, node *ast.Node, exp luau.Expression) luau.Expression {
	if isLuaTupleType(state)(state.TypeChecker.GetNonNullableType(state.GetType(node))) &&
		shouldWrapLuaTuple(state, node, exp) {
		return luau.NewArray(exp)
	}
	return exp
}
