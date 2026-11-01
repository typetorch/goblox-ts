package transform

import (
	"context"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/vfs/vfstest"
)

// checkedCalls type checks source with a minimal noLib global scope and returns a TransformState for
// it and its call expressions in source order.
func checkedCalls(t *testing.T, source string) (*TransformState, []*ast.Node) {
	t.Helper()
	fs := bundled.WrapFS(vfstest.FromMap(map[string]string{
		"/test.ts":       source,
		"/tsconfig.json": `{ "compilerOptions": { "noLib": true, "strict": true }, "files": ["test.ts"] }`,
	}, false))
	host := compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil, nil)
	config, errs := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, host, nil)
	if len(errs) > 0 {
		t.Fatalf("config errors: %v", errs)
	}
	program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
	sourceFile := program.GetSourceFile("/test.ts")
	c, done := program.GetTypeCheckerForFile(context.Background(), sourceFile)
	t.Cleanup(done)

	var calls []*ast.Node
	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		if ast.IsCallExpression(node) {
			calls = append(calls, node)
		}
		node.ForEachChild(visit)
		return false
	}
	sourceFile.AsNode().ForEachChild(visit)
	return &TransformState{TypeChecker: c, SourceFile: sourceFile, Data: &ProjectData{}, MultiTransformState: NewMultiTransformState(), getTypeCache: map[*ast.Node]*checker.Type{}}, calls
}

// Expected Luau from roblox-ts 68648b0.
func TestReadonlyArrayJoin(t *testing.T) {
	t.Parallel()
	state, calls := checkedCalls(t, `
interface Array<T> { [n: number]: T; join(separator?: string): string }
interface Boolean {}
interface Function {}
interface IArguments {}
interface Number {}
interface Object {}
interface RegExp {}
interface String {}
declare const sarr: Array<string>;
declare const barr: Array<boolean>;
declare const sep: string | undefined;
export const a3 = sarr.join();
export const a4 = barr.join("-");
export const a4b = sarr.join(sep);
export const a4c = sarr.join("");
`)
	join := PROPERTY_CALL_MACROS["ReadonlyArray"]["join"]
	cases := []struct {
		receiver string
		args     []luau.Expression
		want     string
	}{
		{"sarr", nil, "local x = table.concat(sarr, \", \")\n"},
		{"barr", []luau.Expression{luau.String("-")}, testTabs(`
-- ▼ ReadonlyArray.join ▼
local _result = table.create(#barr)
for _k, _v in barr do
    _result[_k] = tostring(_v)
end
-- ▲ ReadonlyArray.join ▲
local x = table.concat(_result, "-")
`)},
		{"sarr", []luau.Expression{testId("sep")}, "local x = table.concat(sarr, if sep == nil then \", \" else sep)\n"},
		{"sarr", []luau.Expression{luau.String("")}, "local x = table.concat(sarr, \"\")\n"},
	}
	if len(calls) != len(cases) {
		t.Fatalf("got %d calls, want %d", len(calls), len(cases))
	}
	for i, c := range cases {
		prereqs := NewPrereqs()
		result := join(state, prereqs, calls[i], testId(c.receiver), c.args)
		if got := renderMacroResult(prereqs, result); got != c.want {
			t.Errorf("case %d: got:\n%s\nwant:\n%s", i, got, c.want)
		}
	}
}
