package transform

// Port of TSTransformer/nodes/expressions/transformTaggedTemplateExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformTaggedTemplateExpression is TSTransformer/nodes/expressions/transformTaggedTemplateExpression.ts transformTaggedTemplateExpression.
func transformTaggedTemplateExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) *luau.CallExpression {
	tagged := node.AsTaggedTemplateExpression()
	tagExp := transformExpression(state, prereqs, tagged.Tag)

	if ast.IsTemplateExpression(tagged.Template) {
		template := tagged.Template.AsTemplateExpression()
		strings := []luau.Expression{luau.String(template.Head.Text())}
		for _, templateSpan := range template.TemplateSpans.Nodes {
			strings = append(strings, luau.String(templateSpan.AsTemplateSpan().Literal.Text()))
		}

		spanExpressions := make([]*ast.Node, len(template.TemplateSpans.Nodes))
		for i, templateSpan := range template.TemplateSpans.Nodes {
			spanExpressions[i] = templateSpan.AsTemplateSpan().Expression
		}
		expressions := ensureTransformOrder(state, prereqs, spanExpressions, transformExpression)

		args := append([]luau.Expression{luau.NewArray(strings...)}, expressions...)
		return luau.Call(convertToIndexableExpression(tagExp), args...)
	} else {
		return luau.Call(convertToIndexableExpression(tagExp), luau.NewArray(luau.String(tagged.Template.Text())))
	}
}
