package transform

// Port of TSTransformer/nodes/jsx/transformJsxChildren.ts.

import (
	"fmt"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformJsxChildren is TSTransformer/nodes/jsx/transformJsxChildren.ts transformJsxChildren.
func transformJsxChildren(state *TransformState, prereqs *Prereqs, children []*ast.Node) []luau.Expression {
	lastJsxChildIndex := -1
	for i := len(children) - 1; i >= 0; i-- {
		child := children[i]
		if !ast.IsJsxText(child) || !child.AsJsxText().ContainsOnlyTriviaWhiteSpaces {
			lastJsxChildIndex = i
			break
		}
	}

	for i := 0; i < lastJsxChildIndex; i++ {
		child := children[i]
		if ast.IsJsxExpression(child) && child.AsJsxExpression().DotDotDotToken != nil {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPrecedingJsxSpreadElement.Create(child))
		}
	}

	var filtered []*ast.Node
	for _, v := range children {
		// ignore jsx text that only contains whitespace
		if ast.IsJsxText(v) && v.AsJsxText().ContainsOnlyTriviaWhiteSpaces {
			continue
		}
		// ignore empty jsx expressions, i.e. `{}`
		if ast.IsJsxExpression(v) && v.AsJsxExpression().Expression == nil {
			continue
		}
		filtered = append(filtered, v)
	}

	return ensureTransformOrder(
		state,
		prereqs,
		filtered,
		func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			if ast.IsJsxText(node) {
				text, ok := fixupWhitespaceAndDecodeEntities(node.AsJsxText().Text)
				assert(ok)
				text = strings.ReplaceAll(text, `\`, `\\`)
				text = strings.ReplaceAll(text, `"`, `\"`)
				text = escapeJsxTextControlCharacters(text)
				// decoded text needs quoted escapes even when it contains both quote characters
				return luau.String(text, `"`)
			}
			return transformExpression(state, prereqs, node)
		},
	)
}

// escapeJsxTextControlCharacters replaces each character of /[\x00-\x1f\x7f]/ with a \xHH escape:
// decoded entities can contain literal control characters.
func escapeJsxTextControlCharacters(text string) string {
	var sb strings.Builder
	for i := 0; i < len(text); i++ {
		c := text[i]
		if c <= 0x1f || c == 0x7f {
			fmt.Fprintf(&sb, `\x%02x`, c)
		} else {
			sb.WriteByte(c)
		}
	}
	return sb.String()
}
