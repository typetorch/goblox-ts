package transform

// Port of TSTransformer/nodes/binding/transformArrayBindingPattern.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformArrayBindingPattern is TSTransformer/nodes/binding/transformArrayBindingPattern.ts transformArrayBindingPattern.
func transformArrayBindingPattern(state *TransformState, prereqs *Prereqs, bindingPattern *ast.Node, parentId luau.AnyIdentifier) {
	validateNotAnyType(state, bindingPattern)

	type_ := state.GetType(bindingPattern)
	index := 0
	accessor := createArrayBindingAccessor(state, prereqs, bindingPattern, type_, parentId)

	for _, element := range bindingPattern.Elements() {
		if isOmittedBindingElement(element) {
			accessor.Read(prereqs, index, true)
		} else {
			name := element.Name()
			bindingElement := element.AsBindingElement()

			isSpreadElement := bindingElement.DotDotDotToken != nil
			var value luau.Expression
			if isSpreadElement {
				value = accessor.Rest(prereqs, index)
			} else {
				value = accessor.Read(prereqs, index, false)
			}

			if ast.IsIdentifier(name) {
				id := transformVariable(state, prereqs, name, value)
				if bindingElement.Initializer != nil {
					prereqs.Push(transformInitializer(state, id, bindingElement.Initializer))
				}
			} else {
				id := prereqs.PushToVar(value, "binding")
				if bindingElement.Initializer != nil {
					prereqs.Push(transformInitializer(state, id, bindingElement.Initializer))
				}
				if ast.IsArrayBindingPattern(name) {
					transformArrayBindingPattern(state, prereqs, name, id)
				} else {
					transformObjectBindingPattern(state, prereqs, name, id)
				}
			}
		}
		index++
	}
}
