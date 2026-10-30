package project

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func TestRemoveNonNullAccessParens(t *testing.T) {
	t.Parallel()
	const fileName = "/src/a.ts"
	parse := func(text string) *ast.SourceFile {
		return parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: fileName, Path: fileName}, text, core.ScriptKindTS)
	}
	// the source's own parens around `a.b!` in the second statement stay
	source := parse("a.b!.c;\n(a.b!).d;\nx![0] = 1;\nf!();\ng(a.b!);\n")
	printed := "(a.b!).c;\n(a.b!).d;\n(x!)[0] = 1;\n(f!)();\ng(a.b!);\n"
	want := "a.b!.c;\n(a.b!).d;\nx![0] = 1;\nf!();\ng(a.b!);\n"
	if got := removeNonNullAccessParens(fileName, source, printed); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	// trees that don't line up leave the text alone
	if got := removeNonNullAccessParens(fileName, source, "(a.b!).c;\n"); got != "(a.b!).c;\n" {
		t.Fatalf("mismatched trees changed the text: %q", got)
	}
}

func TestReprintKeepsBlockComments(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		text string
		want bool
	}{
		{"no comments", "const a = 1;\n", true},
		{"line comment", "// a\nconst a = 1;\n", true},
		{"single line", "/* a */ const a = 1;\n", true},
		{"unterminated", "/* a\n\tb", true},
		{"gutter", "/**\n * a\n * b\n */\nconst a = 1;\n", true},
		{"gutter with tabs", "function f() {\n\t/**\n\t * a\n\t */\n}\n", true},
		{"same indentation", "/*\n\ta\n\tb\n*/\n", true},
		{"spaces right of the first line", "if (a) {\n  /* a\n    b\n      c */\n}\n", true},
		{"tabs, different indentation", "/*\n\ta\n\t\tb\n*/\n", false},
		{"spaces left of the first line", "    /*\n  a\n      b\n    */\n", false},
		{"gutter on some lines", "/*\n\t* a\n\t\tb\n*/\n", false},
		{"after a string on the line", "const a = \"x\"; /*\n\ta\n\t\tb\n*/\n", false},
		{"nested opener", "const a = \"/*\"; /* b\n\tc\n\t\td */\n", false},
		{"line separator", "/* a \tb\n\tc */\n", false},
		{"unicode whitespace", "/*\n a\n b\n*/\n", false},
		{"crlf gutter", "/**\r\n * a\r\n */\r\n", true},
	} {
		if got := reprintKeepsBlockComments(test.text); got != test.want {
			t.Errorf("%s: got %v, want %v", test.name, got, test.want)
		}
	}
}

func TestCalculateIndent(t *testing.T) {
	t.Parallel()
	for text, want := range map[string]int{"": 0, "a": 0, "  a": 2, "\ta": 4, "  \ta": 4, "\t  a": 6, "     \ta": 8} {
		if got := calculateIndent(text); got != want {
			t.Errorf("%q: got %d, want %d", text, got, want)
		}
	}
}
