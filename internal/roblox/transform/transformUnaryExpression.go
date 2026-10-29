package transform

// Port of TSTransformer/nodes/expressions/transformUnaryExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

var UPDATE_OPERATORS = map[ast.Kind]luau.AssignmentOperator{
	ast.KindPlusPlusToken:   "+=",
	ast.KindMinusMinusToken: "-=",
}

// transformPostfixUnaryExpression is TSTransformer/nodes/expressions/transformUnaryExpression.ts transformPostfixUnaryExpression.
func transformPostfixUnaryExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.TemporaryIdentifier {
	expression := node.AsPostfixUnaryExpression()
	validateNotAnyType(state, expression.Operand)

	writable := transformWritableExpression(state, prereqs, expression.Operand, true)
	origValue := luau.TempId("original")
	operator, ok := UPDATE_OPERATORS[expression.Operator]
	assert(ok, "Unexpected postfix unary operator")

	prereqs.Push(luau.Create(&luau.VariableDeclaration{
		Left:  origValue,
		Right: writable,
	}))

	prereqs.Push(luau.Create(&luau.Assignment{
		Left:     writable,
		Operator: operator,
		Right:    luau.Number(1),
	}))

	return origValue
}

// transformPrefixUnaryExpression is TSTransformer/nodes/expressions/transformUnaryExpression.ts transformPrefixUnaryExpression.
func transformPrefixUnaryExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	expression := node.AsPrefixUnaryExpression()
	validateNotAnyType(state, expression.Operand)

	if expression.Operator == ast.KindPlusPlusToken || expression.Operator == ast.KindMinusMinusToken {
		writable := transformWritableExpression(state, prereqs, expression.Operand, true)
		operator := UPDATE_OPERATORS[expression.Operator]
		prereqs.Push(luau.Create(&luau.Assignment{
			Left:     writable,
			Operator: operator,
			Right:    luau.Number(1),
		}))
		return writable
	} else if expression.Operator == ast.KindPlusToken {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoUnaryPlus.Create(node))
		return transformExpression(state, prereqs, expression.Operand)
	} else if expression.Operator == ast.KindMinusToken {
		if !isDefinitelyType(state.GetType(expression.Operand), isNumberType) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoNonNumberUnaryMinus.Create(node))
		}
		return luau.Unary("-", transformExpression(state, prereqs, expression.Operand))
	} else if expression.Operator == ast.KindExclamationToken {
		checks := createTruthinessChecks(
			state,
			prereqs,
			transformExpression(state, prereqs, expression.Operand),
			expression.Operand,
		)
		return luau.Unary("not", checks)
	}

	assert(expression.Operator == ast.KindTildeToken, "Unexpected prefix unary operator")
	return luau.Call(luau.Property(luau.Globals.Bit32, "bnot"), transformExpression(state, prereqs, expression.Operand))
}
