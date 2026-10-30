package transform

import (
	"context"
	"reflect"
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

// jsxTestPrelude stands in for @rbxts/compiler-types, which the roblox-ts build of jsxTestSource had.
const jsxTestPrelude = `
interface Array<T> { [n: number]: T; size(): number }
interface Boolean {}
interface Function {}
interface IArguments {}
interface Number {}
interface Object {}
interface RegExp {}
interface String {}
`

// jsxTestSource was compiled by roblox-ts 68648b0 (tsconfig of oracle/tests without project references,
// file src/tests/jsxcases.tsx). The expected Luau of each case below is the body of its function there.
const jsxTestSource = `/** @jsx React.createElement */
/** @jsxFrag React.Fragment */

declare namespace React {
	function createElement(tag: unknown, props: unknown, ...children: Array<unknown>): unknown;
	function Fragment(props: {}): unknown;
	namespace JSX {
		type Element = unknown;
		interface IntrinsicElements {
			frame: { Size?: number; Visible?: boolean; Text?: string; Name?: string };
			textlabel: { Text?: string };
		}
	}
}

interface Props {
	Size: number;
}

declare const label: string;
declare let n: number;
declare const props: Props;
declare const maybe: Props | undefined;
declare const arr: Array<unknown>;
declare function f(): number;
declare function Comp(props: { value: number }): unknown;
declare namespace Lib {
	function Comp(props: {}): unknown;
}

export function intrinsicWithProps() {
	const x = <frame Size={5} Visible Text="hi" />;
}

export function intrinsicNoProps() {
	const x = <frame />;
}

export function children() {
	const x = (
		<frame Name="root">
			<textlabel Text="a" />
			<textlabel />
		</frame>
	);
}

export function fragment() {
	const x = (
		<>
			<frame />
			<textlabel />
		</>
	);
}

export function emptyFragment() {
	const x = <></>;
}

export function expressionChildren() {
	const x = (
		<frame>
			{label}
			{}
			hello &amp; "world"
			{...arr}
		</frame>
	);
}

export function spreadFirst() {
	const x = <frame {...props} Visible />;
}

export function spreadMaybe() {
	const x = <frame Visible {...maybe} />;
}

export function spreadPrereqs() {
	const x = <frame Visible={n++ > f()} {...props} Text={label} Name={` + "`${n++}`" + `} />;
}

export function attributePrereqs() {
	const x = <frame Size={n++} Text={label} />;
}

export function component() {
	const x = <Comp value={1} />;
}

export function propertyAccessComponent() {
	const x = (
		<Lib.Comp>
			<frame />
		</Lib.Comp>
	);
}
`

// checkedJsxInitializers type checks jsxTestPrelude + jsxTestSource and returns a TransformState for it
// and the initializer of `const x` in each exported function, by function name.
func checkedJsxInitializers(t *testing.T) (*TransformState, map[string]*ast.Node) {
	t.Helper()
	fs := bundled.WrapFS(vfstest.FromMap(map[string]string{
		"/prelude.d.ts": jsxTestPrelude,
		"/test.tsx":     jsxTestSource,
		"/tsconfig.json": `{ "compilerOptions": { "noLib": true, "strict": true, "jsx": "react", "target": "ESNext", ` +
			`"module": "commonjs", "moduleDetection": "force" }, "files": ["prelude.d.ts", "test.tsx"] }`,
	}, false))
	host := compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil, nil)
	config, errs := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, host, nil)
	if len(errs) > 0 {
		t.Fatalf("config errors: %v", errs)
	}
	program := compiler.NewProgram(compiler.ProgramOptions{Config: config, Host: host})
	sourceFile := program.GetSourceFile("/test.tsx")
	diags := append(program.GetSyntacticDiagnostics(context.Background(), sourceFile), program.GetSemanticDiagnostics(context.Background(), sourceFile)...)
	if len(diags) > 0 {
		t.Fatalf("type errors: %d, first: %s", len(diags), diags[0].String())
	}
	c, done := program.GetTypeCheckerForFile(context.Background(), sourceFile)
	t.Cleanup(done)

	initializers := map[string]*ast.Node{}
	for _, statement := range sourceFile.Statements.Nodes {
		if !ast.IsFunctionDeclaration(statement) || statement.Body() == nil {
			continue
		}
		declarationList := statement.Body().Statements()[0].AsVariableStatement().DeclarationList
		initializers[statement.Name().Text()] = declarationList.AsVariableDeclarationList().Declarations.Nodes[0].Initializer()
	}

	state := &TransformState{
		TypeChecker:         c,
		SourceFile:          sourceFile,
		sourceFileText:      sourceFile.Text(),
		CompilerOptions:     program.Options(),
		Data:                &ProjectData{},
		MultiTransformState: NewMultiTransformState(),
		Services: &TransformServices{MacroManager: &MacroManager{
			typeChecker:        c,
			symbols:            sentinelMacroSymbols(),
			macroOnlyClasses:   map[*ast.Symbol]bool{},
			identifierMacros:   map[*ast.Symbol]IdentifierMacro{},
			callMacros:         map[*ast.Symbol]CallMacro{},
			constructorMacros:  map[*ast.Symbol]ConstructorMacro{},
			propertyCallMacros: map[*ast.Symbol]PropertyCallMacro{},
		}},
		ClassIdentifierMap:         map[*ast.Node]luau.AnyIdentifier{},
		HoistsByStatement:          map[*ast.Node][]*ast.Node{},
		IsHoisted:                  map[*ast.Symbol]bool{},
		getTypeCache:               map[*ast.Node]*checker.Type{},
		moduleIdBySymbol:           map[*ast.Symbol]luau.AnyIdentifier{},
		SymbolToIdMap:              map[*ast.Symbol]*luau.TemporaryIdentifier{},
		classElementToObjectKeyMap: map[*ast.Node]luau.Expression{},
	}
	return state, initializers
}

// sentinelMacroSymbols maps every SYMBOL_NAMES name and NOMINAL_LUA_TUPLE_NAME to a symbol that nothing in jsxTestSource resolves
// to: the prelude has no compiler-types to look them up in.
func sentinelMacroSymbols() map[string]*ast.Symbol {
	symbols := map[string]*ast.Symbol{NOMINAL_LUA_TUPLE_NAME: {Name: NOMINAL_LUA_TUPLE_NAME}}
	names := reflect.ValueOf(SYMBOL_NAMES)
	for _, field := range names.Fields() {
		name := field.String()
		symbols[name] = &ast.Symbol{Name: name}
	}
	return symbols
}

// Expected Luau from roblox-ts 68648b0 (see jsxTestSource).
func TestJsx(t *testing.T) { //nolint:paralleltest // the subtests share one TransformState.
	state, initializers := checkedJsxInitializers(t)

	tests := []struct {
		name string
		want string
	}{
		{"intrinsicWithProps", `local x = React.createElement("frame", {
    Size = 5,
    Visible = true,
    Text = "hi",
})
`},
		{"intrinsicNoProps", `local x = React.createElement("frame")
`},
		{"children", `local x = (React.createElement("frame", {
    Name = "root",
}, React.createElement("textlabel", {
    Text = "a",
}), React.createElement("textlabel")))
`},
		{"fragment", `local x = (React.createElement(React.Fragment, nil, React.createElement("frame"), React.createElement("textlabel")))
`},
		{"emptyFragment", `local x = React.createElement(React.Fragment)
`},
		{"expressionChildren", `local x = (React.createElement("frame", nil, label, "hello & \"world\"", unpack(arr)))
`},
		{"spreadFirst", `local _attributes = table.clone(props)
setmetatable(_attributes, nil)
_attributes.Visible = true
local x = React.createElement("frame", _attributes)
`},
		{"spreadMaybe", `local _attributes = {
    Visible = true,
}
if maybe then
    for _k, _v in maybe do
        _attributes[_k] = _v
    end
end
local x = React.createElement("frame", _attributes)
`},
		{"spreadPrereqs", `local _attributes = {}
local _original = n
n += 1
_attributes.Visible = _original > f()
for _k, _v in props do
    _attributes[_k] = _v
end
_attributes.Text = label
local _original_1 = n
n += 1
_attributes.Name = ` + "`{_original_1}`" + `
local x = React.createElement("frame", _attributes)
`},
		{"attributePrereqs", `local _attributes = {}
local _original = n
n += 1
_attributes.Size = _original
_attributes.Text = label
local x = React.createElement("frame", _attributes)
`},
		{"component", `local x = React.createElement(Comp, {
    value = 1,
})
`},
		{"propertyAccessComponent", `local x = (React.createElement(Lib.Comp, nil, React.createElement("frame")))
`},
	}

	for _, tt := range tests { //nolint:paralleltest // the subtests share one TransformState.
		t.Run(tt.name, func(t *testing.T) {
			initializer := initializers[tt.name]
			if initializer == nil {
				t.Fatalf("no case %q in jsxTestSource", tt.name)
			}
			prereqs := NewPrereqs()
			result := transformExpression(state, prereqs, initializer)
			if got, want := renderMacroResult(prereqs, result), testTabs(tt.want); got != want {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}

// JSX text whitespace and entity handling, from TypeScript's tsxReactEmitWhitespace cases.
func TestFixupWhitespaceAndDecodeEntities(t *testing.T) {
	t.Parallel()
	tests := []struct {
		text string
		want string
		ok   bool
	}{
		{"value", "value", true},
		{"  a  ", "  a  ", true},
		{"\n\t\t", "", false},
		{"value\n\n\t\t", "value", true},
		{"  a  \n  b  \n  c  ", "  a b c  ", true},
		{"&amp;&lt;&gt;&quot;&apos;&#65;&#x42;&unknown;\\path", "&<>\"'AB&unknown;\\path", true},
		{"&nbsp;&hearts;&#x1F600;", " ♥\U0001F600", true},
	}
	for _, tt := range tests {
		got, ok := fixupWhitespaceAndDecodeEntities(tt.text)
		if got != tt.want || ok != tt.ok {
			t.Errorf("fixupWhitespaceAndDecodeEntities(%q) = %q, %v; want %q, %v", tt.text, got, ok, tt.want, tt.ok)
		}
	}
}
