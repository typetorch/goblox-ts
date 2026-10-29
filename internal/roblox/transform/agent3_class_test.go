package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// parseClasses parses source and returns its class declarations in order.
func parseClasses(t *testing.T, source string) []*ast.Node {
	t.Helper()
	file := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts", Path: "/test.ts"}, source, core.ScriptKindTS)
	var classes []*ast.Node
	for _, statement := range file.Statements.Nodes {
		if ast.IsClassDeclaration(statement) {
			classes = append(classes, statement)
		}
	}
	return classes
}

func TestFindConstructor(t *testing.T) {
	classes := parseClasses(t, `
class A {}
class B {
	constructor(x: number);
	constructor(x?: number) {}
}
declare class C {
	constructor();
}
`)
	if got := findConstructor(classes[0]); got != nil {
		t.Errorf("A: expected no constructor, got %v", got.Kind)
	}
	got := findConstructor(classes[1])
	if got == nil || got.Body() == nil || len(got.Parameters()) != 1 || got.Parameters()[0].AsParameterDeclaration().QuestionToken == nil {
		t.Errorf("B: expected the constructor with a body")
	}
	if got := findConstructor(classes[2]); got != nil {
		t.Errorf("C: expected the bodiless constructor to be skipped")
	}
}

func TestGetExtendsNode(t *testing.T) {
	classes := parseClasses(t, `
class A {}
class B implements I extends A {}
class C extends A {}
class D implements I {}
`)
	for i, want := range []string{"", "A", "A", ""} {
		got := getExtendsNode(classes[i])
		switch {
		case want == "" && got != nil:
			t.Errorf("class %d: expected no extends node", i)
		case want != "" && (got == nil || got.Expression().Text() != want):
			t.Errorf("class %d: expected extends %s", i, want)
		}
	}
}

func TestCountDecorators(t *testing.T) {
	classes := parseClasses(t, `
@a @b class A {
	@c method(@d @e x: number, y: number) {}
}
`)
	if got := countDecorators(classes[0]); got != 2 {
		t.Errorf("class: expected 2 decorators, got %d", got)
	}
	method := classes[0].Members()[0]
	if got := countDecorators(method); got != 1 {
		t.Errorf("method: expected 1 decorator, got %d", got)
	}
	parameters := method.Parameters()
	if got := countDecorators(parameters[0]); got != 2 {
		t.Errorf("x: expected 2 decorators, got %d", got)
	}
	if got := countDecorators(parameters[1]); got != 0 {
		t.Errorf("y: expected 0 decorators, got %d", got)
	}
}

// Classes without instance property initializers never touch the TransformState in
// transformImplicitClassConstructor, so these render without a program.
func TestTransformImplicitClassConstructor(t *testing.T) {
	classes := parseClasses(t, `
class A {}
class B extends A {}
`)
	cases := []struct {
		node *ast.Node
		want string
	}{
		{classes[0], "function A:constructor()\nend\n"},
		{classes[1], "function B:constructor(...)\n\tsuper.constructor(self, ...)\nend\n"},
	}
	for _, c := range cases {
		name := luau.Id(c.node.Name().Text())
		got := luau.RenderAST(transformImplicitClassConstructor(nil, c.node, name))
		if got != c.want {
			t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
		}
	}
}
