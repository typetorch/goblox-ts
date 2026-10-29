package transform

// Port of TSTransformer/util/createStringIndexExpression.ts.

import (
	"math"

	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// isInteger is JavaScript's Number.isInteger.
func isInteger(value float64) bool {
	return !math.IsInf(value, 0) && value == math.Trunc(value)
}

// createStringIndexExpression is TSTransformer/util/createStringIndexExpression.ts createStringIndexExpression.
func createStringIndexExpression(prereqs *Prereqs, expression luau.Expression, indexOperand *EvaluationOperand, indexType *checker.Type) luau.Expression {
	var literalIndex float64
	var hasLiteralIndex bool
	if indexType.IsStringLiteral() {
		literalIndex, hasLiteralIndex = float64(jsnum.FromString(typeStringValue(indexType))), true
	} else {
		literalIndex, hasLiteralIndex = getLiteralNumberValue(indexOperand.Expression)
	}

	return planEvaluation(
		prereqs,
		[]*EvaluationOperand{{Expression: expression, Prereqs: luau.MakeList()}, indexOperand},
		func(indexPrereqs *Prereqs, operands []luau.Expression) luau.Expression {
			value, index := operands[0], operands[1]
			if hasLiteralIndex {
				if !isInteger(literalIndex) || literalIndex < 0 {
					return luau.Nil()
				}
				if indexType.IsStringLiteral() {
					index = luau.Number(literalIndex)
				}
			}

			var conditions []luau.Expression
			if !hasLiteralIndex {
				// TypeScript also accepts numeric string keys such as "0"
				if isPossiblyType(indexType, isStringType) {
					key := index
					index = indexPrereqs.PushToVar(luau.Call(luau.Id("tonumber"), index), "index")
					conditions = append(conditions, luau.Binary(index, "~=", luau.Nil()))

					// string keys must spell a nonnegative integer without signs, padding, or exponents
					var canonicalKey luau.Expression = luau.Binary(
						luau.Binary(key, "==", luau.String("0")),
						"or",
						luau.Binary(
							luau.Call(luau.Globals.String.Match, key, luau.String("^[1-9]%d*$")),
							"~=",
							luau.Nil(),
						),
					)
					if !isDefinitelyType(indexType, isStringType) {
						canonicalKey = luau.Binary(
							luau.Binary(luau.Call(luau.Globals.Type, key), "==", luau.String("number")),
							"or",
							canonicalKey,
						)
					}
					conditions = append(conditions, canonicalKey)
				}
				conditions = append(conditions, luau.Binary(index, ">=", luau.Number(0)))
				conditions = append(conditions, luau.Binary(luau.Binary(index, "%", luau.Number(1)), "==", luau.Number(0)))
			}
			conditions = append(conditions, luau.Binary(index, "<", luau.Unary("#", value)))

			// guard before string.sub so negative offsets and fractional indices cannot select a byte
			at := offset(index, 1)
			return luau.Create(&luau.IfExpression{
				Condition:   binaryExpressionChain(conditions, "and"),
				Expression:  luau.Call(luau.Globals.String.Sub, value, at, at),
				Alternative: luau.Nil(),
			})
		},
	)
}
