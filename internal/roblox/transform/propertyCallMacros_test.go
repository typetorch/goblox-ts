package transform

import (
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// The expected Luau of these tests is the output of roblox-ts 68648b0 for the same statements, with
// arguments that the macro receives already transformed (identifiers stay identifiers).

// parseCallExpression parses one statement and returns it and the first CallExpression or
// NewExpression in it. The parser sets parents, which isUsedAsStatement walks.
func parseCallExpression(t *testing.T, source string) *ast.Node {
	t.Helper()
	sourceFile := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/test.ts", Path: "/test.ts"}, source, core.ScriptKindTS)
	var find func(node *ast.Node) *ast.Node
	find = func(node *ast.Node) *ast.Node {
		if ast.IsCallExpression(node) || ast.IsNewExpression(node) {
			return node
		}
		var result *ast.Node
		node.ForEachChild(func(child *ast.Node) bool {
			result = find(child)
			return result != nil
		})
		return result
	}
	call := find(sourceFile.AsNode())
	if call == nil {
		t.Fatalf("no call expression in %q", source)
	}
	return call
}

// renderMacroResult renders the prereqs of a macro followed by `local x = result` unless the result
// is luau.none().
func renderMacroResult(prereqs *Prereqs, result luau.Expression) string {
	statements := prereqs.Statements
	if !luau.IsNone(result) {
		statements.Push(luau.Create(&luau.VariableDeclaration{Left: luau.Id("x"), Right: result}))
	}
	return luau.RenderAST(statements)
}

// tabs converts the four space indentation of an expected block to the tabs of the renderer.
func testTabs(text string) string {
	lines := strings.Split(strings.TrimPrefix(text, "\n"), "\n")
	for i, line := range lines {
		indent := 0
		for strings.HasPrefix(line, "    ") {
			line = line[4:]
			indent++
		}
		lines[i] = strings.Repeat("\t", indent) + line
	}
	return strings.Join(lines, "\n")
}

func testId(name string) luau.Expression { return luau.Id(name) }

func TestPropertyCallMacros(t *testing.T) {
	t.Parallel()
	arr, cb := testId("arr"), testId("cb")
	cases := []struct {
		source    string
		className string
		method    string
		args      []luau.Expression
		want      string
	}{
		{"x = arr.size()", "ArrayLike", "size", nil, "local x = #arr\n"},
		{"x = arr.isEmpty()", "ReadonlyArray", "isEmpty", nil, "local x = #arr == 0\n"},
		{"x = arr.move(0, 1, 2)", "ReadonlyArray", "move", []luau.Expression{luau.Number(0), luau.Number(1), luau.Number(2)}, "local x = table.move(arr, 1, 2, 3)\n"},
		{"x = arr.move(0, 1, 2, arr)", "ReadonlyArray", "move", []luau.Expression{luau.Number(0), luau.Number(1), luau.Number(2), testId("arr")}, "local x = table.move(arr, 1, 2, 3, arr)\n"},
		{"x = arr.includes(5)", "ReadonlyArray", "includes", []luau.Expression{luau.Number(5)}, "local x = table.find(arr, 5) ~= nil\n"},
		{"x = arr.includes(5, n)", "ReadonlyArray", "includes", []luau.Expression{luau.Number(5), testId("n")}, "local x = table.find(arr, 5, n + 1) ~= nil\n"},
		{"x = arr.indexOf(5)", "ReadonlyArray", "indexOf", []luau.Expression{luau.Number(5)}, "local x = (table.find(arr, 5) or 0) - 1\n"},
		{"x = arr.indexOf(5, 2)", "ReadonlyArray", "indexOf", []luau.Expression{luau.Number(5), luau.Number(2)}, "local x = (table.find(arr, 5, 3) or 0) - 1\n"},
		{"x = arr.every(cb)", "ReadonlyArray", "every", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.every ▼
local _result = true
for _k, _v in arr do
    if not cb(_v, _k - 1, arr) then
        _result = false
        break
    end
end
-- ▲ ReadonlyArray.every ▲
local x = _result
`},
		{"x = arr.some(cb)", "ReadonlyArray", "some", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.some ▼
local _result = false
for _k, _v in arr do
    if cb(_v, _k - 1, arr) then
        _result = true
        break
    end
end
-- ▲ ReadonlyArray.some ▲
local x = _result
`},
		{"arr.forEach(cb);", "ReadonlyArray", "forEach", []luau.Expression{cb}, `
for _k, _v in arr do
    cb(_v, _k - 1, arr)
end
`},
		{"x = arr.map(cb)", "ReadonlyArray", "map", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.map ▼
local _newValue = table.create(#arr)
for _k, _v in arr do
    _newValue[_k] = cb(_v, _k - 1, arr)
end
-- ▲ ReadonlyArray.map ▲
local x = _newValue
`},
		{"x = arr.mapFiltered(cb)", "ReadonlyArray", "mapFiltered", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.mapFiltered ▼
local _newValue = {}
local _length = 0
for _k, _v in arr do
    local _result = cb(_v, _k - 1, arr)
    if _result ~= nil then
        _length += 1
        _newValue[_length] = _result
    end
end
-- ▲ ReadonlyArray.mapFiltered ▲
local x = _newValue
`},
		{"x = arr.filterUndefined()", "ReadonlyArray", "filterUndefined", nil, `
-- ▼ ReadonlyArray.filterUndefined ▼
local _length = 0
for _i in arr do
    if _i > _length then
        _length = _i
    end
end
local _result = {}
local _resultLength = 0
for _i = 1, _length do
    local _v = arr[_i]
    if _v ~= nil then
        _resultLength += 1
        _result[_resultLength] = _v
    end
end
-- ▲ ReadonlyArray.filterUndefined ▲
local x = _result
`},
		{"x = arr.filter(cb)", "ReadonlyArray", "filter", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.filter ▼
local _newValue = {}
local _length = 0
for _k, _v in arr do
    if cb(_v, _k - 1, arr) == true then
        _length += 1
        _newValue[_length] = _v
    end
end
-- ▲ ReadonlyArray.filter ▲
local x = _newValue
`},
		{"x = arr.reduce(cb)", "ReadonlyArray", "reduce", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.reduce ▼
if #arr == 0 then
    error("Attempted to call ` + "`ReadonlyArray.reduce()`" + ` on an empty array without an initialValue.")
end
local _result = arr[1]
for _i = 2, #arr do
    _result = cb(_result, arr[_i], _i - 1, arr)
end
-- ▲ ReadonlyArray.reduce ▲
local x = _result
`},
		{"x = arr.reduce(cb, 10)", "ReadonlyArray", "reduce", []luau.Expression{cb, luau.Number(10)}, `
-- ▼ ReadonlyArray.reduce ▼
local _result = 10
for _i = 1, #arr do
    _result = cb(_result, arr[_i], _i - 1, arr)
end
-- ▲ ReadonlyArray.reduce ▲
local x = _result
`},
		{"x = arr.find(cb)", "ReadonlyArray", "find", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.find ▼
local _result
for _i, _v in arr do
    if cb(_v, _i - 1, arr) == true then
        _result = _v
        break
    end
end
-- ▲ ReadonlyArray.find ▲
local x = _result
`},
		{"x = arr.findIndex(cb)", "ReadonlyArray", "findIndex", []luau.Expression{cb}, `
-- ▼ ReadonlyArray.findIndex ▼
local _result = -1
for _i, _v in arr do
    if cb(_v, _i - 1, arr) == true then
        _result = _i - 1
        break
    end
end
-- ▲ ReadonlyArray.findIndex ▲
local x = _result
`},
		{"arr.push(1, 2);", "Array", "push", []luau.Expression{luau.Number(1), luau.Number(2)}, `
-- ▼ Array.push ▼
table.insert(arr, 1)
table.insert(arr, 2)
-- ▲ Array.push ▲
`},
		{"x = arr.push(3)", "Array", "push", []luau.Expression{luau.Number(3)}, "table.insert(arr, 3)\nlocal x = #arr\n"},
		{"x = arr.push()", "Array", "push", nil, "local x = #arr\n"},
		{"arr.pop();", "Array", "pop", nil, "arr[#arr] = nil\n"},
		{"x = arr.pop()", "Array", "pop", nil, `
-- ▼ Array.pop ▼
local _length = #arr
local _result = arr[_length]
arr[_length] = nil
-- ▲ Array.pop ▲
local x = _result
`},
		{"x = arr.shift()", "Array", "shift", nil, "local x = table.remove(arr, 1)\n"},
		{"arr.unshift(1, 2);", "Array", "unshift", []luau.Expression{luau.Number(1), luau.Number(2)}, `
-- ▼ Array.unshift ▼
table.insert(arr, 1, 2)
table.insert(arr, 1, 1)
-- ▲ Array.unshift ▲
`},
		{"x = arr.unshift(0)", "Array", "unshift", []luau.Expression{luau.Number(0)}, "table.insert(arr, 1, 0)\nlocal x = #arr\n"},
		{"x = arr.remove(n)", "Array", "remove", []luau.Expression{testId("n")}, "local x = table.remove(arr, n + 1)\n"},
		{"arr.unorderedRemove(n + 1);", "Array", "unorderedRemove", []luau.Expression{luau.Binary(testId("n"), "+", luau.Number(1))}, `
-- ▼ Array.unorderedRemove ▼
local _index = n + 2
local _length = #arr
local _value = arr[_index]
if _value ~= nil then
    arr[_index] = arr[_length]
    arr[_length] = nil
end
-- ▲ Array.unorderedRemove ▲
`},
		{"x = arr.unorderedRemove(2)", "Array", "unorderedRemove", []luau.Expression{luau.Number(2)}, `
-- ▼ Array.unorderedRemove ▼
local _length = #arr
local _value = arr[3]
if _value ~= nil then
    arr[3] = arr[_length]
    arr[_length] = nil
end
-- ▲ Array.unorderedRemove ▲
local x = _value
`},
		{"arr.sort();", "Array", "sort", nil, "table.sort(arr)\n"},
		{"x = arr.sort(cb)", "Array", "sort", []luau.Expression{cb}, "table.sort(arr, cb)\nlocal x = arr\n"},
		{"arr.clear();", "Array", "clear", nil, "table.clear(arr)\n"},
		{"x = s.isEmpty()", "ReadonlySet", "isEmpty", nil, "local x = next(s) == nil\n"},
		{"x = s.size()", "ReadonlySet", "size", nil, `
-- ▼ ReadonlySet.size ▼
local _size = 0
for _ in s do
    _size += 1
end
-- ▲ ReadonlySet.size ▲
local x = _size
`},
		{`x = s.has("x")`, "ReadonlySet", "has", []luau.Expression{luau.String("x")}, "local x = s.x ~= nil\n"},
		{`s.delete("x");`, "Set", "delete", []luau.Expression{luau.String("x")}, "s.x = nil\n"},
		{`x = s.delete("y")`, "Set", "delete", []luau.Expression{luau.String("y")}, `
-- ▼ Set.delete ▼
local _valueExisted = s.y ~= nil
s.y = nil
-- ▲ Set.delete ▲
local x = _valueExisted
`},
		{"s.clear();", "Set", "clear", nil, "table.clear(s)\n"},
		{"s.forEach(cb);", "ReadonlySet", "forEach", []luau.Expression{cb}, "for _v in s do\n\tcb(_v, _v, s)\nend\n"},
		{`s.add("z");`, "Set", "add", []luau.Expression{luau.String("z")}, "s.z = true\n"},
		{`x = s.add("w")`, "Set", "add", []luau.Expression{luau.String("w")}, "s.w = true\nlocal x = s\n"},
		{`x = m.get("a")`, "ReadonlyMap", "get", []luau.Expression{luau.String("a")}, "local x = m.a\n"},
		{`m.set("a", 1);`, "Map", "set", []luau.Expression{luau.String("a"), luau.Number(1)}, "m.a = 1\n"},
		{`x = m.set("b", 2)`, "Map", "set", []luau.Expression{luau.String("b"), luau.Number(2)}, "m.b = 2\nlocal x = m\n"},
		{"m.forEach(cb);", "ReadonlyMap", "forEach", []luau.Expression{cb}, "for _k, _v in m do\n\tcb(_v, _k, m)\nend\n"},
		{"x = str.size()", "String", "size", nil, "local x = #str\n"},
		{"x = str.sub(1, 2)", "String", "sub", []luau.Expression{luau.Number(1), luau.Number(2)}, "local x = string.sub(str, 1, 2)\n"},
		{"x = str.upper()", "String", "upper", nil, "local x = string.upper(str)\n"},
		{"x = v3.add(v3)", "Vector3", "add", []luau.Expression{testId("v3")}, "local x = v3 + v3\n"},
		{"x = v3.idiv(v3)", "Vector3", "idiv", []luau.Expression{testId("v3")}, "local x = v3 // v3\n"},
		{"x = n.idiv(2)", "Number", "idiv", []luau.Expression{luau.Number(2)}, "local x = n // 2\n"},
		{"x = p.then(cb)", "Promise", "then", []luau.Expression{cb}, "local x = p:andThen(cb)\n"},
		// insert returns the call itself, which transformExpressionStatement emits as `table.insert(arr, n + 1, 5)`
		{"arr.insert(n, 5);", "Array", "insert", []luau.Expression{testId("n"), luau.Number(5)}, "local x = table.insert(arr, n + 1, 5)\n"},
		{"x = m.size()", "ReadonlyMap", "size", nil, `
-- ▼ ReadonlyMap.size ▼
local _size = 0
for _ in m do
    _size += 1
end
-- ▲ ReadonlyMap.size ▲
local x = _size
`},
		// macros2.server.ts: the String macros returning LuaTuple are wrapped in { } by transformMacroCall, not the macro
		{`x = m.has("a")`, "ReadonlyMap", "has", []luau.Expression{luau.String("a")}, "local x = m.a ~= nil\n"},
		{"x = m.isEmpty()", "ReadonlyMap", "isEmpty", nil, "local x = next(m) == nil\n"},
		{`m.delete("a");`, "Map", "delete", []luau.Expression{luau.String("a")}, "m.a = nil\n"},
		{`x = m.delete("b")`, "Map", "delete", []luau.Expression{luau.String("b")}, `
-- ▼ Map.delete ▼
local _valueExisted = m.b ~= nil
m.b = nil
-- ▲ Map.delete ▲
local x = _valueExisted
`},
		{"m.clear();", "Map", "clear", nil, "table.clear(m)\n"},
		{"x = str.byte(1, 2)", "String", "byte", []luau.Expression{luau.Number(1), luau.Number(2)}, "local x = string.byte(str, 1, 2)\n"},
		{`x = str.find("a", 1, true)`, "String", "find", []luau.Expression{luau.String("a"), luau.Number(1), luau.Bool(true)}, "local x = string.find(str, \"a\", 1, true)\n"},
		{"x = str.format(1)", "String", "format", []luau.Expression{luau.Number(1)}, "local x = string.format(str, 1)\n"},
		{`x = str.gmatch("a")`, "String", "gmatch", []luau.Expression{luau.String("a")}, "local x = string.gmatch(str, \"a\")\n"},
		{`x = str.gsub("a", "b")`, "String", "gsub", []luau.Expression{luau.String("a"), luau.String("b")}, "local x = string.gsub(str, \"a\", \"b\")\n"},
		{"x = str.lower()", "String", "lower", nil, "local x = string.lower(str)\n"},
		{`x = str.match("a")`, "String", "match", []luau.Expression{luau.String("a")}, "local x = string.match(str, \"a\")\n"},
		{"x = str.rep(3)", "String", "rep", []luau.Expression{luau.Number(3)}, "local x = string.rep(str, 3)\n"},
		{"x = str.reverse()", "String", "reverse", nil, "local x = string.reverse(str)\n"},
		{`x = str.split(",")`, "String", "split", []luau.Expression{luau.String(",")}, "local x = string.split(str, \",\")\n"},
		{"x = v3.sub(v3)", "Vector3", "sub", []luau.Expression{testId("v3")}, "local x = v3 - v3\n"},
		{"x = v3.mul(2)", "Vector3", "mul", []luau.Expression{luau.Number(2)}, "local x = v3 * 2\n"},
		{"x = v3.div(n)", "Vector3", "div", []luau.Expression{testId("n")}, "local x = v3 / n\n"},
		{"x = v2.add(v2)", "Vector2", "add", []luau.Expression{testId("v2")}, "local x = v2 + v2\n"},
		{"x = v2i.mul(v2i)", "Vector2int16", "mul", []luau.Expression{testId("v2i")}, "local x = v2i * v2i\n"},
		{"x = cf.mul(cf)", "CFrame", "mul", []luau.Expression{testId("cf")}, "local x = cf * cf\n"},
		{"x = cf.add(v3)", "CFrame", "add", []luau.Expression{testId("v3")}, "local x = cf + v3\n"},
		{"x = u.sub(u)", "UDim", "sub", []luau.Expression{testId("u")}, "local x = u - u\n"},
		{"x = u2.add(u2)", "UDim2", "add", []luau.Expression{testId("u2")}, "local x = u2 + u2\n"},
	}

	receivers := map[string]luau.Expression{"arr": arr, "s": testId("s"), "m": testId("m"), "str": testId("str"), "v3": testId("v3"), "n": testId("n"), "p": testId("p"), "v2": testId("v2"), "v2i": testId("v2i"), "cf": testId("cf"), "u": testId("u"), "u2": testId("u2")}
	for _, c := range cases {
		t.Run(c.source, func(t *testing.T) {
			t.Parallel()
			node := parseCallExpression(t, c.source)
			receiver := receivers[node.Expression().Expression().Text()]
			macro := PROPERTY_CALL_MACROS[c.className][c.method]
			if macro == nil {
				t.Fatalf("no macro %s.%s", c.className, c.method)
			}
			prereqs := NewPrereqs()
			result := macro(nil, prereqs, node, receiver, append([]luau.Expression(nil), c.args...))
			got := renderMacroResult(prereqs, result)
			if want := testTabs(c.want); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// TestPropertyCallMacrosTable checks that every class and method of the roblox-ts table exists.
func TestPropertyCallMacrosTable(t *testing.T) {
	t.Parallel()
	want := map[string]string{
		"CFrame":        "add mul sub",
		"UDim":          "add sub",
		"UDim2":         "add sub",
		"Vector2":       "add div idiv mul sub",
		"Vector2int16":  "add div mul sub",
		"Vector3":       "add div idiv mul sub",
		"Vector3int16":  "add div mul sub",
		"vector":        "add div idiv mul sub",
		"Number":        "idiv",
		"String":        "byte find format gmatch gsub lower match rep reverse size split sub upper",
		"ArrayLike":     "size",
		"ReadonlyArray": "every filter filterUndefined find findIndex forEach includes indexOf isEmpty join map mapFiltered move reduce some",
		"Array":         "clear insert pop push remove shift sort unorderedRemove unshift",
		"ReadonlySet":   "forEach has isEmpty size",
		"Set":           "add clear delete",
		"ReadonlyMap":   "forEach get has isEmpty size",
		"Map":           "clear delete set",
		"Promise":       "then",
	}
	if len(PROPERTY_CALL_MACROS) != len(want) {
		t.Errorf("got %d classes, want %d", len(PROPERTY_CALL_MACROS), len(want))
	}
	for className, methods := range want {
		got := make([]string, 0)
		for name := range PROPERTY_CALL_MACROS[className] {
			got = append(got, name)
		}
		slices.Sort(got)
		if strings.Join(got, " ") != methods {
			t.Errorf("%s: got %v, want %s", className, got, methods)
		}
	}
}

// skipIfNotPorted skips the test when it reaches a transformer that is still a todo stub, so tests
// of paths through other agents' files start running once those land. Use it with defer.
func skipIfNotPorted(t *testing.T) {
	if r := recover(); r != nil {
		if message, ok := r.(string); ok && strings.HasPrefix(message, "goblox: not ported yet: ") {
			t.Skip(message)
		}
		panic(r)
	}
}
