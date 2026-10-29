package transform

// Port of TSTransformer/nodes/binding/transformObjectAssignmentPattern.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformObjectAssignmentPattern is TSTransformer/nodes/binding/transformObjectAssignmentPattern.ts transformObjectAssignmentPattern.
func transformObjectAssignmentPattern(state *TransformState, prereqs *Prereqs, assignmentPattern *ast.Node, parentId luau.AnyIdentifier) {
	type_ := getAssignmentPatternType(state, assignmentPattern)
	var keys []luau.Expression
	for _, property := range assignmentPattern.Properties() {
		valuePrereqs := NewPrereqs()
		if ast.IsSpreadAssignment(property) {
			value := createObjectRest(state, valuePrereqs, property, type_, parentId, keys)
			transformAssignmentPatternElement(state, prereqs, property.Expression(), valuePrereqs, value, nil)
		} else {
			assert(
				ast.IsShorthandPropertyAssignment(property) || ast.IsPropertyAssignment(property),
				"Invalid object assignment property: "+getKindName(property.Kind),
			)
			// evaluate the property key before the destination, then read the value after it
			key, value := objectAccessor(state, prereqs, parentId, type_, property.Name())
			keys = append(keys, key)
			shorthand := ast.IsShorthandPropertyAssignment(property)
			if shorthand {
				transformAssignmentPatternElement(
					state,
					prereqs,
					property.Name(),
					valuePrereqs,
					value,
					property.AsShorthandPropertyAssignment().ObjectAssignmentInitializer,
				)
			} else {
				transformAssignmentPatternElement(
					state,
					prereqs,
					property.Initializer(),
					valuePrereqs,
					value,
					nil,
				)
			}
		}
	}
}
