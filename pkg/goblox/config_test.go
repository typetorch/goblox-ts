package goblox_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/pkg/goblox"
)

func TestAutomaticTypesAndPluginEntries(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFiles(t, dir, map[string]string{
		// no `types`: every package of the type roots is global, as in TypeScript 5
		"base/tsconfig.base.json": `{
			// comments and trailing commas are allowed
			"compilerOptions": {
				"plugins": [{ "transform": "example-transformer", "level": 2, "names": ["a"] }, { "name": "ls-plugin" },],
			},
		}`,
		"tsconfig.json":                 `{ "extends": "./base/tsconfig.base.json", "compilerOptions": { "noLib": true, "typeRoots": ["types"], "strict": true }, "include": ["src"] }`,
		"types/globals/index.d.ts":      "interface Array<T> {} interface Boolean {} interface Function {} interface IArguments {} interface Number {} interface Object {} interface RegExp {} interface String {} interface CallableFunction {} interface NewableFunction {}\ndeclare function globalHelper(): number;",
		"types/not-needed/package.json": `{ "typings": null }`,
		"types/not-needed/index.d.ts":   "this is not valid TypeScript",
		"src/index.ts":                  "export const x: number = globalHelper();",
	})
	ctx := context.Background()
	program, errs := goblox.LoadProgram(goblox.LoadOptions{Project: dir, CurrentDirectory: dir, Checkers: 1})
	if program == nil {
		t.Fatalf("LoadProgram: %v", errs)
	}
	diags := append(goblox.GetConfigDiagnostics(ctx, program), goblox.GetPreEmitDiagnostics(ctx, program, nil)...)
	if len(diags) != 0 {
		var b strings.Builder
		goblox.WriteDiagnostics(&b, diags, dir)
		t.Fatalf("diagnostics:\n%s", b.String())
	}

	got := goblox.PluginEntries(program)
	want := []map[string]any{
		{"transform": "example-transformer", "level": float64(2), "names": []any{"a"}},
		{"name": "ls-plugin"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PluginEntries = %#v, want %#v", got, want)
	}

	if _, errs := goblox.PluginsForProgram(program); len(errs) != 1 {
		t.Fatalf("expected one error for the unregistered transformer, got %v", errs)
	}
	var gotEntry map[string]any
	goblox.RegisterPlugin("example-transformer", func(entry map[string]any) (goblox.Plugin, error) {
		gotEntry = entry
		return doublePlugin{}, nil
	})
	plugins, pluginErrs := goblox.PluginsForProgram(program)
	if len(pluginErrs) != 0 || len(plugins) != 1 || gotEntry["level"] != float64(2) {
		t.Fatalf("PluginsForProgram = %v, %v (entry %v)", plugins, pluginErrs, gotEntry)
	}
}
