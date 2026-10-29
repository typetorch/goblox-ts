package transform

// Port of TSTransformer/nodes/expressions/transformArrayLiteralExpression.ts.

import (
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformArrayLiteralExpression is TSTransformer/nodes/expressions/transformArrayLiteralExpression.ts transformArrayLiteralExpression.
func transformArrayLiteralExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	elements := node.Elements()
	if !slices.ContainsFunc(elements, ast.IsSpreadElement) {
		return luau.NewArray(ensureTransformOrder(state, prereqs, elements, nil)...)
	}

	ptr := createArrayPointer("array")
	lengthId := luau.TempId("length")
	amtElementsSinceUpdate := 0

	// the array pointer becomes an identifier once, when inline construction ends
	initializeLengthId := func() {
		prereqs.Push(luau.Create(&luau.VariableDeclaration{
			Left:  lengthId,
			Right: luau.Unary("#", ptr.Value),
		}))
		amtElementsSinceUpdate = 0
	}

	for i, element := range elements {
		if ast.IsSpreadElement(element) {
			if luau.IsArray(ptr.Value) {
				disableArrayInline(prereqs, ptr)
				initializeLengthId()
			}
			assert(luau.IsAnyIdentifier(ptr.Value))

			type_ := state.GetType(element.Expression())
			addIterableToArrayBuilder := getAddIterableToArrayBuilder(state, element.Expression(), type_)
			spreadExp := transformExpression(state, prereqs, element.Expression())
			shouldUpdateLengthId := i < len(elements)-1
			prereqs.PushList(
				addIterableToArrayBuilder(
					prereqs,
					spreadExp,
					ptr.Value,
					lengthId,
					amtElementsSinceUpdate,
					shouldUpdateLengthId,
				),
			)
		} else {
			expressionPrereqs := NewPrereqs()
			expression := transformExpression(state, expressionPrereqs, element)
			if luau.IsArray(ptr.Value) && !expressionPrereqs.Statements.IsEmpty() {
				disableArrayInline(prereqs, ptr)
				initializeLengthId()
			}
			if array, ok := ptr.Value.(*luau.Array); ok {
				array.Members.Push(expression)
			} else {
				prereqs.PushList(expressionPrereqs.Statements)
				prereqs.Push(luau.Create(&luau.Assignment{
					Left: luau.Create(&luau.ComputedIndexExpression{
						Expression: ptr.Value,
						Index:      luau.Binary(lengthId, "+", luau.Number(float64(amtElementsSinceUpdate+1))),
					}),
					Operator: "=",
					Right:    expression,
				}))
			}
			amtElementsSinceUpdate++
		}
	}

	return ptr.Value
}
