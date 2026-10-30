package project

import (
	"context"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/tspath"
)

// These projects bring their own tiny @rbxts/compiler-types and @rbxts/types, so they run without the oracle
// and declare almost none of what the macro tables look for.
const minimalCompilerTypes = `
interface Array<T> { length: number; [n: number]: T; }
interface Boolean {}
interface CallableFunction {}
interface Function {}
interface IArguments {}
interface NewableFunction {}
interface Number {}
interface Object {}
interface RegExp {}
interface String {}
`

const minimalTypes = `
declare function print(...params: Array<unknown>): void;
interface vector {
	readonly x: number;
}
declare namespace vector {
	function create(x: number, y: number, z: number): vector;
}
`

func buildMinimalProject(t *testing.T, types string, compilerTypes string, files map[string]string) (*EmitResult, string) {
	t.Helper()
	dir := t.TempDir()
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	dir = filepath.ToSlash(dir)
	for name, text := range map[string]string{
		"package.json":         `{ "name": "fixture" }`,
		"default.project.json": incrementalTestProjectJSON,
		"tsconfig.json":        incrementalTestTsconfig(false, ""),
		"node_modules/@rbxts/compiler-types/package.json": `{ "name": "@rbxts/compiler-types", "types": "index.d.ts" }`,
		"node_modules/@rbxts/compiler-types/index.d.ts":   compilerTypes,
		"node_modules/@rbxts/types/package.json":          `{ "name": "@rbxts/types", "types": "index.d.ts" }`,
		"node_modules/@rbxts/types/index.d.ts":            types,
	} {
		writeTestFile(t, dir, name, text)
	}
	for name, text := range files {
		writeTestFile(t, dir, name, text)
	}
	result, err := Build(context.Background(), BuildOptions{
		Project:          dir,
		CurrentDirectory: tspath.NormalizePath(dir),
		FS:               DefaultFS(),
		Overrides:        PartialProjectOptions{},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	return result, dir
}

func wantCleanBuild(t *testing.T, result *EmitResult) {
	t.Helper()
	if len(result.Panics) > 0 {
		t.Fatalf("panics: %v", result.Panics)
	}
	if lines := diagnosticLines(result.Diagnostics); len(lines) > 0 {
		t.Fatalf("diagnostics:\n%s", strings.Join(lines, "\n"))
	}
}

func wantOutput(t *testing.T, dir string, name string, want ...string) {
	t.Helper()
	output := readTestFile(t, dir, name)
	for _, line := range want {
		if !strings.Contains(output, line) {
			t.Errorf("%s has no %q:\n%s", name, line, output)
		}
	}
}

// TypeScript 7 reserves `Object` at the top level of a CommonJS module, TypeScript 5 does not.
func TestObjectNameAtTopLevel(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	result, dir := buildMinimalProject(t, minimalTypes, minimalCompilerTypes, map[string]string{
		"src/shared/object-utils.ts":     "const Object = { size: 1 };\nexport default Object;\n",
		"src/shared/main.ts":             "import Object from \"./object-utils\";\nprint(Object.size);\n",
		"src/shared/named.ts":            "import { Object } from \"./object-named\";\nprint(Object);\n",
		"src/shared/object-named.ts":     "export const Object = 1;\n",
		"src/shared/object-function.ts":  "export function Object() {\n\treturn 1;\n}\n",
		"src/shared/object-namespace.ts": "namespace Object {\n\texport const size = 1;\n}\nprint(Object.size);\nexport {};\n",
	})
	wantCleanBuild(t, result)
	wantOutput(t, dir, "out/shared/main.luau", `local Object = TS.import(script, game:GetService("ReplicatedStorage"), "shared", "object-utils").default`, "print(Object.size)")
	wantOutput(t, dir, "out/shared/object-utils.luau", "local Object = {")
}

// `require` and `exports` are reserved in TypeScript 5 too, so rbxtsc reports them: they have to stay.
func TestOtherReservedNamesStillReported(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	for name, source := range map[string]string{
		"require": "const require = 1;\nprint(require);\n",
		"exports": "const exports = 1;\nprint(exports);\n",
	} {
		result, _ := buildMinimalProject(t, minimalTypes, minimalCompilerTypes, map[string]string{"src/shared/main.ts": source})
		lines := diagnosticLines(result.Diagnostics)
		if !slices.ContainsFunc(lines, func(line string) bool { return strings.Contains(line, "2441") && strings.Contains(line, name) }) {
			t.Errorf("%s: no TS2441 in %v", name, lines)
		}
	}
}

// A macro without a declaration is skipped. minimalTypes declares `vector` without the methods of
// propertyCallMacros' "vector" table, as @rbxts/types before 1.0.957 does, and nearly no other macro exists.
func TestTypesWithoutMacroMethods(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	result, dir := buildMinimalProject(t, minimalTypes, minimalCompilerTypes, map[string]string{
		"src/shared/main.ts": "const a = vector.create(1, 2, 3);\nprint(a.x);\nconst list: Array<number> = [1];\nfor (const value of list) {\n\tprint(value);\n}\n",
	})
	wantCleanBuild(t, result)
	wantOutput(t, dir, "out/shared/main.luau", "local a = vector.create(1, 2, 3)", "print(a.x)")
}

// The macros still apply to what the types do declare, here two of the "vector" table's five methods.
func TestTypesWithSomeMacroMethods(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	types := minimalTypes + "interface vector {\n\tadd(this: vector, v: vector): vector;\n\tsub(this: vector, v: vector): vector;\n}\n"
	result, dir := buildMinimalProject(t, types, minimalCompilerTypes, map[string]string{
		"src/shared/main.ts": "const a = vector.create(1, 2, 3);\nprint(a.add(a));\nprint(a.sub(a));\n",
	})
	wantCleanBuild(t, result)
	wantOutput(t, dir, "out/shared/main.luau", "print(a + a)", "print(a - a)")
}

// Without the nominal property a LuaTuple cannot be told from its array, so using one stays an error.
func TestLuaTupleWithoutNominalProperty(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	compilerTypes := minimalCompilerTypes + "type LuaTuple<T extends Array<any>> = T & { readonly old_nominal: unique symbol };\n"
	types := minimalTypes + "declare function pair(): LuaTuple<[number, number]>;\n"
	result, dir := buildMinimalProject(t, types, compilerTypes, map[string]string{
		"src/shared/unused.ts": "print(1);\n",
	})
	wantCleanBuild(t, result)
	wantOutput(t, dir, "out/shared/unused.luau", "print(1)")

	result, _ = buildMinimalProject(t, types, compilerTypes, map[string]string{
		"src/shared/used.ts": "const [a] = pair();\nprint(a);\n",
	})
	failure := strings.Join(slices.Collect(maps.Values(result.Panics)), "\n") + strings.Join(diagnosticLines(result.Diagnostics), "\n")
	if !strings.Contains(failure, "MacroManager could not find symbol for _nominal_LuaTuple") {
		t.Fatalf("no error for a LuaTuple without its nominal property: %q", failure)
	}
}

// TypeScript 7 flags this module's `export=` symbol as a namespace, and alias resolution stops there. The
// import has a value all the same: the program would otherwise lose it and fail at runtime.
func TestExportEqualsNextToExportedTypes(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	result, dir := buildMinimalProject(t, minimalTypes, minimalCompilerTypes, map[string]string{
		"src/shared/zone.d.ts":      "export interface Zone {\n\tid: number;\n}\nexport type Shape = \"Block\";\nexport interface QuickZone {\n\tzone: Zone;\n}\ndeclare const QuickZone: QuickZone;\nexport = QuickZone;\n",
		"src/shared/zone.lua":       "return { zone = { id = 1 } }\n",
		"src/shared/main.ts":        "import QuickZone from \"./zone\";\nprint(QuickZone.zone.id);\n",
		"src/shared/type-only.d.ts": "export interface Other {\n\tid: number;\n}\ninterface Only {\n\tid: number;\n}\nexport = Only;\n",
		"src/shared/type-only.lua":  "return {}\n",
		"src/shared/types.ts":       "import Only from \"./type-only\";\nconst only: Only = { id: 1 };\nprint(only.id);\n",
	})
	wantCleanBuild(t, result)
	wantOutput(t, dir, "out/shared/main.luau", `local QuickZone = TS.import(script, game:GetService("ReplicatedStorage"), "shared", "zone")`, "print(QuickZone.zone.id)")
	if output := readTestFile(t, dir, "out/shared/types.luau"); strings.Contains(output, "TS.import") {
		t.Errorf("types.luau imports a module that only has a type:\n%s", output)
	}
}
