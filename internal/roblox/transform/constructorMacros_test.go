package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// Expected Luau from roblox-ts 68648b0. The cases with arguments go through ensureTransformOrder and
// transformExpression and are covered by the fixture harness instead.
func TestConstructorMacros(t *testing.T) {
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
			node := parseCallExpression(t, c.source)
			prereqs := NewPrereqs()
			result := CALL_MACROS[c.name](nil, prereqs, node, testId(c.name), c.args)
			if got := renderMacroResult(prereqs, result); got != c.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, c.want)
			}
		})
	}
}
