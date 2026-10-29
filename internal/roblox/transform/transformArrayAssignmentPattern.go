package transform

// Port of TSTransformer/nodes/binding/transformArrayAssignmentPattern.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformArrayAssignmentPattern is TSTransformer/nodes/binding/transformArrayAssignmentPattern.ts transformArrayAssignmentPattern.
func transformArrayAssignmentPattern(state *TransformState, prereqs *Prereqs, assignmentPattern *ast.Node, parentId luau.AnyIdentifier) {
	type_ := getAssignmentPatternType(state, assignmentPattern)
	accessor := createArrayBindingAccessor(state, prereqs, assignmentPattern, type_, parentId)
	for index, element := range assignmentPattern.Elements() {
		if ast.IsOmittedExpression(element) {
			accessor.Read(prereqs, index, true)
			continue
		}

		valuePrereqs := NewPrereqs()
		spread := ast.IsSpreadElement(element)
		var value luau.Expression
		target := element
		if spread {
			value = accessor.Rest(valuePrereqs, index)
			target = element.Expression()
		} else {
			value = accessor.Read(valuePrereqs, index, false)
		}
		transformAssignmentPatternElement(state, prereqs, target, valuePrereqs, value, nil)
	}
}
