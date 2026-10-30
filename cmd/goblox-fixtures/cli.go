package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const cliPackageJson = `{ "name": "@rbxts/cli-scenario", "version": "1.0.0", "main": "out/init.lua", "types": "out/index.d.ts" }
`

const cliTsconfig = `{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"jsx": "react",
		"jsxFactory": "Roact.jsx",
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"resolveJsonModule": true,
		"forceConsistentCasingInFileNames": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"rootDir": "src",
		"outDir": "out",
		"baseUrl": "src"
	}
}
`

const cliRojoGame = `{
	"name": "cli",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"rbxts_include": {
				"$path": "include",
				"node_modules": { "$className": "Folder", "@rbxts": { "$path": "node_modules/@rbxts" } }
			},
			"out": { "$path": "out" }
		}
	}
}
`

const cliRojoModel = `{
	"name": "cli",
	"tree": {
		"$className": "Folder",
		"include": { "$path": "include" },
		"out": { "$path": "out" }
	}
}
`

// cliScenario is one rbxtsc invocation of a small project. files adds to, and overrides, the default
// project; a nil value removes the file.
type cliScenario struct {
	name  string
	flags []string
	files map[string]*string
	// mustWrite and mustPrint are what rbxtsc has to do for the scenario to test what it says it tests
	mustWrite string
	mustPrint string
	// intentional names a difference that docs/goblox/CLI.md documents; only the exit code is compared
	intentional string
}

func text(s string) *string { return &s }

// declarationFiles is a small @rbxts-style package: module and global types, baseUrl imports, a hand-written
// declaration file, comments, classes and generics.
func declarationFiles(overrides map[string]*string) map[string]*string {
	files := map[string]*string{
		"default.project.json": nil,
		"src/main.server.ts":   nil,
		"tsconfig.json":        text(strings.Replace(cliTsconfig, `"baseUrl": "src"`, `"baseUrl": "src", "declaration": true`, 1)),
		"src/index.ts": text(`/** The package's entry point. */
export * from "shared/util";
export { helper as renamedHelper } from "lib/helper";
import { Thing, type Options } from "lib/thing";
import Other = require("lib/other");

// not kept in the output
export function add(a: number, b: number) {
	return a + b;
}

export const part = new Instance("Part");
export const players = game.GetService("Players");

export function makeThing(options: Options) {
	return new Thing(options);
}

export const other = Other.other;
export type Pair<T> = [T, T];
export type Lazy = typeof import("lib/thing");
`),
		"src/lib/helper.ts": text("export function helper<T>(value: T): T {\n\treturn value;\n}\n"),
		"src/lib/other.ts":  text("const other = { a: 1, b: \"two\" } as const;\nexport = { other };\n"),
		"src/lib/thing.ts": text(`export interface Options {
	readonly name: string;
	size?: number;
}

export class Thing {
	public static count = 0;
	private secret = 1;
	protected kept = "a";

	constructor(public readonly options: Options) {}

	/** Doubles it. */
	public double(this: Thing, value = 2) {
		return value * this.secret;
	}

	public name() {
		return this.options.name;
	}
}

export enum Kind {
	A,
	B = "b",
}

export abstract class Base<T extends defined> {
	abstract value(): T;
}
`),
		"src/lib/data.json":   text("{ \"a\": 1 }\n"),
		"src/lib/usesData.ts": text("import data from \"lib/data.json\";\nexport const copy = data;\n"),
		// hand-written: printed again, comments stay, its specifiers are rewritten too
		"src/types.d.ts": text(`// a comment
import { Options } from 'lib/thing';
    declare global {
  interface Extra { options: Options; }
}
export type { Options };
declare const   service :string;
`),
	}
	for name, contents := range overrides {
		files[name] = contents
	}
	return files
}

// compileOrderFiles has a program order of helper.ts, a.ts, b.ts and c.ts. b.ts and c.ts end in a transform error.
func compileOrderFiles(c string, overrides map[string]*string) map[string]*string {
	files := map[string]*string{
		"src/main.server.ts": nil,
		"src/a.ts":           text("import { helper } from \"./helper\";\nexport const a = helper;\n"),
		"src/b.ts":           text("export const b = null;\n"),
		"src/c.ts":           text(c + "export const d = null;\n"),
		"src/helper.ts":      text("export const helper = \"helper\";\n"),
	}
	for name, contents := range overrides {
		files[name] = contents
	}
	return files
}

func cliScenarios() []cliScenario {
	withoutTsconfig := map[string]*string{"tsconfig.json": nil}
	typeError := map[string]*string{"src/bad.ts": text("export const a: string = 1;\n")}
	otherRojo := map[string]*string{"other.project.json": text(cliRojoGame), "default.project.json": nil}
	modelRojo := map[string]*string{"default.project.json": text(cliRojoModel)}
	packageFiles := map[string]*string{
		"default.project.json": nil,
		"src/index.ts":         text("export function add(a: number, b: number) {\n\treturn a + b;\n}\nexport type Pair = [number, number];\n"),
		"src/main.server.ts":   nil,
		"tsconfig.json":        text(strings.Replace(cliTsconfig, `"baseUrl": "src"`, `"baseUrl": "src", "declaration": true`, 1)),
	}
	return []cliScenario{
		{name: "build"},
		{name: "build command", flags: []string{"build"}},
		{name: "missing tsconfig", files: withoutTsconfig},
		{name: "missing -p path", flags: []string{"-p", "does-not-exist"}},
		{name: "-p folder", flags: []string{"-p", "src"}},
		{name: "-p file", flags: []string{"-p", "tsconfig.json"}},
		{name: "invalid tsconfig", files: map[string]*string{"tsconfig.json": text("{ not json")}, intentional: "typescript-go's JSON parser reports TS1005 where TypeScript 5.9 reports TS1136"},
		{name: "tsconfig without outDir", files: map[string]*string{"tsconfig.json": text(strings.Replace(cliTsconfig, `"outDir": "out",`, "", 1))}},
		{name: "tsconfig without rootDir", files: map[string]*string{"tsconfig.json": text(strings.Replace(cliTsconfig, `"rootDir": "src",`, "", 1))}},
		{name: "tsconfig without typeRoots", files: map[string]*string{"tsconfig.json": text(strings.Replace(cliTsconfig, `"typeRoots": ["node_modules/@rbxts"],`, "", 1))}},
		{name: "tsconfig without strict", files: map[string]*string{"tsconfig.json": text(strings.Replace(cliTsconfig, `"strict": true,`, "", 1))}},
		{name: "tsconfig with allowJs", files: map[string]*string{"tsconfig.json": text(strings.Replace(cliTsconfig, `"strict": true,`, `"strict": true, "allowJs": true,`, 1))}},
		{name: "missing package.json", files: map[string]*string{"package.json": nil}},
		{name: "invalid option", flags: []string{"--notAnOption"}},
		{name: "invalid options", flags: []string{"--notAnOption", "value", "--other", "extra"}},
		{name: "invalid kebab option", flags: []string{"--not-an-option"}},
		{name: "invalid letters", flags: []string{"-xyz"}},
		{name: "invalid option with --help", flags: []string{"--notAnOption", "--help"}, intentional: "goblox's flag list is printed by Go's flag package"},
		{name: "extra argument", flags: []string{"extra"}},
		{name: "build after a flag", flags: []string{"--luau", "build"}},
		{name: "--no-include", flags: []string{"--no-include"}},
		{name: "--noLuau", flags: []string{"--noLuau"}},
		{name: "kebab option", flags: []string{"--log-truthy-changes"}},
		{name: "--no-luau", flags: []string{"--no-luau"}},
		{name: "--luau false", flags: []string{"--luau", "false"}},
		{name: "--luau true", flags: []string{"--luau", "true"}},
		{name: "--rojo without value", flags: []string{"--rojo"}},
		{name: "--includePath without value", flags: []string{"--includePath"}},
		{name: "-p without value", flags: []string{"-p"}},
		{name: "invalid --type", flags: []string{"--type", "nonsense"}},
		{name: "--type without value", flags: []string{"--type"}},
		{name: "--usePolling without --watch", flags: []string{"--usePolling"}},
		{name: "type error", files: typeError},
		// the importers of a file come right after it, and an error stops the transforms: only c.ts's error is reported
		{name: "compile order: importers", mustPrint: "src/c.ts", files: compileOrderFiles("import { helper } from \"./helper\";\nexport const c = helper;\n", nil)},
		{name: "compile order: type error", mustPrint: "src/c.ts", files: compileOrderFiles("import { helper } from \"./helper\";\nexport const c: number = helper;\n", map[string]*string{
			"src/b.ts": text("export const b = null;\n"),
		})},
		{name: "compile order: augmentation", mustPrint: "src/c.ts", files: compileOrderFiles("export {};\ndeclare module \"./helper\" {\n\tinterface H {\n\t\tc: number;\n\t}\n}\n", map[string]*string{
			"src/helper.ts": text("export interface H {}\nexport const helper = \"helper\";\n"),
		})},
		{name: "compile order: reference", mustPrint: "src/c.ts", files: compileOrderFiles("/// <reference path=\"./helper.d.ts\" />\nexport {};\n", map[string]*string{
			"src/a.ts":        text("/// <reference path=\"./helper.d.ts\" />\nexport {};\n"),
			"src/helper.ts":   nil,
			"src/helper.d.ts": text("declare const helper: string;\n"),
		})},
		{name: "--noInclude", flags: []string{"--noInclude"}},
		{name: "--type game", flags: []string{"--type", "game"}},
		{name: "--type model", flags: []string{"--type", "model"}, files: modelRojo},
		{name: "--type model, game rojo", flags: []string{"--type", "model"}},
		{name: "inferred model", files: modelRojo},
		{name: "--type package", flags: []string{"--type", "package"}, files: packageFiles, mustWrite: "out/index.d.ts"},
		{name: "package: declarations", flags: []string{"--type", "package"}, files: declarationFiles(nil), mustWrite: "out/lib/thing.d.ts"},
		{name: "package: inferred", files: declarationFiles(nil), mustWrite: "out/types.d.ts"},
		{name: "package: declarationDir", mustWrite: "out/types/index.d.ts.map", files: declarationFiles(map[string]*string{
			"tsconfig.json": text(strings.Replace(cliTsconfig, `"baseUrl": "src"`, `"baseUrl": "src", "declaration": true, "declarationDir": "out/types", "declarationMap": true`, 1)),
		})},
		{name: "package: no declaration", files: declarationFiles(map[string]*string{"tsconfig.json": text(cliTsconfig)}), mustWrite: "out/init.luau"},
		{name: "package: paths", mustWrite: "out/uses.d.ts", files: declarationFiles(map[string]*string{
			"src/index.ts":        text("export const unused = 1;\n"),
			"src/lib/usesData.ts": nil,
			"src/types.d.ts":      nil,
			"tsconfig.json":       text(strings.Replace(cliTsconfig, `"baseUrl": "src"`, `"declaration": true, "paths": { "lib/*": ["./src/lib/*"], "~shared": ["./src/shared/util.ts"] }`, 1)),
			"src/uses.ts":         text("import { helper } from 'lib/helper';\nimport { value } from '~shared';\nexport const a = helper(value);\nexport type B = typeof import('lib/helper');\n"),
		})},
		{name: "package: declaration error", mustPrint: "TS4", files: declarationFiles(map[string]*string{
			"src/bad.ts": text("export const value = class {\n\tprivate hidden = 1;\n};\n"),
		})},
		{name: "package: consumed", mustWrite: "out/main.server.luau", files: map[string]*string{
			"node_modules/@rbxts/mypkg/package.json":   text("{ \"name\": \"@rbxts/mypkg\", \"main\": \"out/init.luau\", \"types\": \"out/index.d.ts\" }\n"),
			"node_modules/@rbxts/mypkg/out/index.d.ts": text("export declare function add(a: number, b: number): number;\n"),
			"node_modules/@rbxts/mypkg/out/init.luau":  text("return { add = function(a, b) return a + b end }\n"),
			"src/main.server.ts":                       text("import { add } from \"@rbxts/mypkg\";\nprint(add(1, 2));\n"),
		}},
		{name: "--type game, no rojo", flags: []string{"--type", "game"}, files: packageFiles},
		{name: "--rojo path", flags: []string{"--rojo", "other.project.json"}, files: otherRojo},
		{name: "--rojo missing path", flags: []string{"--rojo", "missing.project.json"}},
		{name: "--rojo folder", flags: []string{"--rojo", "src"}},
		{name: "--rojo broken json", flags: []string{"--rojo", "broken.project.json"}, files: map[string]*string{"broken.project.json": text("{ not json")}, intentional: "the message of the SyntaxError is JSON.parse's, V8's text isn't reproduced"},
		{name: "nested rojo project", files: map[string]*string{
			"default.project.json": text(strings.Replace(cliRojoGame, `"out": { "$path": "out" }`, `"out": { "$path": "out" }, "nested": { "$path": "nested.project.json" }`, 1)),
			"nested.project.json":  text("{ \"name\": \"nested\", \"tree\": { \"$className\": \"Folder\" } }\n"),
		}},
		{name: "broken nested rojo project", intentional: "the message of the SyntaxError is JSON.parse's, V8's text isn't reproduced", files: map[string]*string{
			"default.project.json": text(strings.Replace(cliRojoGame, `"out": { "$path": "out" }`, `"out": { "$path": "out" }, "nested": { "$path": "nested.project.json" }`, 1)),
			"nested.project.json":  text("{ not json"),
		}},
		{name: "no rojo file", files: map[string]*string{"default.project.json": nil}},
		{name: "--includePath", flags: []string{"--includePath", "other", "--rojo", "other.project.json"}, files: map[string]*string{
			"other.project.json": text(strings.ReplaceAll(cliRojoGame, `"$path": "include"`, `"$path": "other"`)),
		}},
		{name: "--luau", flags: []string{"--luau"}},
		{name: "--luau=false", flags: []string{"--luau=false"}},
		{name: "--verbose", flags: []string{"--verbose"}, intentional: "goblox's --verbose adds a line per build step and the timings vary"},
		{name: "--version", flags: []string{"--version"}, intentional: "goblox prints its own version line"},
		{name: "--help", flags: []string{"--help"}, intentional: "goblox's flag list is printed by Go's flag package"},
	}
}

// runCliScenarios compares rbxtsc and goblox on the invocations docs/goblox/CLI.md promises to match: exit
// code, stdout, stderr and the files written.
func runCliScenarios(oracle string, filter *regexp.Regexp, verbose bool) int {
	tempRoot, err := os.MkdirTemp("", "goblox-fixtures-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees
	if long, err := filepath.EvalSymlinks(tempRoot); err == nil {
		tempRoot = long
	}
	defer os.RemoveAll(tempRoot)
	goblox, err := buildGoblox(tempRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
		return 2
	}

	total, matched := 0, 0
	for i, scenario := range cliScenarios() {
		if filter != nil && !filter.MatchString(scenario.name) {
			continue
		}
		total++
		dir := filepath.Join(tempRoot, fmt.Sprintf("scenario%d", i))
		if err := writeCliProject(oracle, dir, scenario.files); err != nil {
			fmt.Fprintln(os.Stderr, "goblox-fixtures:", err)
			return 2
		}
		differences, sameExitCode, err := compareCli(dir, oracle, goblox, scenario)
		switch {
		case err != nil:
			fmt.Printf("%-32s error: %v\n", scenario.name, err)
		case scenario.intentional != "" && sameExitCode:
			matched++
			fmt.Printf("%-32s intentional: %s\n", scenario.name, scenario.intentional)
		case differences != "":
			fmt.Printf("%-32s differs\n", scenario.name)
			if verbose {
				fmt.Println(strings.TrimRight(differences, "\n"))
			}
		default:
			matched++
			fmt.Printf("%-32s match\n", scenario.name)
		}
	}
	fmt.Printf("\n%d of %d CLI scenarios match\n", matched, total)
	if matched != total {
		return 1
	}
	return 0
}

func writeCliProject(oracle string, dir string, files map[string]*string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := map[string]*string{
		"package.json":         text(cliPackageJson),
		"tsconfig.json":        text(cliTsconfig),
		"default.project.json": text(cliRojoGame),
		"src/main.server.ts":   text("print(\"hello\");\n"),
		"src/shared/util.ts":   text("export const value = [1, 2, 3].size();\n"),
		"src/services.d.ts":    text("declare const service: string;\n"),
		"src/data.json":        text("{}\n"),
		"src/keep.lua":         text("return 1\n"),
	}
	for name, contents := range files {
		base[name] = contents
	}
	for name, contents := range base {
		if contents == nil {
			continue
		}
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(*contents), 0o644); err != nil {
			return err
		}
	}
	nodeModules := filepath.Join(dir, "node_modules")
	if err := os.MkdirAll(nodeModules, 0o755); err != nil {
		return err
	}
	return copyTree(filepath.Join(oracle, "tests", "node_modules"), nodeModules, func(string) bool { return false })
}

// projectOutputs is every file under dir that the compiler may have written, by path relative to dir.
func projectOutputs(dir string, skip map[string]bool) (map[string]string, error) {
	outputs := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !skip[rel] {
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			outputs[rel] = string(contents)
		}
		return nil
	})
	return outputs, err
}

func compareOutputs(want map[string]string, got map[string]string) string {
	names := map[string]bool{}
	for name := range want {
		names[name] = true
	}
	for name := range got {
		names[name] = true
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	var differences strings.Builder
	for _, name := range sorted {
		wantContents, inWant := want[name]
		gotContents, inGot := got[name]
		switch {
		case !inWant:
			fmt.Fprintf(&differences, "only goblox wrote %s\n", name)
		case !inGot:
			fmt.Fprintf(&differences, "only rbxtsc wrote %s\n", name)
		case strings.HasSuffix(name, ".d.ts.map"):
			// typescript-go's source map generator picks other mappings, the files only have to exist
		case wantContents != gotContents:
			fmt.Fprintf(&differences, "%s differs\n%s", name, unifiedDiff("rbxtsc "+name, "goblox "+name, wantContents, gotContents))
		}
	}
	return differences.String()
}

// compareCli runs both compilers on the project in dir, each from the same clean tree.
func compareCli(dir string, oracle string, goblox string, scenario cliScenario) (string, bool, error) {
	flags := scenario.flags
	sources, err := projectOutputs(dir, nil)
	if err != nil {
		return "", false, err
	}
	skip := map[string]bool{}
	for name := range sources {
		skip[name] = true
	}
	removeOutputs := func() (map[string]string, error) {
		outputs, err := projectOutputs(dir, skip)
		if err != nil {
			return nil, err
		}
		for name := range outputs {
			if err := os.Remove(filepath.Join(dir, filepath.FromSlash(name))); err != nil {
				return nil, err
			}
		}
		return outputs, nil
	}

	want, err := runCompiler(dir, append(os.Environ(), "NODE_PATH="+filepath.Join(oracle, "out")), "node",
		append([]string{filepath.Join(oracle, "out", "CLI", "cli.js")}, flags...)...)
	if err != nil {
		return "", false, err
	}
	wantOutputs, err := removeOutputs()
	if err != nil {
		return "", false, err
	}
	var unproven string
	if _, wrote := wantOutputs[scenario.mustWrite]; scenario.mustWrite != "" && !wrote {
		unproven += fmt.Sprintf("rbxtsc did not write %s\n%s", scenario.mustWrite, want.stdout)
	}
	if !strings.Contains(want.stdout, scenario.mustPrint) {
		unproven += fmt.Sprintf("rbxtsc did not print %s\n%s", scenario.mustPrint, want.stdout)
	}
	got, err := runCompiler(dir, os.Environ(), goblox, flags...)
	if err != nil {
		return "", false, err
	}
	gotOutputs, err := removeOutputs()
	if err != nil {
		return "", false, err
	}
	return unproven + describeRunDifferences(want, got) + compareOutputs(wantOutputs, gotOutputs), unproven == "" && want.exitCode == got.exitCode, nil
}
