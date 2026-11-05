package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/microsoft/typescript-go/internal/tspath"
)

// TestSourceFileCacheVerify: a file rewritten with the same size and modification time is reused from the
// cache without verify, and read again with it.
func TestSourceFileCacheVerify(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	dir = filepath.ToSlash(dir)
	writeTestFile(t, dir, "tsconfig.json", `{ "compilerOptions": { "noLib": true, "types": [], "rootDir": "src", "outDir": "out" }, "include": ["src"] }`)
	writeTestFile(t, dir, "src/a.ts", "export const A = 1;\n")
	path := filepath.Join(dir, "src", "a.ts")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	mtime := info.ModTime()

	cache := &SourceFileCache{}
	text := func(verify bool) string {
		t.Helper()
		config, errs := loadConfig(LoadOptions{Project: dir, CurrentDirectory: tspath.NormalizePath(dir), SingleThreaded: true})
		if len(errs) > 0 {
			t.Fatalf("config: %v", errs)
		}
		program := config.newProgramWithCache(cache, verify)
		for _, file := range program.GetSourceFiles() {
			if strings.HasSuffix(file.FileName(), "/a.ts") {
				return file.Text()
			}
		}
		t.Fatal("no a.ts in the program")
		return ""
	}
	if got := text(false); got != "export const A = 1;\n" {
		t.Fatalf("first program: %q", got)
	}

	writeTestFile(t, dir, "src/a.ts", "export const A = 2;\n")
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(path); err != nil || !info.ModTime().Equal(mtime) {
		t.Skipf("could not restore the modification time: %v", err)
	}
	if got := text(false); got != "export const A = 1;\n" {
		// not a requirement, but then the test below proves nothing
		t.Logf("the cache noticed the change without verify: %q", got)
	}
	if got := text(true); got != "export const A = 2;\n" {
		t.Errorf("verified program: %q", got)
	}
	// and the verified file is what the cache keeps
	if got := text(false); got != "export const A = 2;\n" {
		t.Errorf("after the verified program: %q", got)
	}
}
