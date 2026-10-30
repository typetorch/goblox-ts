package project

// Fidelity of the plugin reprint (RunPlugins). roblox-ts prints transformed files with TypeScript's
// printer, which parenthesizes through the factory's parenthesizer rules: `x!.y` stays `x!.y`, since
// a NonNullExpression is a LeftHandSideExpression. typescript-go's printer parenthesizes by precedence
// instead and ranks NonNullExpression as Unary (internal/ast/precedence.go), below Member, so it prints
// `(x!).y`. After the reparse roblox-ts' transformParenthesizedExpression keeps those parens around a
// non-simple expression: `(script.Parent):FindFirstChild(...)`. The printer is upstream code, so the
// added parens are taken out of the printed text here.

import (
	"strings"
	"unicode"

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

// reprintKeepsBlockComments is whether every block comment of text renders to the same Luau after a
// reprint (false when one may not; the file is then reprinted, as roblox-ts always does). The printer
// writes a comment's later lines with `max(0, writerIndent - firstLineIndent + lineIndent)` spaces, tabs
// counted up to the next tab stop, and getBlockCommentText then removes a gutter or the indentation the
// lines share, counted in characters. The result doesn't depend on the writer's indent when every line
// has a gutter, or the same indentation, or only spaces and none left of the comment's first line.
//
// Comments are found without a scanner: a `/*` in a string or a line comment only costs a reprint.
func reprintKeepsBlockComments(text string) bool {
	for pos := 0; ; {
		open := strings.Index(text[pos:], "/*")
		if open < 0 {
			return true
		}
		open += pos
		length := strings.Index(text[open+2:], "*/")
		if length < 0 {
			return true
		}
		body := text[open+2 : open+2+length]
		pos = open + 2 + length + 2
		// a `/*` in a string or a line comment must not hide a comment after it
		if strings.Contains(body, "/*") {
			return false
		}
		if !strings.ContainsAny(body, "\r\n") {
			continue
		}
		lineStart := strings.LastIndexAny(text[:open], "\r\n") + 1
		if strings.ContainsAny(text[lineStart:open], "\"'`/") || !blockCommentLinesKeep(calculateIndent(text[lineStart:open]), body) {
			return false
		}
	}
}

func blockCommentLinesKeep(firstLineIndent int, body string) bool {
	if strings.ContainsAny(body, "  ") {
		return false
	}
	lines := strings.FieldsFunc(body, func(r rune) bool { return r == '\r' || r == '\n' })
	// the first line follows the opening delimiter, so it has no indentation of its own
	if body[0] != '\r' && body[0] != '\n' {
		lines = lines[1:]
	}
	everyGutter, sameIndent, onlySpaces, leftOfFirstLine := true, true, true, false
	indent, seen := "", false
	for _, line := range lines {
		rest := strings.TrimLeft(line, " \t")
		lineIndent := line[:len(line)-len(rest)]
		// empty after the trim: includes the indentation of a closing delimiter on its own line
		if strings.TrimSpace(rest) == "" {
			continue
		}
		// the printer trims any whitespace, getBlockCommentText counts it: leave those to the reprint
		if first := []rune(rest)[0]; first > unicode.MaxASCII && (unicode.IsSpace(first) || first == 0xFEFF) {
			return false
		}
		everyGutter = everyGutter && rest[0] == '*'
		sameIndent = sameIndent && (!seen || lineIndent == indent)
		onlySpaces = onlySpaces && !strings.Contains(lineIndent, "\t")
		leftOfFirstLine = leftOfFirstLine || len(lineIndent) < firstLineIndent
		indent, seen = lineIndent, true
	}
	return everyGutter || sameIndent || (onlySpaces && !leftOfFirstLine)
}

// calculateIndent is the printer's calculateIndent: the width of the leading whitespace of line.
func calculateIndent(line string) int {
	indent := 0
	for _, ch := range line {
		switch ch {
		case '\t':
			indent += 4 - indent%4
		case ' ':
			indent++
		default:
			return indent
		}
	}
	return indent
}
