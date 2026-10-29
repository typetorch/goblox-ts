package transform

// Port of TSTransformer/util/offset.ts.

import (
	"math"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// offset is TSTransformer/util/offset.ts offset.
func offset(expression luau.Expression, value float64) luau.Expression {
	if value == 0 {
		return expression
	}

	// this special case handles when the expression is a binary expression with a number literal on its right
	// i.e. array[offset - 1 + 1] -> array[offset]
	if binary, ok := expression.(*luau.BinaryExpression); ok && (binary.Operator == "+" || binary.Operator == "-") {
		if rightValue, ok := getLiteralNumberValue(binary.Right); ok {
			sign := 1.0
			if binary.Operator == "-" {
				sign = -1
			}
			newRightValue := rightValue + value*sign
			if newRightValue == 0 {
				return binary.Left
			} else {
				return luau.Binary(binary.Left, binary.Operator, luau.Number(newRightValue))
			}
		}
	}

	if literalValue, ok := getLiteralNumberValue(expression); ok {
		return luau.Number(literalValue + value)
	} else {
		operator := luau.BinaryOperator("-")
		if value > 0 {
			operator = "+"
		}
		return luau.Binary(expression, operator, luau.Number(math.Abs(value)))
	}
}
