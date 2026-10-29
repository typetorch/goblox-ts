package transform

// Port of TSTransformer/nodes/jsx/transformJsxTagName.ts.

import (
	"strings"
	"unicode/utf8"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func transformJsxTagNameExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	// host component
	if ast.IsIdentifier(node) {
		text := node.Text()
		firstChar, _ := utf8.DecodeRuneInString(text)
		// node.text[0] is one UTF-16 code unit: half of a surrogate pair is unchanged by toLowerCase
		if firstChar > 0xFFFF || strings.ToLower(string(firstChar)) == string(firstChar) {
			return luau.String(text)
		}
	}

	if ast.IsPropertyAccessExpression(node) {
		name := node.AsPropertyAccessExpression().Name()
		assert(ast.IsIdentifier(name))
		return luau.Property(
			convertToIndexableExpression(transformExpression(state, prereqs, node.Expression())),
			name.Text(),
		)
	} else if ast.IsJsxNamespacedName(node) {
		// Text() of a JsxNamespacedName is ts.getTextOfJsxNamespacedName
		return luau.String(node.Text())
	} else {
		return transformExpression(state, prereqs, node)
	}
}

// transformJsxTagName is TSTransformer/nodes/jsx/transformJsxTagName.ts transformJsxTagName.
func transformJsxTagName(state *TransformState, tagName *ast.Node) luau.Expression {
	expressionPrereqs := NewPrereqs()
	expression := transformJsxTagNameExpression(state, expressionPrereqs, tagName)
	// JSX tag names only allow identifiers, this, dotted names, and namespaced names
	assert(expressionPrereqs.Statements.IsEmpty())
	return expression
}
