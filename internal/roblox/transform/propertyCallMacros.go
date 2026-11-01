package transform

// Port of TSTransformer/macros/propertyCallMacros.ts.

import (
	"maps"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func makeMathMethod(operator luau.BinaryOperator) PropertyCallMacro {
	return func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return luau.Binary(expression, operator, args[0])
	}
}

var OPERATOR_TO_NAME_MAP = map[luau.BinaryOperator]string{
	"+":  "add",
	"-":  "sub",
	"*":  "mul",
	"/":  "div",
	"//": "idiv",
}

func makeMathSet(operators ...luau.BinaryOperator) MacroList[PropertyCallMacro] {
	result := MacroList[PropertyCallMacro]{}
	for _, operator := range operators {
		methodName, ok := OPERATOR_TO_NAME_MAP[operator]
		assert(ok)
		result[methodName] = makeMathMethod(operator)
	}
	return result
}

func makeStringCallback(strCallback *luau.PropertyAccessExpression) PropertyCallMacro {
	return func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		return luau.Call(strCallback, append([]luau.Expression{expression}, args...)...)
	}
}

func STRING_CALLBACKS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"size": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Unary("#", expression)
		},

		"byte":    makeStringCallback(luau.Globals.String.Byte),
		"find":    makeStringCallback(luau.Globals.String.Find),
		"format":  makeStringCallback(luau.Globals.String.Format),
		"gmatch":  makeStringCallback(luau.Globals.String.Gmatch),
		"gsub":    makeStringCallback(luau.Globals.String.Gsub),
		"lower":   makeStringCallback(luau.Globals.String.Lower),
		"match":   makeStringCallback(luau.Globals.String.Match),
		"rep":     makeStringCallback(luau.Globals.String.Rep),
		"reverse": makeStringCallback(luau.Globals.String.Reverse),
		"split":   makeStringCallback(luau.Globals.String.Split),
		"sub":     makeStringCallback(luau.Globals.String.Sub),
		"upper":   makeStringCallback(luau.Globals.String.Upper),
	}
}

type callbackArgsListMaker = func(keyId *luau.TemporaryIdentifier, valueId *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression

func makeEveryOrSomeMethod(callbackArgsListMaker callbackArgsListMaker, initialState bool) PropertyCallMacro {
	return func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
		resultId := prereqs.PushToVar(luau.Bool(initialState), "result")
		callbackId := convertToIndexableExpression(args[0])

		keyId := luau.TempId("k")
		valueId := luau.TempId("v")

		callCallback := luau.Call(callbackId, callbackArgsListMaker(keyId, valueId, expression)...)
		var condition luau.Expression = callCallback
		if initialState {
			condition = luau.Unary("not", callCallback)
		}
		prereqs.Push(
			luau.Create(&luau.ForStatement{
				Ids:        luau.MakeList(keyId, valueId),
				Expression: expression,
				Statements: luau.MakeList(
					luau.Create(&luau.IfStatement{
						Condition: condition,
						Statements: luau.MakeList(
							luau.Create(&luau.Assignment{
								Left:     resultId,
								Operator: "=",
								Right:    luau.Bool(!initialState),
							}),
							luau.Create(&luau.BreakStatement{}),
						),
						ElseBody: luau.MakeList(),
					}),
				),
			}),
		)

		return resultId
	}
}

func makeEveryMethod(callbackArgsListMaker callbackArgsListMaker) PropertyCallMacro {
	return makeEveryOrSomeMethod(callbackArgsListMaker, true)
}

func makeSomeMethod(callbackArgsListMaker callbackArgsListMaker) PropertyCallMacro {
	return makeEveryOrSomeMethod(callbackArgsListMaker, false)
}

func argumentsWithDefaults(state *TransformState, node *ast.Node, args []luau.Expression, defaults []luau.Expression) []luau.Expression {
	arguments := node.Arguments()
	for i := range defaults {
		var argument luau.Expression
		if i < len(args) {
			argument = args[i]
		}
		var source *ast.Node
		if i < len(arguments) {
			source = arguments[i]
		}
		if argument == nil || luau.IsNilLiteral(argument) {
			for len(args) <= i {
				args = append(args, nil)
			}
			args[i] = defaults[i]
		} else if !luau.IsSimplePrimitive(argument) &&
			(source == nil || ast.IsSpreadElement(source) || isPossiblyType(state.GetType(source), isUndefinedType)) {
			args[i] = luau.Create(&luau.IfExpression{
				Condition:   luau.Binary(argument, "==", luau.Nil()),
				Expression:  defaults[i],
				Alternative: argument,
			})
		}
	}

	return args
}

// resultOfStatementMacro is `!isUsedAsStatement(node) ? value : luau.none()`.
func resultOfStatementMacro(node *ast.Node, value luau.Expression) luau.Expression {
	if !isUsedAsStatement(node) {
		return value
	}
	return luau.NewNone()
}

func ARRAY_LIKE_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"size": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Unary("#", expression)
		},
	}
}

func READONLY_ARRAY_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"isEmpty": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Binary(luau.Unary("#", expression), "==", luau.Number(0))
		},

		"join": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			args = argumentsWithDefaults(state, node, args, []luau.Expression{luau.Strings.CommaSpace})
			indexType := state.TypeChecker.GetIndexTypeOfType(
				state.GetType(node.Expression().Expression()),
				state.TypeChecker.GetNumberType(),
			)

			// table.concat only works on string and number types, so call tostring() otherwise
			if indexType != nil && !isDefinitelyType(indexType, isStringType, isNumberType) {
				id := prereqs.PushToVar(luau.Call(luau.Globals.Table.Create, luau.Unary("#", expression)), "result")
				keyId := luau.TempId("k")
				valueId := luau.TempId("v")
				prereqs.Push(
					luau.Create(&luau.ForStatement{
						Ids:        luau.MakeList(keyId, valueId),
						Expression: expression,
						Statements: luau.MakeList(
							luau.Create(&luau.Assignment{
								Left: luau.Create(&luau.ComputedIndexExpression{
									Expression: id,
									Index:      keyId,
								}),
								Operator: "=",
								Right:    luau.Call(luau.Globals.Tostring, valueId),
							}),
						),
					}),
				)

				expression = id
			}

			return luau.Call(luau.Globals.Table.Concat, expression, args[0])
		},

		"move": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			moveArgs := []luau.Expression{expression, offset(args[0], 1), offset(args[1], 1), offset(args[2], 1)}
			if len(args) > 3 && args[3] != nil {
				moveArgs = append(moveArgs, args[3])
			}
			return luau.Call(luau.Globals.Table.Move, moveArgs...)
		},

		"includes": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callArgs := []luau.Expression{expression, args[0]}
			if len(args) > 1 && args[1] != nil {
				callArgs = append(callArgs, offset(args[1], 1))
			}
			return luau.Binary(luau.Call(luau.Globals.Table.Find, callArgs...), "~=", luau.Nil())
		},

		"indexOf": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			findArgs := []luau.Expression{expression, args[0]}

			if len(args) > 1 {
				findArgs = append(findArgs, offset(args[1], 1))
			}

			return offset(
				luau.Create(&luau.BinaryExpression{
					Left:     luau.Call(luau.Globals.Table.Find, findArgs...),
					Operator: "or",
					Right:    luau.Number(0),
				}),
				-1,
			)
		},

		"every": makeEveryMethod(func(keyId *luau.TemporaryIdentifier, valueId *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression {
			return []luau.Expression{valueId, offset(keyId, -1), expression}
		}),

		"some": makeSomeMethod(func(keyId *luau.TemporaryIdentifier, valueId *luau.TemporaryIdentifier, expression luau.Expression) []luau.Expression {
			return []luau.Expression{valueId, offset(keyId, -1), expression}
		}),

		"forEach": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callbackId := convertToIndexableExpression(args[0])
			keyId := luau.TempId("k")
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(keyId, valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.CallStatement{
							Expression: luau.Call(callbackId, valueId, offset(keyId, -1), expression),
						}),
					),
				}),
			)

			return resultOfStatementMacro(node, luau.Nil())
		},

		"map": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			newValueId := prereqs.PushToVar(
				luau.Call(luau.Globals.Table.Create, luau.Unary("#", expression)),
				"newValue",
			)
			callbackId := convertToIndexableExpression(args[0])
			keyId := luau.TempId("k")
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(keyId, valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{
							Left: luau.Create(&luau.ComputedIndexExpression{
								Expression: newValueId,
								Index:      keyId,
							}),
							Operator: "=",
							Right:    luau.Call(callbackId, valueId, offset(keyId, -1), expression),
						}),
					),
				}),
			)

			return newValueId
		},

		"mapFiltered": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			newValueId := prereqs.PushToVar(luau.NewArray(), "newValue")
			callbackId := convertToIndexableExpression(args[0])
			lengthId := prereqs.PushToVar(luau.Number(0), "length")
			keyId := luau.TempId("k")
			valueId := luau.TempId("v")
			resultId := luau.TempId("result")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(keyId, valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.VariableDeclaration{
							Left:  resultId,
							Right: luau.Call(callbackId, valueId, offset(keyId, -1), expression),
						}),
						luau.Create(&luau.IfStatement{
							Condition: luau.Binary(resultId, "~=", luau.Nil()),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     lengthId,
									Operator: "+=",
									Right:    luau.Number(1),
								}),
								luau.Create(&luau.Assignment{
									Left: luau.Create(&luau.ComputedIndexExpression{
										Expression: newValueId,
										Index:      lengthId,
									}),
									Operator: "=",
									Right:    resultId,
								}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			return newValueId
		},

		"filterUndefined": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			lengthId := prereqs.PushToVar(luau.Number(0), "length")
			indexId1 := luau.TempId("i")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(indexId1),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.IfStatement{
							Condition: luau.Binary(indexId1, ">", lengthId),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     lengthId,
									Operator: "=",
									Right:    indexId1,
								}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			resultId := prereqs.PushToVar(luau.NewArray(), "result")
			resultLengthId := prereqs.PushToVar(luau.Number(0), "resultLength")
			indexId2 := luau.TempId("i")
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.NumericForStatement{
					Id:    indexId2,
					Start: luau.Number(1),
					End:   lengthId,
					Step:  nil,

					Statements: luau.MakeList(
						luau.Create(&luau.VariableDeclaration{
							Left: valueId,
							Right: luau.Create(&luau.ComputedIndexExpression{
								Expression: convertToIndexableExpression(expression),
								Index:      indexId2,
							}),
						}),
						luau.Create(&luau.IfStatement{
							Condition: luau.Binary(valueId, "~=", luau.Nil()),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     resultLengthId,
									Operator: "+=",
									Right:    luau.Number(1),
								}),
								luau.Create(&luau.Assignment{
									Left: luau.Create(&luau.ComputedIndexExpression{
										Expression: resultId,
										Index:      resultLengthId,
									}),
									Operator: "=",
									Right:    valueId,
								}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			return resultId
		},

		"filter": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			newValueId := prereqs.PushToVar(luau.NewArray(), "newValue")
			callbackId := convertToIndexableExpression(args[0])
			lengthId := prereqs.PushToVar(luau.Number(0), "length")
			keyId := luau.TempId("k")
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(keyId, valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.IfStatement{
							Condition: luau.Create(&luau.BinaryExpression{
								Left:     luau.Call(callbackId, valueId, offset(keyId, -1), expression),
								Operator: "==",
								Right:    luau.Bool(true),
							}),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     lengthId,
									Operator: "+=",
									Right:    luau.Number(1),
								}),
								luau.Create(&luau.Assignment{
									Left: luau.Create(&luau.ComputedIndexExpression{
										Expression: newValueId,
										Index:      lengthId,
									}),
									Operator: "=",
									Right:    valueId,
								}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			return newValueId
		},

		"reduce": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			var start luau.Expression = luau.Number(1)
			end := luau.Unary("#", expression)

			lengthExp := luau.Unary("#", expression)

			var resultId *luau.TemporaryIdentifier
			// if there was no initialValue supplied
			if len(args) < 2 {
				prereqs.Push(
					luau.Create(&luau.IfStatement{
						Condition: luau.Binary(lengthExp, "==", luau.Number(0)),
						Statements: luau.MakeList(
							luau.Create(&luau.CallStatement{
								Expression: luau.Call(luau.Globals.Error,
									luau.String(
										"Attempted to call `ReadonlyArray.reduce()` on an empty array without an initialValue.",
									),
								),
							}),
						),
						ElseBody: luau.MakeList(),
					}),
				)
				resultId = prereqs.PushToVar(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      start,
					}),
					"result",
				)
				start = offset(start, 1)
			} else {
				resultId = prereqs.PushToVar(args[1], "result")
			}
			callbackId := convertToIndexableExpression(args[0])

			iteratorId := luau.TempId("i")
			prereqs.Push(
				luau.Create(&luau.NumericForStatement{
					Id:    iteratorId,
					Start: start,
					End:   end,
					Step:  nil,
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{
							Left:     resultId,
							Operator: "=",
							Right: luau.Call(callbackId,
								resultId,
								luau.Create(&luau.ComputedIndexExpression{
									Expression: convertToIndexableExpression(expression),
									Index:      iteratorId,
								}),
								offset(iteratorId, -1),
								expression,
							),
						}),
					),
				}),
			)

			return resultId
		},

		"find": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callbackId := convertToIndexableExpression(args[0])
			loopId := luau.TempId("i")
			valueId := luau.TempId("v")
			resultId := prereqs.PushToVar(nil, "result")

			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Expression: expression,
					Ids:        luau.MakeList(loopId, valueId),
					Statements: luau.MakeList(
						luau.Create(&luau.IfStatement{
							Condition: luau.Create(&luau.BinaryExpression{
								Left:     luau.Call(callbackId, valueId, offset(loopId, -1), expression),
								Operator: "==",
								Right:    luau.Bool(true),
							}),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     resultId,
									Operator: "=",
									Right:    valueId,
								}),
								luau.Create(&luau.BreakStatement{}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			return resultId
		},

		"findIndex": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callbackId := convertToIndexableExpression(args[0])
			loopId := luau.TempId("i")
			valueId := luau.TempId("v")
			resultId := prereqs.PushToVar(luau.Number(-1), "result")

			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Expression: expression,
					Ids:        luau.MakeList(loopId, valueId),
					Statements: luau.MakeList(
						luau.Create(&luau.IfStatement{
							Condition: luau.Create(&luau.BinaryExpression{
								Left:     luau.Call(callbackId, valueId, offset(loopId, -1), expression),
								Operator: "==",
								Right:    luau.Bool(true),
							}),
							Statements: luau.MakeList(
								luau.Create(&luau.Assignment{
									Left:     resultId,
									Operator: "=",
									Right:    offset(loopId, -1),
								}),
								luau.Create(&luau.BreakStatement{}),
							),
							ElseBody: luau.MakeList(),
						}),
					),
				}),
			)

			return resultId
		},
	}
}

func ARRAY_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"push": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			// for `a.push()` always emit luau.unary so the call doesn't disappear in emit
			if len(args) == 0 {
				return luau.Unary("#", expression)
			}

			for i := range args {
				prereqs.Push(
					luau.Create(&luau.CallStatement{
						Expression: luau.Call(luau.Globals.Table.Insert, expression, args[i]),
					}),
				)
			}

			return resultOfStatementMacro(node, luau.Unary("#", expression))
		},

		"pop": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			var lengthExp luau.Expression = luau.Unary("#", expression)

			returnValueIsUsed := !isUsedAsStatement(node)
			var retValue *luau.TemporaryIdentifier
			if returnValueIsUsed {
				lengthExp = prereqs.PushToVar(lengthExp, "length")
				retValue = prereqs.PushToVar(
					luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      lengthExp,
					}),
					"result",
				)
			}

			prereqs.Push(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      lengthExp,
					}),
					Operator: "=",
					Right:    luau.Nil(),
				}),
			)

			if returnValueIsUsed {
				return retValue
			}
			return luau.NewNone()
		},

		"shift": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Call(luau.Globals.Table.Remove, expression, luau.Number(1))
		},

		"unshift": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			for i := len(args) - 1; i >= 0; i-- {
				arg := args[i]
				prereqs.Push(
					luau.Create(&luau.CallStatement{
						Expression: luau.Call(luau.Globals.Table.Insert, expression, luau.Number(1), arg),
					}),
				)
			}

			return resultOfStatementMacro(node, luau.Unary("#", expression))
		},

		"insert": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Call(luau.Globals.Table.Insert, expression, offset(args[0], 1), args[1])
		},

		"remove": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Call(luau.Globals.Table.Remove, expression, offset(args[0], 1))
		},

		"unorderedRemove": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			indexExp := prereqs.PushToVarIfComplex(offset(args[0], 1), "index")

			lengthId := prereqs.PushToVar(luau.Unary("#", expression), "length")

			valueIsUsed := !isUsedAsStatement(node)
			valueId := prereqs.PushToVar(
				luau.Create(&luau.ComputedIndexExpression{
					Expression: convertToIndexableExpression(expression),
					Index:      indexExp,
				}),
				"value",
			)

			prereqs.Push(
				luau.Create(&luau.IfStatement{
					Condition: luau.Binary(valueId, "~=", luau.Nil()),
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{
							Left: luau.Create(&luau.ComputedIndexExpression{
								Expression: convertToIndexableExpression(expression),
								Index:      indexExp,
							}),
							Operator: "=",
							Right: luau.Create(&luau.ComputedIndexExpression{
								Expression: convertToIndexableExpression(expression),
								Index:      lengthId,
							}),
						}),
						luau.Create(&luau.Assignment{
							Left: luau.Create(&luau.ComputedIndexExpression{
								Expression: convertToIndexableExpression(expression),
								Index:      lengthId,
							}),
							Operator: "=",
							Right:    luau.Nil(),
						}),
					),
					ElseBody: luau.MakeList(),
				}),
			)

			if valueIsUsed {
				return valueId
			}
			return luau.NewNone()
		},

		"sort": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			valueIsUsed := !isUsedAsStatement(node)

			args = append([]luau.Expression{expression}, args...)

			prereqs.Push(
				luau.Create(&luau.CallStatement{
					Expression: luau.Call(luau.Globals.Table.Sort, args...),
				}),
			)

			if valueIsUsed {
				return expression
			}
			return luau.NewNone()
		},

		"clear": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			prereqs.Push(
				luau.Create(&luau.CallStatement{
					Expression: luau.Call(luau.Globals.Table.Clear, expression),
				}),
			)
			return resultOfStatementMacro(node, luau.Nil())
		},
	}
}

func READONLY_SET_MAP_SHARED_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"isEmpty": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Binary(luau.Call(luau.Globals.Next, expression), "==", luau.Nil())
		},

		"size": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			sizeId := prereqs.PushToVar(luau.Number(0), "size")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(luau.TempId()),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{
							Left:     sizeId,
							Operator: "+=",
							Right:    luau.Number(1),
						}),
					),
				}),
			)
			return sizeId
		},

		"has": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			left := luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      args[0],
			})
			return luau.Binary(left, "~=", luau.Nil())
		},
	}
}

func SET_MAP_SHARED_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"delete": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			arg := args[0]
			valueIsUsed := !isUsedAsStatement(node)
			var valueExistedId *luau.TemporaryIdentifier
			if valueIsUsed {
				valueExistedId = prereqs.PushToVar(
					luau.Create(&luau.BinaryExpression{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: convertToIndexableExpression(expression),
							Index:      arg,
						}),
						Operator: "~=",
						Right:    luau.Nil(),
					}),
					"valueExisted",
				)
			}

			prereqs.Push(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      arg,
					}),
					Operator: "=",
					Right:    luau.Nil(),
				}),
			)

			if valueIsUsed {
				return valueExistedId
			}
			return luau.NewNone()
		},

		"clear": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			prereqs.Push(
				luau.Create(&luau.CallStatement{
					Expression: luau.Call(luau.Globals.Table.Clear, expression),
				}),
			)
			return resultOfStatementMacro(node, luau.Nil())
		},
	}
}

// mergeMacroLists is the object spread `{ ...a, ...b }`.
func mergeMacroLists(lists ...MacroList[PropertyCallMacro]) MacroList[PropertyCallMacro] {
	result := MacroList[PropertyCallMacro]{}
	for _, list := range lists {
		maps.Copy(result, list)
	}
	return result
}

func READONLY_SET_METHODS() MacroList[PropertyCallMacro] {
	return mergeMacroLists(READONLY_SET_MAP_SHARED_METHODS(), MacroList[PropertyCallMacro]{
		"forEach": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callbackId := convertToIndexableExpression(args[0])
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.CallStatement{
							Expression: luau.Call(callbackId, valueId, valueId, expression),
						}),
					),
				}),
			)

			return resultOfStatementMacro(node, luau.Nil())
		},
	})
}

func SET_METHODS() MacroList[PropertyCallMacro] {
	return mergeMacroLists(SET_MAP_SHARED_METHODS(), MacroList[PropertyCallMacro]{
		"add": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			valueIsUsed := !isUsedAsStatement(node)
			prereqs.Push(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      args[0],
					}),
					Operator: "=",
					Right:    luau.Bool(true),
				}),
			)
			if valueIsUsed {
				return expression
			}
			return luau.NewNone()
		},
	})
}

func READONLY_MAP_METHODS() MacroList[PropertyCallMacro] {
	return mergeMacroLists(READONLY_SET_MAP_SHARED_METHODS(), MacroList[PropertyCallMacro]{
		"forEach": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			callbackId := convertToIndexableExpression(args[0])
			keyId := luau.TempId("k")
			valueId := luau.TempId("v")
			prereqs.Push(
				luau.Create(&luau.ForStatement{
					Ids:        luau.MakeList(keyId, valueId),
					Expression: expression,
					Statements: luau.MakeList(
						luau.Create(&luau.CallStatement{
							Expression: luau.Call(callbackId, valueId, keyId, expression),
						}),
					),
				}),
			)

			return resultOfStatementMacro(node, luau.Nil())
		},

		"get": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Create(&luau.ComputedIndexExpression{
				Expression: convertToIndexableExpression(expression),
				Index:      args[0],
			})
		},
	})
}

func MAP_METHODS() MacroList[PropertyCallMacro] {
	return mergeMacroLists(SET_MAP_SHARED_METHODS(), MacroList[PropertyCallMacro]{
		"set": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			keyExp, valueExp := args[0], args[1]
			valueIsUsed := !isUsedAsStatement(node)
			prereqs.Push(
				luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: convertToIndexableExpression(expression),
						Index:      keyExp,
					}),
					Operator: "=",
					Right:    valueExp,
				}),
			)
			if valueIsUsed {
				return expression
			}
			return luau.NewNone()
		},
	})
}

func PROMISE_METHODS() MacroList[PropertyCallMacro] {
	return MacroList[PropertyCallMacro]{
		"then": func(state *TransformState, prereqs *Prereqs, node *ast.Node, expression luau.Expression, args []luau.Expression) luau.Expression {
			return luau.Create(&luau.MethodCallExpression{
				Expression: convertToIndexableExpression(expression),
				Name:       "andThen",
				Args:       luau.MakeList(args...),
			})
		},
	}
}

// PROPERTY_CALL_MACROS is filled by init with the comment wrapping applied. The MacroLists are
// functions so that every class gets its own copy of the shared method lists, like the object
// spreads of the TypeScript source, and so that the table does not form an initialization cycle
// with the transformers the macros call.
var PROPERTY_CALL_MACROS map[string]MacroList[PropertyCallMacro]

func init() {
	PROPERTY_CALL_MACROS = map[string]MacroList[PropertyCallMacro]{
		// math classes
		"CFrame":       makeMathSet("+", "-", "*"),
		"UDim":         makeMathSet("+", "-"),
		"UDim2":        makeMathSet("+", "-"),
		"Vector2":      makeMathSet("+", "-", "*", "/", "//"),
		"Vector2int16": makeMathSet("+", "-", "*", "/"),
		"Vector3":      makeMathSet("+", "-", "*", "/", "//"),
		"Vector3int16": makeMathSet("+", "-", "*", "/"),
		"vector":       makeMathSet("+", "-", "*", "/", "//"),
		"Number":       makeMathSet("//"),

		"String":        STRING_CALLBACKS(),
		"ArrayLike":     ARRAY_LIKE_METHODS(),
		"ReadonlyArray": READONLY_ARRAY_METHODS(),
		"Array":         ARRAY_METHODS(),
		"ReadonlySet":   READONLY_SET_METHODS(),
		"Set":           SET_METHODS(),
		"ReadonlyMap":   READONLY_MAP_METHODS(),
		"Map":           MAP_METHODS(),
		"Promise":       PROMISE_METHODS(),
	}

	// apply comment wrapping
	for className, macroList := range PROPERTY_CALL_MACROS {
		for methodName, macro := range macroList {
			macroList[methodName] = wrapComments(className+"."+methodName, macro)
		}
	}
}

// comment logic

func header(text string) luau.Statement {
	return luau.NewComment(" ▼ " + text + " ▼")
}

func footer(text string) luau.Statement {
	return luau.NewComment(" ▲ " + text + " ▲")
}

func wrapComments(methodName string, callback PropertyCallMacro) PropertyCallMacro {
	return func(state *TransformState, prereqs *Prereqs, callNode *ast.Node, callExp luau.Expression, args []luau.Expression) luau.Expression {
		expressionPrereqs := NewPrereqs()
		expression := callback(state, expressionPrereqs, callNode, callExp, args)
		if expressionPrereqs.Statements.Size() > 1 && !state.Data.ProjectOptions.StripComments {
			expressionPrereqs.Unshift(header(methodName))
			expressionPrereqs.Push(footer(methodName))
		}
		prereqs.PushList(expressionPrereqs.Statements)
		return expression
	}
}
