package transform

// Port of TSTransformer/nodes/expressions/transformYieldExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformYieldExpression is TSTransformer/nodes/expressions/transformYieldExpression.ts transformYieldExpression.
func transformYieldExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	yield := node.AsYieldExpression()
	if yield.Expression == nil {
		return luau.Call(luau.Globals.Coroutine.Yield)
	}

	expression := transformExpression(state, prereqs, yield.Expression)
	if yield.AsteriskToken != nil {
		// delegation keeps the original next function even if the iterator is later modified
		nextId := prereqs.PushToVar(luau.Property(convertToIndexableExpression(expression), "next"), "next")
		currentInputId := prereqs.PushToVar(nil, "currentInput")
		resultId := luau.TempId("result")

		finalizer := luau.MakeList(luau.Create(&luau.BreakStatement{}))
		var evaluated luau.Expression = luau.NewNone()

		if !isUsedAsStatement(node) {
			returnValue := prereqs.PushToVar(nil, "returnValue")
			finalizer.Unshift(
				luau.Create(&luau.Assignment{
					Left:     returnValue,
					Operator: "=",
					Right:    luau.Property(resultId, "value"),
				}),
			)
			evaluated = returnValue
		}

		yieldCall := luau.Call(luau.Globals.Coroutine.Yield, luau.Property(resultId, "value"))

		prereqs.Push(
			luau.Create(&luau.WhileStatement{
				Condition: luau.Bool(true),
				Statements: luau.MakeList(
					luau.Create(&luau.VariableDeclaration{
						Left:  resultId,
						Right: luau.Call(nextId, currentInputId),
					}),
					luau.Create(&luau.IfStatement{
						Condition:  luau.Property(resultId, "done"),
						Statements: finalizer,
						ElseBody:   luau.MakeList(),
					}),
					luau.Create(&luau.Assignment{
						Left:     currentInputId,
						Operator: "=",
						Right:    yieldCall,
					}),
				),
			}),
		)

		return evaluated
	} else {
		return luau.Call(luau.Globals.Coroutine.Yield, expression)
	}
}
