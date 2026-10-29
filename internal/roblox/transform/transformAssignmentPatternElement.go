package transform

// Port of TSTransformer/nodes/binding/transformAssignmentPatternElement.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformAssignmentPatternElement is TSTransformer/nodes/binding/transformAssignmentPatternElement.ts transformAssignmentPatternElement.
func transformAssignmentPatternElement(state *TransformState, prereqs *Prereqs, target *ast.Node, valuePrereqs *Prereqs, value luau.Expression, initializer *ast.Node) {
	panic(todo("nodes/binding/transformAssignmentPatternElement.ts", "transformAssignmentPatternElement"))
}
