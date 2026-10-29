package transform

// Port of TSTransformer/nodes/binding/transformBindingName.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformBindingName is TSTransformer/nodes/binding/transformBindingName.ts transformBindingName.
func transformBindingName(state *TransformState, name *ast.Node, initializers *luau.List) luau.AnyIdentifier {
	var id luau.AnyIdentifier
	if ast.IsIdentifier(name) {
		id = transformIdentifierDefined(state, name)
	} else {
		id = luau.TempId("binding")
		bindingPrereqs := NewPrereqs()
		if ast.IsArrayBindingPattern(name) {
			transformArrayBindingPattern(state, bindingPrereqs, name, id)
		} else {
			transformObjectBindingPattern(state, bindingPrereqs, name, id)
		}
		initializers.PushList(bindingPrereqs.Statements)
	}
	return id
}
