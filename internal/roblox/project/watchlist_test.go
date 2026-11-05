package project

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/tspath"
)

// newWatchFixture writes a project below <tmp>/game that extends <tmp>/base/tsconfig.base.json, has typings
// outside rootDir and a nested Rojo project outside src, and folders the build never reads (.claude,
// .typetorch, docs). No @rbxts packages: the program has errors, but the build reads everything first.
func newWatchFixture(t *testing.T) (root string, project string) {
	t.Helper()
	root = t.TempDir()
	if long, err := filepath.EvalSymlinks(root); err == nil {
		root = long
	}
	root = filepath.ToSlash(root)
	project = root + "/game"
	writeTestFile(t, root, "base/tsconfig.base.json", `{ "compilerOptions": { "strict": true, "noLib": true } }`)
	for name, text := range map[string]string{
		"package.json": `{ "name": "game" }`,
		"bun.lock":     "{}",
		"tsconfig.json": `{
	"extends": "../base/tsconfig.base.json",
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"module": "commonjs",
		"moduleResolution": "Node",
		"moduleDetection": "force",
		"target": "ESNext",
		"typeRoots": ["node_modules/@rbxts"],
		"types": [],
		"rootDir": "src",
		"outDir": "out"
	},
	"include": ["src", "typings"]
}
`,
		"default.project.json": `{ "name": "game", "tree": { "$className": "DataModel",
	"ReplicatedStorage": { "$className": "ReplicatedStorage", "shared": { "$path": "out" }, "ui": { "$path": "packages/ui.project.json" } } } }`,
		"packages/ui.project.json":         `{ "name": "ui", "tree": { "$path": "assets" } }`,
		"src/main.server.ts":               "print(GLOBAL);\n",
		"src/shared/data.json":             "{}\n",
		"typings/globals.d.ts":             "declare const GLOBAL: number;\n",
		".claude/worktrees/a/src/x.ts":     "export {};\n",
		".typetorch/cache/state.json":      "{}\n",
		"docs/readme.md":                   "docs\n",
		"node_modules/@rbxts/x/index.d.ts": "export {};\n",
	} {
		writeTestFile(t, project, name, text)
	}
	return root, project
}

func buildWatchList(t *testing.T, project string) *WatchList {
	t.Helper()
	watch := &WatchList{}
	_, _ = Build(t.Context(), BuildOptions{
		Project:          project,
		CurrentDirectory: tspath.NormalizePath(project),
		FS:               DefaultFS(),
		Overrides:        PartialProjectOptions{},
		Watch:            watch,
	})
	return watch
}

func TestWatchList(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	root, project := newWatchFixture(t)
	watch := buildWatchList(t, project)

	if !slices.Equal(watch.Dirs, []string{project + "/src"}) {
		t.Errorf("Dirs = %v, want only src", watch.Dirs)
	}
	for _, want := range []string{project + "/out", project + "/include"} {
		if !slices.Contains(watch.Ignored, want) {
			t.Errorf("Ignored has no %s: %v", want, watch.Ignored)
		}
	}
	for _, want := range []string{
		project + "/tsconfig.json",
		root + "/base/tsconfig.base.json",
		project + "/default.project.json",
		project + "/packages/ui.project.json",
		project + "/typetorch.json",
		project + "/package.json",
		project + "/bun.lock",
		// missing, but read: a monorepo's package.json and lockfile can be above the project
		root + "/package.json",
		root + "/pnpm-lock.yaml",
		project + "/typings/globals.d.ts",
	} {
		if !slices.Contains(watch.Files, want) {
			t.Errorf("Files has no %s", want)
		}
	}
	for _, file := range watch.Files {
		for _, unread := range []string{"/src/", "/.claude/", "/.typetorch/", "/docs/", "/node_modules/", "/out/"} {
			if strings.Contains(strings.TrimPrefix(file, project), unread) {
				t.Errorf("Files has %s, which is under %s", file, unread)
			}
		}
	}
}

func TestWatchListFailedBuilds(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	root, project := newWatchFixture(t)

	// an extended config that does not parse is still watched, so fixing it rebuilds
	writeTestFile(t, root, "base/tsconfig.base.json", `{ "compilerOptions": { "strict": tru`)
	watch := buildWatchList(t, project)
	for _, want := range []string{project + "/tsconfig.json", root + "/base/tsconfig.base.json", project + "/package.json"} {
		if !slices.Contains(watch.Files, want) {
			t.Errorf("broken extended config: Files has no %s: %v", want, watch.Files)
		}
	}

	// without a tsconfig, every tsconfig.json the build looked for
	if err := os.Remove(filepath.Join(project, "tsconfig.json")); err != nil {
		t.Fatal(err)
	}
	watch = buildWatchList(t, project+"/src")
	for _, want := range []string{project + "/src/tsconfig.json", project + "/tsconfig.json", root + "/tsconfig.json"} {
		if !slices.Contains(watch.Files, want) {
			t.Errorf("no tsconfig: Files has no %s: %v", want, watch.Files)
		}
	}
	if len(watch.Dirs) != 0 {
		t.Errorf("no tsconfig: Dirs = %v", watch.Dirs)
	}
}
