package transform

// Port of TSTransformer/nodes/expressions/transformElementAccessExpression.ts.

import (
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// numberLiteralValue is JavaScript's Number(value.replace(/_/g, "")) for a Luau number literal's text.
// Unparsable text (NaN in JavaScript) gives -1, which like NaN is never 0.
func numberLiteralValue(value string) float64 {
	value = strings.ReplaceAll(value, "_", "")
	if result, err := strconv.ParseFloat(value, 64); err == nil {
		return result
	}
	if result, err := strconv.ParseInt(value, 0, 64); err == nil {
		return float64(result)
	}
	return -1
}

// transformElementAccessExpressionInner is TSTransformer/nodes/expressions/transformElementAccessExpression.ts transformElementAccessExpressionInner.
func transformElementAccessExpressionInner(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, argumentExpression *ast.Node) luau.Expression {
	// a in a[b]
	validateNotAnyType(state, node.Expression())
	// b in a[b]
	validateNotAnyType(state, node.AsElementAccessExpression().ArgumentExpression)

	receiverType := state.TypeChecker.GetNonOptionalType(state.GetType(node.Expression()))
	memberType := state.TypeChecker.GetNonOptionalType(state.GetType(node))
	addIndexDiagnostics(state, node, memberType, receiverType)

	indexPrereqs := NewPrereqs()
	index := transformExpression(state, indexPrereqs, argumentExpression)

	// optional chains already guard this access against nil
	nonNullableType := state.TypeChecker.GetNonNullableType(receiverType)
	if isDefinitelyType(nonNullableType, isStringType) {
		return createStringIndexExpression(
			prereqs,
			expression,
			&EvaluationOperand{Expression: index, Prereqs: indexPrereqs.Statements},
			state.GetType(argumentExpression),
		)
	}

	if isMixedStringType(nonNullableType) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMixedStringIndex.Create(node))
		return luau.NewNone()
	}

	lateIndexEffects := NO_EFFECTS
	if isLateRead(expression) {
		lateIndexEffects = getEffects(index)
	}
	if !effectsCommute(getEffects(expression), joinEffects(getEffectsOfListOrNode(indexPrereqs.Statements), lateIndexEffects)) {
		// hack because wrapReturnIfLuaTuple will not wrap this, but now we need to!
		if isLuaTupleType(state)(receiverType) {
			expression = luau.NewArray(expression)
		}

		expression = prereqs.PushToVar(expression, "exp")
	}
	prereqs.PushList(indexPrereqs.Statements)

	// LuaTuple<T> checks
	if luau.IsCall(expression) && isLuaTupleType(state)(receiverType) {
		// wrap in select() if it isn't the first value
		if numberLiteral, ok := index.(*luau.NumberLiteral); !ok || numberLiteralValue(numberLiteral.Value) != 0 {
			expression = luau.Call(luau.Globals.Select, offset(index, 1), expression)
		}
		// parentheses to trim off the rest of the values
		return luau.Create(&luau.ParenthesizedExpression{Expression: expression})
	}

	if ast.IsDeleteExpression(skipUpwards(node).Parent) {
		prereqs.Push(luau.Create(&luau.Assignment{
			Left: luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      addOneIfArrayType(state, receiverType, index),
			}),
			Operator: "=",
			Right:    luau.Nil(),
		}))
		return luau.NewNone()
	}

	access := luau.Create(&luau.ComputedIndexExpression{
		Expression: convertToIndexableExpression(expression),
		Index:      addOneIfArrayType(state, receiverType, index),
	})
	tryMarkBuiltinMember(state, node, access)
	return access
}

// transformElementAccessExpression is TSTransformer/nodes/expressions/transformElementAccessExpression.ts transformElementAccessExpression.
func transformElementAccessExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	if constantValue := getConstantValueLiteral(state, node); constantValue != nil {
		return constantValue
	}

	return transformOptionalChain(state, prereqs, node)
}
