package transform

// Port of TSTransformer/nodes/expressions/transformTemplateExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTemplateExpression is TSTransformer/nodes/expressions/transformTemplateExpression.ts transformTemplateExpression.
func transformTemplateExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.InterpolatedString {
	template := node.AsTemplateExpression()
	parts := luau.MakeList()

	if len(template.Head.Text()) > 0 {
		parts.Push(transformInterpolatedStringPart(template.Head))
	}

	templateSpans := template.TemplateSpans.Nodes
	spanExpressions := make([]*ast.Node, len(templateSpans))
	for i, templateSpan := range templateSpans {
		spanExpressions[i] = templateSpan.AsTemplateSpan().Expression
	}
	orderedExpressions := ensureTransformOrder(state, prereqs, spanExpressions, transformExpression)

	for i := range templateSpans {
		parts.Push(orderedExpressions[i])

		templateSpan := templateSpans[i].AsTemplateSpan()
		if len(templateSpan.Literal.Text()) > 0 {
			parts.Push(transformInterpolatedStringPart(templateSpan.Literal))
		}
	}

	return luau.Create(&luau.InterpolatedString{Parts: parts})
}
