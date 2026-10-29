package project

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
)

func TestRemoveNonNullAccessParens(t *testing.T) {
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
