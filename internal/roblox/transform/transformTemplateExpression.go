package transform

// Port of TSTransformer/nodes/expressions/transformTemplateExpression.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTemplateExpression is TSTransformer/nodes/expressions/transformTemplateExpression.ts transformTemplateExpression.
func transformTemplateExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.InterpolatedString {
	panic(todo("nodes/expressions/transformTemplateExpression.ts", "transformTemplateExpression"))
}
