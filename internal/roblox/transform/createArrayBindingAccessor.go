package transform

// Port of TSTransformer/util/binding/createArrayBindingAccessor.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type BindingAccessor interface {
	Read(prereqs *Prereqs, index int, omitted bool) luau.Expression
	Rest(prereqs *Prereqs, index int) luau.Expression
}

// bindingAccessor is a BindingAccessor object literal.
type bindingAccessor struct {
	read func(prereqs *Prereqs, index int, omitted bool) luau.Expression
	rest func(prereqs *Prereqs, index int) luau.Expression
}

func (a *bindingAccessor) Read(prereqs *Prereqs, index int, omitted bool) luau.Expression {
	return a.read(prereqs, index, omitted)
}

func (a *bindingAccessor) Rest(prereqs *Prereqs, index int) luau.Expression {
	return a.rest(prereqs, index)
}

// iteratorStep is the step function of createIteratorAccessor; it returns IteratorStep's value and done.
type iteratorStep func(prereqs *Prereqs, omitted bool) (value luau.Expression, done luau.Expression)

func ifNotDone(done luau.Expression, value luau.Expression) *luau.IfExpression {
	return luau.Create(&luau.IfExpression{
		Condition:   done,
		Expression:  luau.Nil(),
		Alternative: value,
	})
}

// prefix elements and rest consume one iterator and share its terminal state
func createIteratorAccessor(step iteratorStep, hasCompletionValue bool) BindingAccessor {
	var doneId luau.AnyIdentifier

	return &bindingAccessor{
		read: func(prereqs *Prereqs, index int, omitted bool) luau.Expression {
			previousDone := doneId
			stepPrereqs := NewPrereqs()
			value, done := step(stepPrereqs, omitted)
			if previousDone == nil {
				doneId = stepPrereqs.PushToVar(done, "done")
				prereqs.PushList(stepPrereqs.Statements)
				if omitted {
					return luau.NewNone()
				}
				if hasCompletionValue {
					return ifNotDone(doneId, value)
				}
				return value
			}

			var valueId *luau.TemporaryIdentifier
			if !omitted {
				valueId = prereqs.PushToVar(nil, "value")
			}
			stepPrereqs.Push(
				luau.Create(&luau.Assignment{Left: previousDone, Operator: "=", Right: done}),
			)
			if valueId != nil {
				var right luau.Expression = value
				if hasCompletionValue {
					right = ifNotDone(previousDone, value)
				}
				stepPrereqs.Push(
					luau.Create(&luau.Assignment{
						Left:     valueId,
						Operator: "=",
						Right:    right,
					}),
				)
			}
			prereqs.Push(
				luau.Create(&luau.IfStatement{
					Condition:  luau.Unary("not", previousDone),
					Statements: stepPrereqs.Statements,
					ElseBody:   luau.MakeList(),
				}),
			)
			if valueId != nil {
				return valueId
			}
			return luau.NewNone()
		},
		rest: func(prereqs *Prereqs, index int) luau.Expression {
			rest := prereqs.PushToVar(luau.NewArray(), "rest")
			restPrereqs := NewPrereqs()
			length := restPrereqs.PushToVar(luau.Number(0), "length")
			body := NewPrereqs()
			value, done := step(body, false)
			body.Push(
				luau.Create(&luau.IfStatement{
					Condition:  done,
					Statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
					ElseBody:   luau.MakeList(),
				}),
			)
			body.Push(luau.Create(&luau.Assignment{Left: length, Operator: "+=", Right: luau.Number(1)}))
			body.Push(
				luau.Create(&luau.Assignment{
					Left:     luau.Create(&luau.ComputedIndexExpression{Expression: rest, Index: length}),
					Operator: "=",
					Right:    value,
				}),
			)
			restPrereqs.Push(
				luau.Create(&luau.WhileStatement{
					Condition:  luau.Bool(true),
					Statements: body.Statements,
				}),
			)
			if doneId != nil {
				prereqs.Push(
					luau.Create(&luau.IfStatement{
						Condition:  luau.Unary("not", doneId),
						Statements: restPrereqs.Statements,
						ElseBody:   luau.MakeList(),
					}),
				)
			} else {
				prereqs.PushList(restPrereqs.Statements)
			}
			return rest
		},
	}
}

func createFunctionAccessor(iterator luau.AnyIdentifier, tuple bool) BindingAccessor {
	return createIteratorAccessor(func(prereqs *Prereqs, omitted bool) (luau.Expression, luau.Expression) {
		call := luau.Call(iterator)
		var callValue luau.Expression = call
		if tuple && !omitted {
			callValue = luau.NewArray(call)
		}
		value := prereqs.PushToVar(callValue, "value")
		var first luau.Expression = value
		if tuple && !omitted {
			first = luau.Create(&luau.ComputedIndexExpression{Expression: value, Index: luau.Number(1)})
		}
		done := luau.Binary(first, "==", luau.Nil())
		return value, done
	}, tuple)
}

func createCollectionAccessor(
	prereqs *Prereqs,
	parentId luau.AnyIdentifier,
	isMap bool,
	elementCount int,
) BindingAccessor {
	key := prereqs.PushToVar(nil, "key")
	var consumed luau.AnyIdentifier
	accessor := createIteratorAccessor(func(prereqs *Prereqs, omitted bool) (luau.Expression, luau.Expression) {
		var member *luau.TemporaryIdentifier
		if isMap && !omitted {
			member = prereqs.PushToVar(nil, "value")
		}
		advance := func() luau.Statement {
			var left luau.ListOrNode = key
			if member != nil {
				left = luau.MakeList(key, member)
			}
			return luau.Create(&luau.Assignment{
				Left:     left,
				Operator: "=",
				Right:    luau.Call(luau.Globals.Next, parentId, key),
			})
		}
		if consumed != nil {
			// destination evaluation can remove the cursor and invalidate next's continuation key
			prereqs.Push(
				luau.Create(&luau.IfStatement{
					Condition: luau.Binary(
						luau.Binary(key, "~=", luau.Nil()),
						"and",
						luau.Binary(
							luau.Create(&luau.ComputedIndexExpression{Expression: parentId, Index: key}),
							"==",
							luau.Nil(),
						),
					),
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{Left: key, Operator: "=", Right: luau.Nil()}),
					),
					ElseBody: luau.MakeList(),
				}),
			)
		}
		prereqs.Push(advance())
		if consumed != nil {
			// restarting must not return prefix elements that were already consumed
			prereqs.Push(
				luau.Create(&luau.WhileStatement{
					Condition: luau.Binary(
						luau.Binary(key, "~=", luau.Nil()),
						"and",
						luau.Create(&luau.ComputedIndexExpression{Expression: consumed, Index: key}),
					),
					Statements: luau.MakeList(advance()),
				}),
			)
		}
		var value luau.Expression = key
		if member != nil {
			value = luau.NewArray(key, member)
		}
		return value, luau.Binary(key, "==", luau.Nil())
	}, isMap)

	return &bindingAccessor{
		read: func(prereqs *Prereqs, index int, omitted bool) luau.Expression {
			value := accessor.Read(prereqs, index, omitted)
			if index == elementCount-1 {
				return value
			}

			if consumed == nil {
				consumed = prereqs.PushToVar(luau.NewArray(), "consumed")
			}
			prereqs.Push(
				luau.Create(&luau.IfStatement{
					Condition: luau.Binary(key, "~=", luau.Nil()),
					Statements: luau.MakeList(
						luau.Create(&luau.Assignment{
							Left: luau.Create(&luau.ComputedIndexExpression{
								Expression: consumed,
								Index:      key,
							}),
							Operator: "=",
							Right:    luau.Bool(true),
						}),
					),
					ElseBody: luau.MakeList(),
				}),
			)
			return value
		},
		rest: accessor.Rest,
	}
}

// noneBindingAccessor is `{ read: () => luau.none(), rest: () => luau.none() }`.
func noneBindingAccessor() BindingAccessor {
	return &bindingAccessor{
		read: func(*Prereqs, int, bool) luau.Expression { return luau.NewNone() },
		rest: func(*Prereqs, int) luau.Expression { return luau.NewNone() },
	}
}

// createArrayBindingAccessor is TSTransformer/util/binding/createArrayBindingAccessor.ts createArrayBindingAccessor.
func createArrayBindingAccessor(state *TransformState, prereqs *Prereqs, node *ast.Node, type_ *checker.Type, parentId luau.AnyIdentifier) BindingAccessor {
	if isDefinitelyType(type_, isArrayType(state)) {
		return &bindingAccessor{
			read: func(prereqs *Prereqs, index int, omitted bool) luau.Expression {
				return luau.Create(&luau.ComputedIndexExpression{
					Expression: parentId,
					Index:      luau.Number(float64(index + 1)),
				})
			},
			rest: func(prereqs *Prereqs, index int) luau.Expression {
				return luau.Call(luau.Globals.Table.Move,
					parentId,
					luau.Number(float64(index+1)),
					luau.Unary("#", parentId),
					luau.Number(1),
					luau.NewArray(),
				)
			},
		}
	}

	if isDefinitelyType(type_, isStringType) {
		matcher := prereqs.PushToVar(
			luau.Call(luau.Globals.String.Gmatch, parentId, luau.Globals.Utf8.Charpattern),
			"matcher",
		)
		return &bindingAccessor{
			read: func(prereqs *Prereqs, index int, omitted bool) luau.Expression {
				value := luau.Call(matcher)
				if omitted {
					prereqs.Push(luau.Create(&luau.CallStatement{Expression: value}))
					return luau.NewNone()
				}
				return value
			},
			rest: func(prereqs *Prereqs, index int) luau.Expression {
				rest := prereqs.PushToVar(luau.NewArray(), "rest")
				char := luau.TempId("char")
				prereqs.Push(
					luau.Create(&luau.ForStatement{
						Ids:        luau.MakeList(char),
						Expression: matcher,
						Statements: luau.MakeList(
							luau.Create(&luau.CallStatement{
								Expression: luau.Call(luau.Globals.Table.Insert, rest, char),
							}),
						),
					}),
				)
				return rest
			},
		}
	}

	if isDefinitelyType(type_, isSetType(state)) || isDefinitelyType(type_, isMapType(state)) {
		return createCollectionAccessor(
			prereqs,
			parentId,
			isDefinitelyType(type_, isMapType(state)),
			len(node.Elements()),
		)
	}

	if isDefinitelyType(type_, isSharedTableType(state)) {
		// SharedTable supports generalized iteration, but not next()
		source := prereqs.PushToVar(parentId, "iterable")
		key := luau.TempId("key")
		value := luau.TempId("value")
		iterator := prereqs.PushToVar(
			luau.Call(luau.Property(luau.Id("coroutine"), "wrap"),
				luau.Create(&luau.FunctionExpression{
					Parameters:   luau.MakeList(),
					HasDotDotDot: false,
					Statements: luau.MakeList(
						luau.Create(&luau.ForStatement{
							Ids:        luau.MakeList(key, value),
							Expression: source,
							Statements: luau.MakeList(
								luau.Create(&luau.CallStatement{
									Expression: luau.Call(luau.Globals.Coroutine.Yield, key, value),
								}),
							),
						}),
					),
				}),
			),
			"iterator",
		)
		return createFunctionAccessor(iterator, true)
	}

	if isDefinitelyType(type_, isIterableFunctionType(state)) {
		return createFunctionAccessor(parentId, isDefinitelyType(type_, isIterableFunctionLuaTupleType(state)))
	}

	if isDefinitelyType(type_, func(t *checker.Type) bool {
		next := state.TypeChecker.GetTypeOfPropertyOfType(t, "next")
		return next != nil && len(typeGetCallSignatures(next)) > 0
	}) {
		next := prereqs.PushToVar(luau.Property(parentId, "next"), "next")
		return createIteratorAccessor(func(prereqs *Prereqs, omitted bool) (luau.Expression, luau.Expression) {
			result := prereqs.PushToVar(luau.Call(next), "result")
			done := luau.Property(result, "done")
			return luau.Property(result, "value"), done
		}, true)
	}

	if isDefinitelyType(type_, isIterableType(state)) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIterableIteration.Create(node))
		return noneBindingAccessor()
	}

	DiagnosticService.AddDiagnostic(diagnostics.Errors.NoUnsupportedIteration.Create(node))
	return noneBindingAccessor()
}
