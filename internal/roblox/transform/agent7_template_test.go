package transform_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// The sources and expected Luau of these tests are files under testdata/agent7, copied byte for byte
// from a build of roblox-ts 68648b0, so escapes and backticks need no Go quoting. Each <name>.ts compiles
// to the oracle's <name>.luau.
//
// The project needs @rbxts/compiler-types and @rbxts/types (MacroManager looks up their symbols), which
// this repository does not vendor: they are copied from the oracle's test project (GOBLOX_ORACLE, see docs/goblox/PORTING.md),
// and the test is skipped when that is not available.

const defaultOracle = `C:/Users/phasenull/.t3/scratch/2026-10-29-act-as-the-implementation-sub-37576c8e/oracle`

var agent7Fixtures = []string{
	"templatePlain",        // NoSubstitutionTemplateLiteral: escapes, braces, line continuations
	"templateSubstitution", // TemplateExpression: spans, empty heads and tails, ensureTransformOrder prereqs
	"templateTagged",       // TaggedTemplateExpression: strings array, cooked text, property tags
	"misc",                 // conditional, non-null/as/satisfies/type assertion, await, yield and yield*
}

func copyDir(t *testing.T, from string, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestTemplateAndMiscExpressions(t *testing.T) {
	oracle := os.Getenv("GOBLOX_ORACLE")
	if oracle == "" {
		oracle = defaultOracle
	}
	rbxts := filepath.Join(oracle, "tests", "node_modules", "@rbxts")
	packages := []string{"compiler-types", "types"}
	for _, name := range packages {
		if _, err := os.Stat(filepath.Join(rbxts, name)); err != nil {
			t.Skipf("no @rbxts/%s in the oracle (set GOBLOX_ORACLE): %v", name, err)
		}
	}

	dir := t.TempDir()
	// expand 8.3 short names (C:/Users/PHASEN~1) so paths match the ones the program sees, as cmd/goblox-fixtures does
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	for _, name := range packages {
		copyDir(t, filepath.Join(rbxts, name), filepath.Join(dir, "node_modules", "@rbxts", name))
	}
	writeFile(t, filepath.Join(dir, "package.json"), `{ "name": "agent7", "version": "1.0.0" }`)
	writeFile(t, filepath.Join(dir, "tsconfig.json"), `{
	"include": ["src"],
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"rootDir": "src",
		"outDir": "out"
	}
}`)
	writeFile(t, filepath.Join(dir, "default.project.json"), `{
	"name": "agent7",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"include": { "$path": "include" },
			"src": { "$path": "out" }
		}
	}
}`)
	for _, name := range agent7Fixtures {
		source, err := os.ReadFile(filepath.Join("testdata", "agent7", name+".ts"))
		if err != nil {
			t.Fatal(err)
		}
		writeFile(t, filepath.Join(dir, "src", name+".ts"), string(source))
	}

	result, err := project.Build(context.Background(), project.BuildOptions{
		Project:          tspath.NormalizePath(dir),
		CurrentDirectory: tspath.NormalizePath(dir),
		KeepGoing:        true,
		Checkers:         1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Diagnostics {
		t.Errorf("diagnostic: %s", diagnostic.String())
	}
	for file, panicValue := range result.Panics {
		t.Errorf("%s panicked: %s", file, panicValue)
	}

	for _, name := range agent7Fixtures {
		t.Run(name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", "agent7", name+".luau"))
			if err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(filepath.Join(dir, "out", name+".luau"))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(want) {
				t.Errorf("got:\n%s\nwant:\n%s", got, want)
			}
		})
	}
}
