package project

// Fidelity of the plugin reprint (RunPlugins). roblox-ts prints transformed files with TypeScript's
// printer, which parenthesizes through the factory's parenthesizer rules: `x!.y` stays `x!.y`, since
// a NonNullExpression is a LeftHandSideExpression. typescript-go's printer parenthesizes by precedence
// instead and ranks NonNullExpression as Unary (internal/ast/precedence.go), below Member, so it prints
// `(x!).y`. After the reparse roblox-ts' transformParenthesizedExpression keeps those parens around a
// non-simple expression: `(script.Parent):FindFirstChild(...)`. The printer is upstream code, so the
// added parens are taken out of the printed text here.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/scanner"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// nonNullAccessBases lists, in document order, the NonNullExpression bases of property accesses,
// element accesses and calls below root (`x!` in `x!.y`, `x![i]`, `x!()`), with the ParenthesizedExpression
// around one, or nil when it has none.
func nonNullAccessBases(root *ast.Node) []*ast.Node {
	var bases []*ast.Node
	var visit ast.Visitor
	visit = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindPropertyAccessExpression, ast.KindElementAccessExpression, ast.KindCallExpression:
			expression := node.Expression()
			if ast.IsNonNullExpression(expression) {
				bases = append(bases, nil)
			} else if ast.IsParenthesizedExpression(expression) && ast.IsNonNullExpression(expression.Expression()) {
				bases = append(bases, expression)
			}
		}
		node.ForEachChild(visit)
		return false
	}
	root.ForEachChild(visit)
	return bases
}

// removeNonNullAccessParens takes the parentheses typescript-go's printer added around NonNullExpression
// bases out of text, the printed form of transformed. Parentheses that are in transformed itself (the
// source had them, or a plugin made them) stay. When the two trees don't line up the text is returned
// unchanged.
func removeNonNullAccessParens(fileName string, transformed *ast.SourceFile, text string) string {
	want := nonNullAccessBases(transformed.AsNode())
	if len(want) == 0 {
		return text
	}
	printed := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: fileName, Path: tspath.Path(fileName)}, text, core.EnsureScriptKindFromFileName(fileName))
	got := nonNullAccessBases(printed.AsNode())
	if len(got) != len(want) {
		return text
	}
	var removals []int
	for i, paren := range got {
		if paren == nil || want[i] != nil {
			continue
		}
		open, close := scanner.SkipTrivia(text, paren.Pos()), paren.End()-1
		if open >= close || text[open] != '(' || text[close] != ')' {
			return text
		}
		removals = append(removals, open, close)
	}
	if len(removals) == 0 {
		return text
	}
	out := make([]byte, 0, len(text)-len(removals))
	removed := make(map[int]bool, len(removals))
	for _, pos := range removals {
		removed[pos] = true
	}
	for i := 0; i < len(text); i++ {
		if !removed[i] {
			out = append(out, text[i])
		}
	}
	return string(out)
}
