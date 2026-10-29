package transform

// Port of TSTransformer/util/createStringFromLiteral.ts.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/scanner"
)

var (
	lineEndingPattern    = regexp.MustCompile(`\r\n?`)
	surrogatePairPattern = regexp.MustCompile(`\\\\|\\u([dD][89aAbB][\da-fA-F]{2})\\u([dD][c-fC-F][\da-fA-F]{2})`)
	escapePattern        = regexp.MustCompile(`\\(?:u([\da-fA-F]{4})|([\s\S]))`)
)

// createStringFromLiteral is TSTransformer/util/createStringFromLiteral.ts createStringFromLiteral.
// preserve source spelling except for escapes with different syntax or meaning in Luau
func createStringFromLiteral(node *ast.Node) string {
	// node.text decodes escapes, which could turn \b into \b and give it a different meaning in Luau
	source := scanner.GetTextOfNode(node)

	// every token starts with a quote, backtick, or }, and ends with one delimiter except heads and middles ending in ${
	end := len(source) - 1
	if ast.IsTemplateHead(node) || ast.IsTemplateMiddle(node) {
		end = len(source) - 2
	}
	text := source[1:end]

	// match TypeScript's normalization of physical line endings in templates, leaving written \r and \n escapes intact
	// normalize before removing line continuations so CR, CRLF, and LF all use the same check
	text = lineEndingPattern.ReplaceAllString(text, "\n")

	// emit one Unicode code point per surrogate pair for Luau, e.g. 😀 becomes \u{1f600}
	// combine high (D800-DBFF) and low (DC00-DFFF) surrogates before converting individual \uXXXX escapes
	text = surrogatePairPattern.ReplaceAllStringFunc(text, func(escape string) string {
		match := surrogatePairPattern.FindStringSubmatch(escape)
		// consume escaped backslashes first so literal text like \\uD83D\\uDE00 stays untouched
		if match[1] == "" {
			return escape
		}
		high, _ := strconv.ParseUint(match[1], 16, 32)
		low, _ := strconv.ParseUint(match[2], 16, 32)
		codePoint := 0x10000 + (high-0xD800)<<10 + (low - 0xDC00)
		return fmt.Sprintf("\\u{%x}", codePoint)
	})

	// consume whole escapes so \\u0041 stays literal text instead of matching a Unicode escape at the second backslash
	// [\s\S] includes physical line breaks so the same pass can remove line continuations
	text = escapePattern.ReplaceAllStringFunc(text, func(escape string) string {
		match := escapePattern.FindStringSubmatch(escape)
		hex, character := match[1], match[2]
		// add the braces Luau requires for Unicode escapes, preserving \b as \u{005c}b instead of decoding it to \b
		if hex != "" {
			return "\\u{" + hex + "}"
		}

		// a backslash followed by LF, a line separator, or a paragraph separator adds no characters in TypeScript
		if character == "\n" || character == " " || character == " " {
			return ""
		}

		// preserve shared escapes for Luau to interpret, including \xHH and existing or newly emitted \u{...} escapes
		// keep backticks escaped in templates because the emitted Luau template also uses backtick delimiters
		if strings.Contains("0bfnrtvux\\\"'", character) || (character == "`" && !ast.IsStringLiteral(node)) {
			return escape
		}

		// strip identity escape backslashes to match TypeScript, since Luau uses \a for a bell and \z to skip whitespace
		return character
	})

	return text
}
