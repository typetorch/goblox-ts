package transform

// Port of TSTransformer/util/getBlockCommentText.ts.

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	blockCommentLineEndingPattern = regexp.MustCompile(`\r\n?`)
	blockCommentGutterTestPattern = regexp.MustCompile(`^[ \t]*\*`)
	blockCommentGutterPattern     = regexp.MustCompile(`^[ \t]*\* ?`)
)

// getBlockCommentText is TSTransformer/util/getBlockCommentText.ts getBlockCommentText.
//
// convert a block comment to Luau comment text, which renders multiple lines in an indented --[[ ]] block
// so the source layout of the delimiters, gutter, and indentation is removed
func getBlockCommentText(source string) string {
	// the second * of a JSDoc comment's /** delimiter is not part of its text
	start := 2
	if strings.HasPrefix(source, "/**") {
		start = 3
	}
	end := len(source) - 2
	text := ""
	if start < end {
		text = source[start:end]
	}

	// match TypeScript's normalization of physical line endings
	text = blockCommentLineEndingPattern.ReplaceAllString(text, "\n")

	// trailing whitespace never matters in a comment, and removing it keeps blank lines empty
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = jsTrimEnd(line)
	}

	// text on the first line follows the opening delimiter, so only later lines have source indentation
	first := jsTrimStart(lines[0])
	rest := lines[1:]

	// remove a gutter that every line has, e.g. " * " in a JSDoc comment
	everyGutter := true
	for _, line := range rest {
		if !(line == "" || blockCommentGutterTestPattern.MatchString(line)) {
			everyGutter = false
			break
		}
	}
	if everyGutter {
		for i, line := range rest {
			rest[i] = blockCommentGutterPattern.ReplaceAllString(line, "")
		}
	}

	// remove the indentation shared by later lines, since the renderer indents them inside --[[ ]]
	// (JavaScript whitespace characters are all one UTF-16 code unit, so indentation is counted in runes)
	indent := -1
	for _, line := range rest {
		if line != "" {
			lineIndent := utf8.RuneCountInString(line) - utf8.RuneCountInString(jsTrimStart(line))
			if indent == -1 || lineIndent < indent {
				indent = lineIndent
			}
		}
	}
	for i, line := range rest {
		if indent == -1 {
			// line.slice(Infinity)
			rest[i] = ""
			continue
		}
		cut := 0
		for n := 0; n < indent && cut < len(line); n++ {
			_, size := utf8.DecodeRuneInString(line[cut:])
			cut += size
		}
		rest[i] = line[cut:]
	}

	// drop the lines left empty by delimiters on their own lines, e.g. /** and */
	result := append([]string{first}, rest...)
	if result[0] == "" {
		result = result[1:]
	}
	if len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}

	text = strings.Join(result, "\n")

	// one line renders as --text, so keep a space after the dashes to avoid starting --[[ or a --! directive
	if len(result) == 1 && text != "" {
		return " " + text
	}

	return text
}
