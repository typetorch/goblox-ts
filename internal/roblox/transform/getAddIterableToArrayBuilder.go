package transform

// Port of TSTransformer/util/getAddIterableToArrayBuilder.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type AddIterableToArrayBuilder func(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List

func addArray(
	prereqs *Prereqs,
	expression luau.Expression,
	arrayId luau.AnyIdentifier,
	lengthId luau.AnyIdentifier,
	amtElementsSinceUpdate int,
	shouldUpdateLengthId bool,
) *luau.List {
	result := luau.MakeList()

	inputArray := prereqs.PushToVarIfNonId(expression, "array")
	var inputLength luau.Expression = luau.Unary("#", inputArray)
	if shouldUpdateLengthId {
		inputLength = prereqs.PushToVar(inputLength, valueToIdStr(inputArray)+"Length")
	}

	result.Push(luau.Create(&luau.CallStatement{
		Expression: luau.Call(luau.Globals.Table.Move,
			inputArray,
			luau.Number(1),
			inputLength,
			luau.Binary(lengthId, "+", luau.Number(float64(amtElementsSinceUpdate+1))),
			arrayId,
		),
	}))

	if shouldUpdateLengthId {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthId,
			Operator: "+=",
			Right:    inputLength,
		}))
	}

	return result
}

func createForLoopArrayBuilder(valueName string, getLoopExpression func(expression luau.Expression) luau.Expression) AddIterableToArrayBuilder {
	if getLoopExpression == nil {
		getLoopExpression = func(expression luau.Expression) luau.Expression { return expression }
	}
	return func(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List {
		result := luau.MakeList()

		if amtElementsSinceUpdate > 0 {
			result.Push(luau.Create(&luau.Assignment{
				Left:     lengthId,
				Operator: "+=",
				Right:    luau.Number(float64(amtElementsSinceUpdate)),
			}))
		}

		valueId := luau.TempId(valueName)

		result.Push(luau.Create(&luau.ForStatement{
			Ids:        luau.MakeList(valueId),
			Expression: getLoopExpression(expression),
			Statements: luau.MakeList(
				luau.Create(&luau.Assignment{
					Left:     lengthId,
					Operator: "+=",
					Right:    luau.Number(1),
				}),
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: arrayId,
						Index:      lengthId,
					}),
					Operator: "=",
					Right:    valueId,
				}),
			),
		}))

		return result
	}
}

var (
	addString = createForLoopArrayBuilder("char", func(expression luau.Expression) luau.Expression {
		return luau.Call(luau.Globals.String.Gmatch, expression, luau.Globals.Utf8.Charpattern)
	})
	addSet              = createForLoopArrayBuilder("v", nil)
	addIterableFunction = createForLoopArrayBuilder("result", nil)
)

func addMap(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List {
	result := luau.MakeList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthId,
			Operator: "+=",
			Right:    luau.Number(float64(amtElementsSinceUpdate)),
		}))
	}

	keyId := luau.TempId("k")
	valueId := luau.TempId("v")
	result.Push(luau.Create(&luau.ForStatement{
		Ids:        luau.MakeList(keyId, valueId),
		Expression: expression,
		Statements: luau.MakeList(
			luau.Create(&luau.Assignment{
				Left:     lengthId,
				Operator: "+=",
				Right:    luau.Number(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayId,
					Index:      lengthId,
				}),
				Operator: "=",
				Right:    luau.NewArray(keyId, valueId),
			}),
		),
	}))

	return result
}

func addIterableFunctionLuaTuple(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List {
	result := luau.MakeList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthId,
			Operator: "+=",
			Right:    luau.Number(float64(amtElementsSinceUpdate)),
		}))
	}

	iterFuncId := prereqs.PushToVar(expression, "iterFunc")
	valueId := luau.TempId("results")
	result.Push(luau.Create(&luau.WhileStatement{
		Condition: luau.Bool(true),
		Statements: luau.MakeList(
			luau.Create(&luau.VariableDeclaration{
				Left:  valueId,
				Right: luau.NewArray(luau.Call(iterFuncId)),
			}),
			luau.Create(&luau.IfStatement{
				// later return values do not keep an iterator alive after its first return is nil
				Condition: luau.Binary(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: valueId,
						Index:      luau.Number(1),
					}),
					"==",
					luau.Nil(),
				),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.MakeList(),
			}),
			luau.Create(&luau.Assignment{
				Left:     lengthId,
				Operator: "+=",
				Right:    luau.Number(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayId,
					Index:      lengthId,
				}),
				Operator: "=",
				Right:    valueId,
			}),
		),
	}))

	return result
}

func addGenerator(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List {
	result := luau.MakeList()

	if amtElementsSinceUpdate > 0 {
		result.Push(luau.Create(&luau.Assignment{
			Left:     lengthId,
			Operator: "+=",
			Right:    luau.Number(float64(amtElementsSinceUpdate)),
		}))
	}

	iterId := luau.TempId("result")
	result.Push(luau.Create(&luau.ForStatement{
		Ids:        luau.MakeList(iterId),
		Expression: luau.Property(convertToIndexableExpression(expression), "next"),
		Statements: luau.MakeList(
			luau.Create(&luau.IfStatement{
				Condition:  luau.Property(iterId, "done"),
				Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
				ElseBody:   luau.MakeList(),
			}),
			luau.Create(&luau.Assignment{
				Left:     lengthId,
				Operator: "+=",
				Right:    luau.Number(1),
			}),
			luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: arrayId,
					Index:      lengthId,
				}),
				Operator: "=",
				Right:    luau.Property(iterId, "value"),
			}),
		),
	}))

	return result
}

func emptyAddIterableToArrayBuilder(prereqs *Prereqs, expression luau.Expression, arrayId luau.AnyIdentifier, lengthId luau.AnyIdentifier, amtElementsSinceUpdate int, shouldUpdateLengthId bool) *luau.List {
	return luau.MakeList()
}

// getAddIterableToArrayBuilder is TSTransformer/util/getAddIterableToArrayBuilder.ts getAddIterableToArrayBuilder.
func getAddIterableToArrayBuilder(state *TransformState, node *ast.Node, type_ *checker.Type) AddIterableToArrayBuilder {
	if isDefinitelyType(type_, isArrayType(state)) {
		return addArray
	} else if isDefinitelyType(type_, isStringType) {
		return addString
	} else if isDefinitelyType(type_, isSetType(state)) {
		return addSet
	} else if isDefinitelyType(type_, isMapType(state)) || isDefinitelyType(type_, isSharedTableType(state)) {
		return addMap
	} else if isDefinitelyType(type_, isIterableFunctionLuaTupleType(state)) {
		return addIterableFunctionLuaTuple
	} else if isDefinitelyType(type_, isIterableFunctionType(state)) {
		return addIterableFunction
	} else if isDefinitelyType(type_, isGeneratorType(state)) {
		return addGenerator
	} else if isDefinitelyType(type_, isIterableType(state)) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIterableIteration.Create(node))
		return emptyAddIterableToArrayBuilder
	} else if type_.IsUnion() {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMacroUnion.Create(node))
		return emptyAddIterableToArrayBuilder
	} else {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoUnsupportedIteration.Create(node))
		return emptyAddIterableToArrayBuilder
	}
}
