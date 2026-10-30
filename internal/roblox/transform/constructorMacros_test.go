package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// Expected Luau from roblox-ts 68648b0. The cases with arguments go through ensureTransformOrder and
// transformExpression and are covered by the fixture harness instead.
func TestConstructorMacros(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source    string
		className string
		want      string
	}{
		{"x = new Array<number>()", "ArrayConstructor", "local x = {}\n"},
		{"x = new Set<string>()", "SetConstructor", "local x = {}\n"},
		{"x = new Map<string, number>()", "MapConstructor", "local x = {}\n"},
		{"x = new ReadonlySet<string>()", "ReadonlySetConstructor", "local x = {}\n"},
		{"x = new ReadonlyMap<string, number>()", "ReadonlyMapConstructor", "local x = {}\n"},
		{"x = new WeakSet<object>()", "WeakSetConstructor", "local x = setmetatable({}, {\n\t__mode = \"k\",\n})\n"},
		{"x = new WeakMap<object, number>()", "WeakMapConstructor", "local x = setmetatable({}, {\n\t__mode = \"k\",\n})\n"},
	}
	if len(CONSTRUCTOR_MACROS) != len(cases) {
		t.Errorf("got %d constructor macros, want %d", len(CONSTRUCTOR_MACROS), len(cases))
	}
	for _, c := range cases {
		t.Run(c.source, func(t *testing.T) {
			t.Parallel()
			node := parseCallExpression(t, c.source)
			prereqs := NewPrereqs()
			result := CONSTRUCTOR_MACROS[c.className](nil, prereqs, node)
			if got := renderMacroResult(prereqs, result); got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

func TestCallMacros(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source string
		name   string
		args   []luau.Expression
		want   string
	}{
		{`x = typeOf(v)`, "typeOf", []luau.Expression{testId("v")}, "local x = typeof(v)\n"},
		{`x = typeIs(v, "number")`, "typeIs", []luau.Expression{testId("v"), luau.String("number")}, "local x = type(v) == \"number\"\n"},
		{`x = typeIs(v, "Vector3")`, "typeIs", []luau.Expression{testId("v"), luau.String("Vector3")}, "local x = typeof(v) == \"Vector3\"\n"},
		{`x = classIs(v, "Part")`, "classIs", []luau.Expression{testId("v"), luau.String("Part")}, "local x = v.ClassName == \"Part\"\n"},
		{`x = identity(5)`, "identity", []luau.Expression{luau.Number(5)}, "local x = 5\n"},
	}
	for _, c := range cases {
		t.Run(c.source, func(t *testing.T) {
			t.Parallel()
			node := parseCallExpression(t, c.source)
			prereqs := NewPrereqs()
			result := CALL_MACROS[c.name](nil, prereqs, node, testId(c.name), c.args)
			if got := renderMacroResult(prereqs, result); got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}

// Expected Luau from roblox-ts 68648b0.
func TestConstructorMacrosWithArguments(t *testing.T) {
	t.Parallel()
	source := `
interface Array<T> { [n: number]: T }
interface Boolean {}
interface Function {}
interface IArguments {}
interface Number {}
interface Object {}
interface RegExp {}
interface String {}
declare function f(): number;
export const h1 = new Array<number>(5);
export const h2 = new Set(["a", "b"]);
export const h4 = new Map([["a", 1]]);
export const h11 = new Array<number>(f());
`
	state, _ := checkedCalls(t, source)
	var news []*ast.Node
	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		if ast.IsNewExpression(node) {
			news = append(news, node)
		}
		node.ForEachChild(visit)
		return false
	}
	state.SourceFile.AsNode().ForEachChild(visit)

	cases := []struct {
		className string
		want      string
	}{
		{"ArrayConstructor", "local x = table.create(5)\n"},
		{"SetConstructor", "local x = {\n\ta = true,\n\tb = true,\n}\n"},
		{"MapConstructor", "local x = {\n\ta = 1,\n}\n"},
		{"ArrayConstructor", "local x = table.create(f())\n"},
	}
	if len(news) != len(cases) {
		t.Fatalf("got %d new expressions, want %d", len(news), len(cases))
	}
	defer skipIfNotPorted(t)
	for i, c := range cases {
		if i == 3 {
			// f() goes through transformCallExpression, which needs a full TransformState (MacroManager,
			// Roblox types) that this noLib program lacks; the fixtures cover it (agent 5).
			t.Skip("case 3 needs a full TransformState for transformCallExpression")
		}
		prereqs := NewPrereqs()
		result := CONSTRUCTOR_MACROS[c.className](state, prereqs, news[i])
		if got := renderMacroResult(prereqs, result); got != c.want {
			t.Errorf("case %d: got:\n%s\nwant:\n%s", i, got, c.want)
		}
	}
}
