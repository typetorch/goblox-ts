package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

func writeWatchTestFile(t *testing.T, path string, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	// a later modification time than the snapshot's, also on file systems with coarse times
	later := time.Now().Add(time.Duration(len(text)+1) * time.Second)
	if err := os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}
}

// newWatchTestProject writes a project below <tmp>/game that extends ../base/tsconfig.base.json, next to
// folders the build never reads, and returns the watcher of its first build's inputs.
func newWatchTestProject(t *testing.T) (root string, watcher *fileWatcher) {
	t.Helper()
	root = t.TempDir()
	if long, err := filepath.EvalSymlinks(root); err == nil {
		root = long
	}
	files := map[string]string{
		"base/tsconfig.base.json": `{ "compilerOptions": { "strict": true, "noLib": true } }`,
		"game/package.json":       `{ "name": "game" }`,
		"game/tsconfig.json": `{
	"extends": "../base/tsconfig.base.json",
	"compilerOptions": {
		"allowSyntheticDefaultImports": true, "module": "commonjs", "moduleResolution": "Node",
		"moduleDetection": "force", "target": "ESNext", "typeRoots": ["node_modules/@rbxts"], "types": [],
		"rootDir": "src", "outDir": "out"
	},
	"include": ["src"]
}`,
		"game/default.project.json":                 `{ "name": "game", "tree": { "$path": "out" } }`,
		"game/src/main.server.ts":                   "print(1);\n",
		"game/out/main.server.luau":                 "print(1)\n",
		"game/.claude/worktrees/a/src/x.ts":         "export {};\n",
		"game/.typetorch/cache.json":                "{}\n",
		"game/src/node_modules/@rbxts/x/index.ts":   "export {};\n",
		"game/node_modules/@rbxts/types/index.d.ts": "export {};\n",
	}
	for name, text := range files {
		writeWatchTestFile(t, filepath.Join(root, filepath.FromSlash(name)), text)
	}
	list := &project.WatchList{}
	gameDir := filepath.ToSlash(filepath.Join(root, "game"))
	_, _ = project.Build(t.Context(), project.BuildOptions{
		Project:          gameDir,
		CurrentDirectory: tspath.NormalizePath(gameDir),
		FS:               project.DefaultFS(),
		Overrides:        project.PartialProjectOptions{},
		Watch:            list,
	})
	watcher = &fileWatcher{}
	watcher.setList(*list, false)
	watcher.take()
	return root, watcher
}

func TestWatcherRebuildsOnInputs(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	for _, change := range []struct { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
		name   string
		apply  func(t *testing.T, root string)
		change bool
	}{
		{"extended tsconfig", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "base", "tsconfig.base.json"), `{ "compilerOptions": { "strict": false } }`)
		}, true},
		{"package.json", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "package.json"), `{ "name": "game", "version": "1.0.0" }`)
		}, true},
		{"new lockfile", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "bun.lock"), "{}")
		}, true},
		{"Rojo project", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "default.project.json"), `{ "name": "game2", "tree": { "$path": "out" } }`)
		}, true},
		{"new typetorch.json", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "typetorch.json"), "{}")
		}, true},
		{"source", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "src", "main.server.ts"), "print(2);\n")
		}, true},
		{"new source", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "src", "shared", "util.ts"), "export {};\n")
		}, true},
		{"deleted source", func(t *testing.T, root string) {
			if err := os.Remove(filepath.Join(root, "game", "src", "main.server.ts")); err != nil {
				t.Fatal(err)
			}
		}, true},
		{"agent worktree", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", ".claude", "worktrees", "a", "src", "x.ts"), "export const x = 1;\n")
		}, false},
		{"dot folder", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", ".typetorch", "cache.json"), `{ "a": 1 }`)
		}, false},
		{"unrelated folder", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "docs", "notes.md"), "notes\n")
		}, false},
		{"output", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "out", "main.server.luau"), "print(2)\n")
		}, false},
		{"node_modules", func(t *testing.T, root string) {
			writeWatchTestFile(t, filepath.Join(root, "game", "src", "node_modules", "@rbxts", "x", "index.ts"), "export const x = 1;\n")
		}, false},
	} {
		t.Run(change.name, func(t *testing.T) {
			root, watcher := newWatchTestProject(t)
			if watcher.changed() {
				t.Fatal("changed before any change")
			}
			change.apply(t, root)
			if got := watcher.changed(); got != change.change {
				t.Errorf("changed() = %v, want %v (dirs %v, files %v)", got, change.change, watcher.dirs, watcher.files)
			}
		})
	}
}

func TestWatcherFailedBuildKeepsList(t *testing.T) {
	t.Parallel()
	watcher := &fileWatcher{}
	watcher.setList(project.WatchList{Dirs: []string{"/game/src"}, Files: []string{"/game/tsconfig.json"}}, false)
	watcher.setList(project.WatchList{Files: []string{"/game/tsconfig.json", "/game/package.json"}}, true)
	if len(watcher.dirs) != 1 || len(watcher.files) != 2 {
		t.Errorf("kept dirs %v, files %v", watcher.dirs, watcher.files)
	}
	watcher.setList(project.WatchList{Files: []string{"/game/tsconfig.json"}}, false)
	if len(watcher.dirs) != 0 || len(watcher.files) != 1 {
		t.Errorf("replaced dirs %v, files %v", watcher.dirs, watcher.files)
	}
}

// BenchmarkWatcherPoll is an idle poll of 700 files in 35 folders.
func BenchmarkWatcherPoll(b *testing.B) {
	root := b.TempDir()
	for dir := range 35 {
		for file := range 20 {
			path := filepath.Join(root, "src", fmt.Sprintf("dir%d", dir), fmt.Sprintf("file%d.ts", file))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				b.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("export {};\n"), 0o644); err != nil {
				b.Fatal(err)
			}
		}
	}
	watcher := &fileWatcher{}
	watcher.setList(project.WatchList{Dirs: []string{filepath.ToSlash(filepath.Join(root, "src"))}, Files: []string{filepath.ToSlash(filepath.Join(root, "tsconfig.json"))}}, false)
	watcher.take()
	b.ReportAllocs()
	for b.Loop() {
		if watcher.changed() {
			b.Fatal("changed")
		}
	}
}
