package goblox_test

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/microsoft/typescript-go/pkg/goblox"
)

// reportingPlugin doubles like doublePlugin and reports a diagnostic for every file.
type reportingPlugin struct{ doublePlugin }

func (p reportingPlugin) TransformSourceFile(ctx *goblox.TransformContext, file *goblox.SourceFile) *goblox.SourceFile {
	ctx.AddDiagnostic(goblox.NewDiagnostic(file, 0, 1, goblox.CategoryWarning, "test", "first"))
	result := p.doublePlugin.TransformSourceFile(ctx, file)
	ctx.AddDiagnostic(goblox.NewDiagnostic(file, 0, 1, goblox.CategoryWarning, "test", "second"))
	return result
}

// parallelPlugin hands out forChecker, counting the calls.
type parallelPlugin struct {
	goblox.Plugin
	calls      *atomic.Int32
	forChecker func() goblox.Plugin
}

func (p parallelPlugin) ForChecker() goblox.Plugin {
	p.calls.Add(1)
	return p.forChecker()
}

// panicPlugin panics for the files named in panics.
type panicPlugin struct {
	doublePlugin
	panics map[string]bool
}

func (p panicPlugin) TransformSourceFile(ctx *goblox.TransformContext, file *goblox.SourceFile) *goblox.SourceFile {
	for name := range p.panics {
		if strings.HasSuffix(file.FileName(), name) {
			panic(name)
		}
	}
	return p.doublePlugin.TransformSourceFile(ctx, file)
}

func loadParallelProject(t *testing.T) (*goblox.Program, string) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"tsconfig.json":   `{ "compilerOptions": { "strict": true, "target": "esnext", "module": "commonjs", "noLib": true }, "include": ["src"] }`,
		"src/macros.d.ts": `declare function double(value: number): number;`,
	}
	for i := range 20 {
		files[fmt.Sprintf("src/file%02d.ts", i)] = fmt.Sprintf("export const a = double(%d);\n", i)
	}
	writeFiles(t, dir, files)
	program, errs := goblox.LoadProgram(goblox.LoadOptions{Project: dir, CurrentDirectory: dir, Checkers: 4})
	if program == nil {
		t.Fatalf("LoadProgram: %v", errs)
	}
	return program, dir
}

func TestParallelPluginMatchesSequential(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	program, dir := loadParallelProject(t)
	sequential := goblox.RunPlugins(ctx, program, []goblox.Plugin{reportingPlugin{}}, goblox.PluginOptions{CurrentDirectory: dir})

	var calls atomic.Int32
	plugin := parallelPlugin{Plugin: reportingPlugin{}, calls: &calls, forChecker: func() goblox.Plugin { return reportingPlugin{} }}
	parallel := goblox.RunPlugins(ctx, program, []goblox.Plugin{plugin}, goblox.PluginOptions{CurrentDirectory: dir})

	checkers := map[*goblox.Checker]bool{}
	for _, file := range goblox.CompilableSourceFiles(program) {
		c, done := program.GetTypeCheckerForFile(ctx, file)
		done()
		checkers[c] = true
	}
	if len(checkers) < 2 {
		t.Fatalf("the files all share a checker, so this tests nothing")
	}
	if got := int(calls.Load()); got != len(checkers) {
		t.Fatalf("ForChecker called %d times, want once per checker (%d)", got, len(checkers))
	}
	if len(sequential.Transformed) != 20 || !reflect.DeepEqual(sequential.Transformed, parallel.Transformed) {
		t.Fatalf("transformed files differ:\n%v\n%v", sequential.Transformed, parallel.Transformed)
	}
	describe := func(diagnostics []*goblox.Diagnostic) []string {
		var result []string
		for _, d := range diagnostics {
			result = append(result, d.File().FileName()+":"+d.MessageText())
		}
		return result
	}
	if want, got := describe(sequential.Diagnostics), describe(parallel.Diagnostics); len(want) != 40 || !reflect.DeepEqual(want, got) {
		t.Fatalf("diagnostics differ:\n%v\n%v", want, got)
	}
}

func TestParallelPluginDeclined(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	program, dir := loadParallelProject(t)
	var calls atomic.Int32
	plugin := parallelPlugin{Plugin: doublePlugin{}, calls: &calls, forChecker: func() goblox.Plugin { return nil }}
	result := goblox.RunPlugins(ctx, program, []goblox.Plugin{plugin}, goblox.PluginOptions{CurrentDirectory: dir})
	if len(result.Transformed) != 20 {
		t.Fatalf("transformed %d files, want 20", len(result.Transformed))
	}
}

func TestParallelPluginSubsetOfFiles(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	program, dir := loadParallelProject(t)
	files := goblox.CompilableSourceFiles(program)[5:8]
	var calls atomic.Int32
	plugin := parallelPlugin{Plugin: doublePlugin{}, calls: &calls, forChecker: func() goblox.Plugin { return doublePlugin{} }}
	result := goblox.RunPlugins(ctx, program, []goblox.Plugin{plugin}, goblox.PluginOptions{CurrentDirectory: dir, Files: files})
	if len(result.Transformed) != 3 {
		t.Fatalf("transformed %d files, want 3", len(result.Transformed))
	}
	for _, file := range files {
		if _, ok := result.Transformed[file.FileName()]; !ok {
			t.Fatalf("%s was not transformed: %v", file.FileName(), result.Transformed)
		}
	}
}

func TestParallelPluginPanicsWithFirstFile(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	program, dir := loadParallelProject(t)
	var calls atomic.Int32
	failing := panicPlugin{panics: map[string]bool{"file03.ts": true, "file04.ts": true, "file17.ts": true}}
	plugin := parallelPlugin{Plugin: failing, calls: &calls, forChecker: func() goblox.Plugin { return failing }}
	defer func() {
		if r := recover(); r != "file03.ts" {
			t.Fatalf("panic = %v, want the first file's", r)
		}
	}()
	goblox.RunPlugins(ctx, program, []goblox.Plugin{plugin}, goblox.PluginOptions{CurrentDirectory: dir})
	t.Fatal("expected a panic")
}
