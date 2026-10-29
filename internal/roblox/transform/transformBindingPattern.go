package transform

// Port of TSTransformer/util/binding/transformBindingPattern.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformBindingPattern is TSTransformer/util/binding/transformBindingPattern.ts transformBindingPattern.
func transformBindingPattern(state *TransformState, prereqs *Prereqs, pattern *ast.Node, value luau.Expression) {
	reference := luau.TempId("binding")
	body := NewPrereqs()
	if ast.IsArrayBindingPattern(pattern) {
		transformArrayBindingPattern(state, body, pattern, reference)
	} else {
		transformObjectBindingPattern(state, body, pattern, reference)
	}

	// capture only when the emitted destructuring can change the source binding
	var source luau.Expression
	if luau.IsAnyIdentifier(value) && effectsCommute(getEffects(value), getStatementsEffects(body.Statements)) {
		source = value
	} else {
		source = prereqs.PushToVar(value, "binding")
	}
	replacements := map[int]luau.Expression{reference.Id: source}
	prereqs.PushList(luau.MakeList(luau.ListMapToArray(body.Statements, func(statement luau.Node) luau.Node {
		return substitute(statement, replacements)
	})...))
}
