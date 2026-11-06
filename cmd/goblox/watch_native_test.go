package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/microsoft/typescript-go/internal/fswatch"
)

// waitForWake is whether n wakes the loop within the deadline.
func waitForWake(n *nativeWatch, deadline time.Duration) bool {
	select {
	case <-n.wake:
		return true
	case <-time.After(deadline):
		return false
	}
}

// drainWakes empties n's wake signal and waits out events still on their way.
func drainWakes(n *nativeWatch) {
	for {
		if !waitForWake(n, 300*time.Millisecond) {
			return
		}
	}
}

func TestNativeWatchWakes(t *testing.T) {
	t.Parallel()
	if !fswatch.Default().Available() {
		t.Skip("no native file watcher on this platform")
	}
	root := t.TempDir()
	src := filepath.Join(root, "src")
	out := filepath.Join(root, "out")
	modules := filepath.Join(src, "node_modules")
	for _, dir := range []string{src, out, modules} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	tsconfig := filepath.Join(root, "tsconfig.json")
	for _, file := range []string{tsconfig, filepath.Join(src, "main.ts")} {
		if err := os.WriteFile(file, []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	w := &fileWatcher{}
	w.dirs = []string{src}
	w.files = []string{tsconfig}
	w.ignored = []string{out}
	n := newNativeWatch(false)
	defer n.close()
	n.follow(w)
	if !n.active() {
		t.Skip("the native watcher could not watch the temporary folder")
	}
	drainWakes(n)

	if err := os.WriteFile(filepath.Join(src, "main.ts"), []byte("export {}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitForWake(n, 5*time.Second) {
		t.Error("an edited source file did not wake the watcher")
	}
	drainWakes(n)

	if err := os.WriteFile(tsconfig, []byte(`{ "compilerOptions": {} }`), 0o644); err != nil {
		t.Fatal(err)
	}
	if !waitForWake(n, 5*time.Second) {
		t.Error("an edited tsconfig did not wake the watcher")
	}
	drainWakes(n)

	if err := os.WriteFile(filepath.Join(modules, "dep.ts"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if waitForWake(n, time.Second) {
		t.Error("a change in node_modules woke the watcher")
	}

	if n.pollInterval() != nativePollInterval || n.debounce() != nativeDebounce {
		t.Errorf("timings with native events: poll %v, debounce %v", n.pollInterval(), n.debounce())
	}
}

func TestNativeWatchDisabled(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	w := &fileWatcher{}
	w.dirs = []string{root}
	n := newNativeWatch(true)
	defer n.close()
	n.follow(w)
	if n.active() {
		t.Error("--usePolling still watches with the OS")
	}
	if n.pollInterval() != watchPollInterval || n.debounce() != watchDebounce {
		t.Errorf("timings without native events: poll %v, debounce %v", n.pollInterval(), n.debounce())
	}
}

func TestNativeWatchIgnoredPath(t *testing.T) {
	t.Parallel()
	out := filepath.Join(string(filepath.Separator)+"game", "out")
	game := string(filepath.Separator) + "game"
	paths := &watchPaths{dirs: []string{filepath.Join(game, "src")}, files: []string{filepath.Join(game, "tsconfig.json")}, ignored: []string{out}}
	for _, c := range []struct {
		path string
		want bool
	}{
		{filepath.Join(out, "main.luau"), true},
		{out, true},
		{filepath.Join(game, "outside.ts"), false},
		{filepath.Join(game, "out2", "a.ts"), false},
		{filepath.Join(game, "node_modules", "x"), true},
		{filepath.Join(game, "src", ".git", "x"), true},
		{filepath.Join(game, "src", "main.ts"), false},
	} {
		if got := paths.ignoredPath(c.path); got != c.want {
			t.Errorf("ignoredPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestNativeWatchFilters(t *testing.T) {
	t.Parallel()
	game := string(filepath.Separator) + "game"
	paths := &watchPaths{
		dirs:    []string{filepath.Join(game, "src")},
		files:   []string{filepath.Join(game, "tsconfig.json"), filepath.Join(game, "package.json"), filepath.Join(string(filepath.Separator), "package.json")},
		ignored: []string{filepath.Join(game, "src", "out")},
	}
	for _, c := range []struct {
		path      string
		watched   bool
		inProject bool
	}{
		{filepath.Join(game, "src", "main.ts"), true, true},
		{filepath.Join(game, "src", "a", "b.ts"), true, true},
		{filepath.Join(game, "src", "out", "main.luau"), false, true},
		{filepath.Join(game, "src", "node_modules", "x.ts"), false, true},
		{filepath.Join(game, "tsconfig.json"), true, true},
		{filepath.Join(game, "notes.txt"), false, true},
		{filepath.Join(game, ".claude", "x.json"), false, true},
		{filepath.Join(string(filepath.Separator), "package.json"), true, false},
		{filepath.Join(string(filepath.Separator), "other.txt"), false, false},
	} {
		if got := paths.watched(c.path); got != c.watched {
			t.Errorf("watched(%q) = %v, want %v", c.path, got, c.watched)
		}
		if got := paths.inProject(c.path); got != c.inProject {
			t.Errorf("inProject(%q) = %v, want %v", c.path, got, c.inProject)
		}
	}
}
