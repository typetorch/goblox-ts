package transform

// Port of TSTransformer/nodes/binding/transformObjectBindingPattern.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformObjectBindingPattern is TSTransformer/nodes/binding/transformObjectBindingPattern.ts transformObjectBindingPattern.
func transformObjectBindingPattern(state *TransformState, prereqs *Prereqs, bindingPattern *ast.Node, parentId luau.AnyIdentifier) {
	validateNotAnyType(state, bindingPattern)
	var keys []luau.Expression
	type_ := state.GetType(bindingPattern)
	for _, element := range bindingPattern.Elements() {
		bindingElement := element.AsBindingElement()
		name := element.Name()
		prop := bindingElement.PropertyName
		isSpread := bindingElement.DotDotDotToken != nil

		if ast.IsIdentifier(name) {
			var value luau.Expression
			if isSpread {
				value = createObjectRest(state, prereqs, element, type_, parentId, keys)
			} else {
				accessName := prop
				if accessName == nil {
					accessName = name
				}
				key, accessValue := objectAccessor(state, prereqs, parentId, type_, accessName)
				keys = append(keys, key)
				value = accessValue
			}

			id := transformVariable(state, prereqs, name, value)
			if bindingElement.Initializer != nil {
				prereqs.Push(transformInitializer(state, id, bindingElement.Initializer))
			}
		} else {
			// if name is not identifier, it must be a binding pattern
			// in that case, prop is guaranteed to exist
			assert(prop != nil)
			assert(!isSpread)
			key, value := objectAccessor(state, prereqs, parentId, type_, prop)
			keys = append(keys, key)

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
}
