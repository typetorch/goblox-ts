package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// The fixtures build a project, apply an edit, build incrementally and compare the outputs and diagnostics
// with a clean build of the same files. The project needs @rbxts/compiler-types and @rbxts/types, copied from
// the oracle's test project (GOBLOX_ORACLE, see docs/goblox/PORTING.md).
const defaultOracle = `C:/Users/phasenull/.t3/scratch/2026-10-29-act-as-the-implementation-sub-37576c8e/oracle`

// literalDoublePlugin replaces `double(x)` with twice the literal type of x, so its output depends on types
// declared in other files, like @typetorch/transformer's.
type literalDoublePlugin struct{}

func (literalDoublePlugin) Name() string { return "literal-double" }

func (literalDoublePlugin) TransformSourceFile(ctx *TransformContext, file *ast.SourceFile) *ast.SourceFile {
	var visitor *ast.NodeVisitor
	visitor = ctx.EmitContext.NewNodeVisitor(func(node *ast.Node) *ast.Node {
		if node.Kind == ast.KindCallExpression {
			call := node.AsCallExpression()
			symbol := ctx.Checker.GetSymbolAtLocation(call.Expression)
			if symbol != nil && symbol.Name == "double" {
				text := ctx.Checker.TypeToString(ctx.Checker.GetTypeAtLocation(call.Arguments.Nodes[0]))
				if value, err := strconv.Atoi(text); err == nil {
					return ctx.Factory.NewNumericLiteral(strconv.Itoa(value*2), ast.TokenFlagsNone)
				}
			}
		}
		return visitor.VisitEachChild(node)
	})
	return visitor.VisitSourceFile(file)
}

var registerLiteralDouble = sync.OnceFunc(func() {
	RegisterPlugin("literal-double", func(map[string]any) (Plugin, error) { return literalDoublePlugin{}, nil })
})

func writeTestFile(t *testing.T, dir string, name string, text string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, dir string, name string) string {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(text)
}

func editTestFile(t *testing.T, dir string, name string, old string, new string) {
	t.Helper()
	text := readTestFile(t, dir, name)
	if !strings.Contains(text, old) {
		t.Fatalf("%s has no %q:\n%s", name, old, text)
	}
	writeTestFile(t, dir, name, strings.Replace(text, old, new, 1))
}

// readTree reads every file below dir, without the build info files, by relative path.
func readTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() || strings.HasSuffix(path, ".tsbuildinfo") {
			return nil
		}
		text, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(text)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func diagnosticLines(diags []*ast.Diagnostic) []string {
	lines := make([]string, 0, len(diags))
	for _, d := range diags {
		fileName := ""
		if d.File() != nil {
			fileName = d.File().FileName()
		}
		lines = append(lines, fmt.Sprintf("%s:%d: %d %s", fileName, d.Pos(), d.Code(), d.String()))
	}
	return lines
}

// incrementalFixture is a project and the session of the builds to test.
type incrementalFixture struct {
	t       *testing.T
	dir     string
	persist bool
	session *IncrementalSession
	cache   *SourceFileCache
	// emitFirst is the next incremental build's EmitFirst
	emitFirst bool
	// watchPersist is what `goblox -w` does: keep the state in memory and write the cache too
	watchPersist bool
}

const incrementalTestProjectJSON = `{
	"name": "fixture",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"include": { "$path": "include", "node_modules": { "$className": "Folder", "@rbxts": { "$path": "node_modules/@rbxts" } } },
			"shared": { "$path": "out/shared" }
		},
		"ServerScriptService": { "$className": "ServerScriptService", "server": { "$path": "out/server" } },
		"StarterPlayer": { "$className": "StarterPlayer", "StarterPlayerScripts": { "$className": "StarterPlayerScripts", "client": { "$path": "out/client" } } }
	}
}
`

func incrementalTestTsconfig(plugins bool, extra string) string {
	pluginEntries := ""
	if plugins {
		pluginEntries = `"plugins": [{ "transform": "literal-double" }],`
	}
	return `{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"experimentalDecorators": true,
		"resolveJsonModule": true,
		"typeRoots": ["node_modules/@rbxts"],
		"types": ["types", "compiler-types"],
		"rootDir": "src",
		"outDir": "out",
		"incremental": true,
		"tsBuildInfoFile": "out/tsconfig.tsbuildinfo",
		` + pluginEntries + extra + `
		"allowUnreachableCode": false
	},
	"include": ["src"]
}
`
}

func newIncrementalFixture(t *testing.T, rbxts string, plugins bool, persist bool) *incrementalFixture {
	t.Helper()
	dir := t.TempDir()
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	dir = filepath.ToSlash(dir)
	for _, name := range []string{"compiler-types", "types"} {
		if err := os.CopyFS(filepath.Join(dir, "node_modules", "@rbxts", name), os.DirFS(filepath.Join(rbxts, name))); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{
		"package.json":         `{ "name": "fixture" }`,
		"default.project.json": incrementalTestProjectJSON,
		"tsconfig.json":        incrementalTestTsconfig(plugins, ""),
		"src/shared/globals.d.ts": "declare function double(value: number): number;\n" +
			"declare const GLOBAL_LIMIT: number;\n",
		"src/shared/types.ts": "export interface Item {\n\tid: number;\n}\n",
		"src/shared/net.ts": "export const PORT = 1;\n" +
			"export const TYPED_PORT: number = 5;\n" +
			"export function connect(port: number): number {\n\treturn port + PORT + GLOBAL_LIMIT;\n}\n",
		"src/server/main.server.ts": "import { connect, PORT } from \"shared/net\";\nimport { Item } from \"shared/types\";\n" +
			"const item: Item = { id: connect(double(PORT)) };\nprint(item.id);\n",
		"src/server/other.ts": "import { TYPED_PORT } from \"shared/net\";\nexport const VALUE = TYPED_PORT + 1;\n",
		"src/client/user.client.ts": "import { connect } from \"shared/net\";\nprint(connect(2));\n",
		"src/client/leaf.ts":        "export const LEAF = double(21);\n",
		"src/client/quiet.ts":       "export const QUIET = 1;\n",
	} {
		writeTestFile(t, dir, name, text)
	}
	// imports below src go through baseUrl
	editTestFile(t, dir, "tsconfig.json", `"rootDir": "src",`, `"rootDir": "src", "baseUrl": "src",`)
	return &incrementalFixture{t: t, dir: dir, persist: persist, session: &IncrementalSession{Persist: persist, Memory: !persist}, cache: &SourceFileCache{}}
}

func (f *incrementalFixture) build(incremental bool) *EmitResult {
	f.t.Helper()
	opts := BuildOptions{
		Project:          f.dir,
		CurrentDirectory: tspath.NormalizePath(f.dir),
		FS:               DefaultFS(),
		Overrides:        PartialProjectOptions{},
	}
	if incremental {
		if f.persist {
			// a new process
			f.session = &IncrementalSession{Persist: true}
		} else if f.watchPersist {
			f.session.Persist = true
			opts.SourceFileCache = f.cache
		} else {
			opts.SourceFileCache = f.cache
		}
		opts.Incremental = f.session
		opts.EmitFirst = f.emitFirst
	}
	result, err := Build(context.Background(), opts)
	if err != nil {
		f.t.Fatalf("build: %v", err)
	}
	if len(result.Panics) > 0 {
		f.t.Fatalf("panics: %v", result.Panics)
	}
	return result
}

// check builds incrementally and compares with a clean build, which leaves the incremental build's files
// in place. It returns the result of the incremental build.
func (f *incrementalFixture) check(step string) *EmitResult {
	f.t.Helper()
	result := f.build(true)
	outDir := filepath.Join(f.dir, "out")

	// keep the incremental build's outputs, with its cache
	saved := map[string][]byte{}
	if err := filepath.WalkDir(outDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		saved[path], err = os.ReadFile(path)
		return err
	}); err != nil {
		f.t.Fatal(err)
	}
	if err := os.RemoveAll(outDir); err != nil {
		f.t.Fatal(err)
	}
	clean := f.build(false)
	cleanOutput := readTree(f.t, outDir)
	if err := os.RemoveAll(outDir); err != nil {
		f.t.Fatal(err)
	}
	for path, text := range saved {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(path, text, 0o644); err != nil {
			f.t.Fatal(err)
		}
	}

	got, want := diagnosticLines(result.Diagnostics), diagnosticLines(clean.Diagnostics)
	if !slices.Equal(got, want) {
		f.t.Errorf("%s: diagnostics differ from a clean build\nincremental:\n  %s\nclean:\n  %s", step, strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	if result.EmitSkipped != clean.EmitSkipped {
		f.t.Errorf("%s: EmitSkipped %v, clean %v", step, result.EmitSkipped, clean.EmitSkipped)
	}
	if !clean.EmitSkipped {
		output := readTree(f.t, outDir)
		for name, text := range cleanOutput {
			if output[name] != text {
				f.t.Errorf("%s: %s differs from a clean build\nincremental:\n%s\nclean:\n%s", step, name, output[name], text)
			}
		}
		for name := range output {
			if _, ok := cleanOutput[name]; !ok {
				f.t.Errorf("%s: %s is not in a clean build", step, name)
			}
		}
	}
	return result
}

func (f *incrementalFixture) compiled(step string, atMost int) {
	f.t.Helper()
	if f.session.lastCompiled > atMost {
		f.t.Errorf("%s: compiled %d files, want at most %d", step, f.session.lastCompiled, atMost)
	}
}

func (f *incrementalFixture) wantErrorIn(step string, result *EmitResult, fileName string) {
	f.t.Helper()
	for _, d := range result.Diagnostics {
		if d.File() != nil && strings.HasSuffix(d.File().FileName(), fileName) {
			return
		}
	}
	f.t.Errorf("%s: no diagnostic in %s: %v", step, fileName, diagnosticLines(result.Diagnostics))
}

func TestIncrementalFixtures(t *testing.T) {
	oracle := os.Getenv("GOBLOX_ORACLE")
	if oracle == "" {
		oracle = defaultOracle
	}
	rbxts := filepath.Join(oracle, "tests", "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(rbxts, "compiler-types")); err != nil {
		t.Skipf("no @rbxts/compiler-types in the oracle (set GOBLOX_ORACLE): %v", err)
	}
	registerLiteralDouble()

	for _, plugins := range []bool{false, true} {
		for _, persist := range []bool{false, true} {
			t.Run(fmt.Sprintf("plugins=%v,persist=%v", plugins, persist), func(t *testing.T) {
				runIncrementalFixtures(t, rbxts, plugins, persist)
			})
		}
	}
}

func runIncrementalFixtures(t *testing.T, rbxts string, plugins bool, persist bool) {
	f := newIncrementalFixture(t, rbxts, plugins, persist)
	dir := f.dir

	f.check("cold")
	total := f.session.lastCompiled
	if total < 6 {
		t.Fatalf("cold: compiled %d files", total)
	}
	if plugins && !strings.Contains(readTestFile(t, dir, "out/client/leaf.luau"), "42") {
		t.Fatalf("the plugin did not run:\n%s", readTestFile(t, dir, "out/client/leaf.luau"))
	}

	result := f.check("no change")
	f.compiled("no change", 0)
	if len(result.EmittedFiles) != 0 {
		t.Errorf("no change: wrote %v", result.EmittedFiles)
	}

	// edits that reach one file
	editTestFile(t, dir, "src/client/quiet.ts", "export const QUIET = 1;", "// a comment\nexport const QUIET = 1;")
	f.check("leaf comment")
	f.compiled("leaf comment", 1)

	editTestFile(t, dir, "src/client/quiet.ts", "QUIET = 1", "QUIET = 2")
	result = f.check("leaf edit")
	f.compiled("leaf edit", 1)
	if len(result.EmittedFiles) != 1 {
		t.Errorf("leaf edit: wrote %v", result.EmittedFiles)
	}

	editTestFile(t, dir, "src/shared/net.ts", "return port", "// not part of the shape\n\treturn port")
	f.check("shared comment")
	f.compiled("shared comment", 1)

	editTestFile(t, dir, "src/shared/net.ts", "TYPED_PORT: number = 5", "TYPED_PORT: number = 6")
	f.check("shared value")
	f.compiled("shared value", 1)

	// edits that reach the dependents
	editTestFile(t, dir, "src/shared/net.ts", "PORT = 1", "PORT = 3")
	f.check("shared literal type")
	if f.session.lastCompiled < 2 {
		t.Errorf("shared literal type: compiled %d files", f.session.lastCompiled)
	}

	editTestFile(t, dir, "src/shared/net.ts", "export function connect(port: number): number", "export function connect(port: number, extra?: string): number")
	f.check("shared shape")
	if f.session.lastCompiled < 3 {
		t.Errorf("shared shape: compiled %d files", f.session.lastCompiled)
	}

	editTestFile(t, dir, "src/shared/types.ts", "id: number;", "id: number;\n\tname?: string;")
	f.check("type-only")
	f.compiled("type-only", 2)

	// the file list
	writeTestFile(t, dir, "src/client/added.ts", "export const ADDED = 1;\n")
	f.check("added file")
	f.compiled("added file", 1)

	editTestFile(t, dir, "src/client/leaf.ts", "export const LEAF", "import { ADDED } from \"./added\";\nprint(ADDED);\nexport const LEAF")
	f.check("import of the added file")
	f.compiled("import of the added file", 1)

	if err := os.Rename(filepath.Join(dir, "src/client/added.ts"), filepath.Join(dir, "src/client/renamed.ts")); err != nil {
		t.Fatal(err)
	}
	result = f.check("renamed file breaks its importer")
	f.wantErrorIn("renamed file breaks its importer", result, "client/leaf.ts")
	editTestFile(t, dir, "src/client/leaf.ts", "./added", "./renamed")
	f.check("importer of the renamed file fixed")

	editTestFile(t, dir, "src/client/leaf.ts", "import { ADDED } from \"./renamed\";\nprint(ADDED);\n", "")
	if err := os.Remove(filepath.Join(dir, "src/client/renamed.ts")); err != nil {
		t.Fatal(err)
	}
	f.check("deleted file")

	// JSON is a program file and an asset
	writeTestFile(t, dir, "src/shared/data.json", `{ "size": 1 }`)
	writeTestFile(t, dir, "src/client/usesdata.ts", "import data from \"shared/data.json\";\nexport const SIZE: number = data.size;\n")
	f.check("added JSON")
	writeTestFile(t, dir, "src/shared/data.json", `{ "size": 2 }`)
	f.check("JSON value")
	writeTestFile(t, dir, "src/shared/data.json", `{ "size": "big" }`)
	result = f.check("JSON that breaks its importer")
	f.wantErrorIn("JSON that breaks its importer", result, "client/usesdata.ts")
	writeTestFile(t, dir, "src/shared/data.json", `{ "size": 3 }`)
	f.check("JSON fixed")
	if err := os.Remove(filepath.Join(dir, "src/shared/data.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "src/client/usesdata.ts")); err != nil {
		t.Fatal(err)
	}
	f.check("deleted JSON")

	// export * order, and what the Rojo walk finds
	writeTestFile(t, dir, "src/shared/star.ts", "export const FIRST = 1;\nexport const SECOND = 2;\n")
	writeTestFile(t, dir, "src/shared/reexport.ts", "export * from \"./star\";\n")
	f.check("export star")
	writeTestFile(t, dir, "src/shared/star.ts", "export const SECOND = 2;\nexport const FIRST = 1;\n")
	f.check("export star, other order")
	if err := os.Remove(filepath.Join(dir, "src/shared/star.ts")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "src/shared/reexport.ts")); err != nil {
		t.Fatal(err)
	}
	f.check("export star deleted")

	// a .d.ts and the options reach every file
	editTestFile(t, dir, "src/shared/globals.d.ts", "declare const GLOBAL_LIMIT: number;", "declare const GLOBAL_LIMIT: number;\ndeclare const GLOBAL_OTHER: number;")
	f.check("changed d.ts")

	editTestFile(t, dir, "src/shared/globals.d.ts", "GLOBAL_LIMIT: number", "GLOBAL_LIMIT: string")
	result = f.check("d.ts that breaks a file")
	f.wantErrorIn("d.ts that breaks a file", result, "shared/net.ts")
	editTestFile(t, dir, "src/shared/globals.d.ts", "GLOBAL_LIMIT: string", "GLOBAL_LIMIT: number")
	f.check("d.ts fixed")

	editTestFile(t, dir, "tsconfig.json", `"allowUnreachableCode": false`, `"allowUnreachableCode": true`)
	f.check("changed tsconfig")
	if f.session.lastCompiled != total {
		t.Errorf("changed tsconfig: compiled %d files, want %d", f.session.lastCompiled, total)
	}

	editTestFile(t, dir, "default.project.json", `"shared": { "$path": "out/shared" }`, `"common": { "$path": "out/shared" }`)
	f.check("changed Rojo project")
	if f.session.lastCompiled != total {
		t.Errorf("changed Rojo project: compiled %d files, want %d", f.session.lastCompiled, total)
	}

	writeTestFile(t, dir, "package.json", `{ "name": "fixture", "version": "1.0.0" }`)
	f.check("changed package.json")

	// an error in a file nobody edited is reported until it is fixed
	editTestFile(t, dir, "src/shared/net.ts", "connect(port: number, extra?: string)", "connect(port: string, extra?: string)")
	result = f.check("error in a dependent")
	f.wantErrorIn("error in a dependent", result, "client/user.client.ts")
	result = f.check("error, built again")
	f.wantErrorIn("error, built again", result, "client/user.client.ts")
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET = 2", "QUIET = 3")
	result = f.check("error, another file edited")
	f.wantErrorIn("error, another file edited", result, "client/user.client.ts")
	editTestFile(t, dir, "src/shared/net.ts", "connect(port: string, extra?: string)", "connect(port: number, extra?: string)")
	result = f.check("error fixed")
	if hasErrors(result.Diagnostics) {
		t.Errorf("error fixed: %v", diagnosticLines(result.Diagnostics))
	}

	// the same in the edited file, fixed with text the last good build hasn't seen
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET = 3", "QUIET: string = 3")
	result = f.check("error in the edited file")
	f.wantErrorIn("error in the edited file", result, "client/quiet.ts")
	f.check("error in the edited file, built again")
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET: string = 3", "QUIET = 4")
	result = f.check("edited file fixed")
	if hasErrors(result.Diagnostics) {
		t.Errorf("edited file fixed: %v", diagnosticLines(result.Diagnostics))
	}

	// and an error reverted, back to the last good state
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET = 4", "QUIET: string = 4")
	f.check("error to revert")
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET: string = 4", "QUIET = 4")
	result = f.check("error reverted")
	if hasErrors(result.Diagnostics) {
		t.Errorf("error reverted: %v", diagnosticLines(result.Diagnostics))
	}

	// the outputs
	writeTestFile(t, dir, "out/client/quiet.luau", "-- not what the compiler wrote\n")
	f.check("changed output")
	f.compiled("changed output", 1)
	if err := os.Remove(filepath.Join(dir, "out/client/quiet.luau")); err != nil {
		t.Fatal(err)
	}
	f.check("deleted output")
	f.compiled("deleted output", 1)
	writeTestFile(t, dir, "out/client/stray.luau", "-- not an output\n")
	f.check("stray output")
	f.compiled("stray output", 0)

	if persist {
		cachePath := filepath.Join(dir, "out", "tsconfig.goblox.tsbuildinfo")
		good := readTestFile(t, dir, "out/tsconfig.goblox.tsbuildinfo")
		for name, text := range map[string]string{
			"empty":          "",
			"truncated":      good[:len(good)/2],
			"old":            "goblox-incremental-0\n" + good[len(incrementalCacheMagic):],
			"not a checksum": incrementalCacheMagic + "abc\n{}",
			"bad JSON": func() string {
				return strings.Replace(good, `"files"`, `"filez"`, 1)
			}(),
		} {
			if err := os.WriteFile(cachePath, []byte(text), 0o644); err != nil {
				t.Fatal(err)
			}
			f.check("cache: " + name)
			if f.session.lastCompiled != total {
				t.Errorf("cache %s: compiled %d files, want a full build of %d", name, f.session.lastCompiled, total)
			}
		}
		if err := os.Remove(cachePath); err != nil {
			t.Fatal(err)
		}
		f.check("cache: missing")
		if f.session.lastCompiled != total {
			t.Errorf("cache missing: compiled %d files, want a full build of %d", f.session.lastCompiled, total)
		}
		f.check("cache: written again")
		f.compiled("cache: written again", 0)
	}
}

// TestEmitFirst: the files without errors are written, the file with the error keeps its output, and the
// errors don't leave anything behind once they are fixed.
func TestEmitFirst(t *testing.T) {
	oracle := os.Getenv("GOBLOX_ORACLE")
	if oracle == "" {
		oracle = defaultOracle
	}
	rbxts := filepath.Join(oracle, "tests", "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(rbxts, "compiler-types")); err != nil {
		t.Skipf("no @rbxts/compiler-types in the oracle (set GOBLOX_ORACLE): %v", err)
	}
	f := newIncrementalFixture(t, rbxts, false, false)
	f.watchPersist = true
	dir := f.dir
	f.check("cold")

	before := readTestFile(t, dir, "out/client/quiet.luau")
	editTestFile(t, dir, "src/client/quiet.ts", "QUIET = 1", "QUIET: string = 1")
	editTestFile(t, dir, "src/server/other.ts", "TYPED_PORT + 1", "TYPED_PORT + 2")
	f.emitFirst = true
	result := f.build(true)
	f.emitFirst = false
	f.wantErrorIn("emit first", result, "client/quiet.ts")
	if !strings.Contains(readTestFile(t, dir, "out/server/other.luau"), "+ 2") {
		t.Errorf("the file without errors was not written:\n%s", readTestFile(t, dir, "out/server/other.luau"))
	}
	if got := readTestFile(t, dir, "out/client/quiet.luau"); got != before {
		t.Errorf("the file with the error was written:\n%s", got)
	}

	editTestFile(t, dir, "src/client/quiet.ts", "QUIET: string = 1", "QUIET = 2")
	f.emitFirst = true
	result = f.build(true)
	f.emitFirst = false
	if hasErrors(result.Diagnostics) {
		t.Errorf("fixed: %v", diagnosticLines(result.Diagnostics))
	}
	f.check("fixed")
	f.compiled("fixed", 0)
}

func TestDecoratorText(t *testing.T) {
	text := "@Service()\nexport class A {\n\t@Inject() b = 1;\n\tm(@Param() p: number) {}\n}\nfunction f() {}\n"
	file := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/a.ts", Path: "/a.ts"}, text, core.ScriptKindTS)
	if got, want := decoratorText(file), "@Service()\n@Inject()\n@Param()\n"; got != want {
		t.Errorf("decoratorText = %q, want %q", got, want)
	}
}
