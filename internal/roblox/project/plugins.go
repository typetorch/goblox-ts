package project

import (
	"context"
	"maps"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/printer"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// The plugin pass of src/Project/functions/compileFiles.ts ("running transformers..") and
// src/Project/transformers/createTransformerWatcher.ts.

// TransformContext is what a plugin gets for each source file it transforms.
type TransformContext struct {
	Program     *compiler.Program
	Checker     *checker.Checker
	Options     *core.CompilerOptions
	EmitContext *printer.EmitContext
	Factory     *printer.NodeFactory
	// CurrentDirectory is the directory of the project's tsconfig.json.
	CurrentDirectory string
	// ProjectType is roblox-ts' `--type` option ("game", "model", "package"), empty when not passed.
	ProjectType string
	// Verbose is roblox-ts' `--verbose`.
	Verbose bool
	// AddDiagnostic reports a diagnostic (errors fail the compile).
	AddDiagnostic func(*ast.Diagnostic)
}

// Plugin is a before-transform run over every source file of the program, in place of a ts-patch
// style JS transformer (`{ "transform": "..." }` in tsconfig's plugins).
type Plugin interface {
	// Name is how the plugin is referred to in diagnostics and logs.
	Name() string
	// TransformSourceFile returns the transformed file (or the file itself when nothing changed).
	TransformSourceFile(ctx *TransformContext, file *ast.SourceFile) *ast.SourceFile
}

// ParallelPlugin is a Plugin that can run on several files at once, one goroutine per checker.
type ParallelPlugin interface {
	Plugin
	// ForChecker returns the plugin that transforms the files of one checker, always from a single goroutine and in
	// the program's file order. RunPlugins calls it once per checker, before it starts any of them. The plugins of
	// different checkers run concurrently, so they can't share state unsynchronized, and what they output can't depend
	// on what the others did. Returning nil runs every plugin on one goroutine, as if this method didn't exist.
	ForChecker() Plugin
}

// PluginOptions configures RunPlugins.
type PluginOptions struct {
	CurrentDirectory string
	ProjectType      string
	Verbose          bool
	// Files are the files to transform. Defaults to CompilableSourceFiles(program).
	Files []*ast.SourceFile
	// PrintUnchanged prints and parses again the files no plugin changed too.
	PrintUnchanged bool
	// Reused maps the file name of files outside of Files to the text the plugins printed for them in an
	// earlier build (incremental builds).
	Reused map[string]string
	// ParseTransformed, when set, parses the printed text of a changed file (incremental builds).
	ParseTransformed func(opts ast.SourceFileParseOptions, text string) *ast.SourceFile
}

// PluginResult is the outcome of RunPlugins.
type PluginResult struct {
	// Program is the program to compile: the input program when no plugin changed anything, otherwise
	// a new program in which every changed file was printed and parsed again, so its nodes have
	// symbols and types. Unchanged files are the input program's SourceFile objects.
	Program *compiler.Program
	// Transformed maps the file name of every changed file to its printed text.
	Transformed map[string]string
	// Diagnostics are the diagnostics the plugins reported.
	Diagnostics []*ast.Diagnostic
}

// CompilableSourceFiles are the program's .ts/.tsx files outside of node_modules (roblox-ts's
// isCompilableFile, applied to the program's files).
func CompilableSourceFiles(program *compiler.Program) []*ast.SourceFile {
	var files []*ast.SourceFile
	for _, file := range program.GetSourceFiles() {
		if file.IsDeclarationFile || program.IsSourceFileFromExternalLibrary(file) || program.IsSourceFileDefaultLibrary(file.Path()) {
			continue
		}
		if !tspath.FileExtensionIsOneOf(file.FileName(), []string{tspath.ExtensionTs, tspath.ExtensionTsx}) {
			continue
		}
		files = append(files, file)
	}
	return files
}

// RunPlugins runs every plugin over every file, in order, like `ts.transformNodes` with the plugins'
// `before` transformers. A file's nodes may be replaced by several plugins. When every plugin is a
// ParallelPlugin the files of each checker run on their own goroutine (a checker gets the same queries
// in the same order as it would on one goroutine); the result doesn't depend on the scheduling.
//
// Transformed files are printed and parsed again (roblox-ts does the same: "transformed nodes don't
// have symbol or type information, there's no way to rebind an existing file, so we have to reprint
// it"). Positions in a changed file therefore refer to the printed text, not the original source.
// roblox-ts reprints the files no plugin changed as well. Here those keep their parsed and bound
// SourceFile, unless reprinting them would change their Luau (see reprintKeepsBlockComments).
func RunPlugins(ctx context.Context, program *compiler.Program, plugins []Plugin, opts PluginOptions) PluginResult {
	result := PluginResult{Program: program}
	if len(plugins) == 0 {
		return result
	}
	files := opts.Files
	if files == nil {
		files = CompilableSourceFiles(program)
	}

	fileResults := make([]pluginFileResult, len(files))
	if !runPluginsPerChecker(ctx, program, plugins, opts, files, fileResults) {
		for i, file := range files {
			c, done := program.GetTypeCheckerForFile(ctx, file)
			runPluginsOnFile(program, c, plugins, opts, file, &fileResults[i])
			done()
		}
	}

	result.Transformed = map[string]string{}
	for i, file := range files {
		fileResult := &fileResults[i]
		if fileResult.panicked {
			panic(fileResult.panicValue)
		}
		result.Diagnostics = append(result.Diagnostics, fileResult.diagnostics...)
		if fileResult.printed {
			result.Transformed[file.FileName()] = fileResult.text
		}
	}

	maps.Copy(result.Transformed, opts.Reused)

	if len(result.Transformed) == 0 {
		return result
	}
	host := &proxyHost{CompilerHost: program.Host(), program: program, transformed: result.Transformed, parse: opts.ParseTransformed}
	result.Program = compiler.NewProgram(compiler.ProgramOptions{Config: program.CommandLine(), Host: host})
	return result
}

// pluginFileResult is what the plugins did to one file.
type pluginFileResult struct {
	diagnostics []*ast.Diagnostic
	// printed is whether text has the file's printed text.
	printed bool
	text    string
	// panicked is whether a plugin panicked with panicValue. RunPlugins panics with it again, for the first such file.
	panicked   bool
	panicValue any
}

// runPluginsPerChecker runs the plugins with one goroutine per checker, false when not every plugin can.
func runPluginsPerChecker(ctx context.Context, program *compiler.Program, plugins []Plugin, opts PluginOptions, files []*ast.SourceFile, results []pluginFileResult) bool {
	for _, plugin := range plugins {
		if _, ok := plugin.(ParallelPlugin); !ok {
			return false
		}
	}

	type group struct {
		plugins []Plugin
		indices []int
	}
	var groups []*group
	byChecker := map[*checker.Checker]*group{}
	for i, file := range files {
		c, done := program.GetTypeCheckerForFile(ctx, file)
		done()
		g := byChecker[c]
		if g == nil {
			g = &group{}
			for _, plugin := range plugins {
				forChecker := plugin.(ParallelPlugin).ForChecker()
				if forChecker == nil {
					return false
				}
				g.plugins = append(g.plugins, forChecker)
			}
			byChecker[c] = g
			groups = append(groups, g)
		}
		g.indices = append(g.indices, i)
	}

	var wg sync.WaitGroup
	for _, g := range groups {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, done := program.GetTypeCheckerForFileExclusive(ctx, files[g.indices[0]])
			defer done()
			for _, i := range g.indices {
				result := &results[i]
				func() {
					defer func() {
						if r := recover(); r != nil {
							result.panicked = true
							result.panicValue = r
						}
					}()
					runPluginsOnFile(program, c, g.plugins, opts, files[i], result)
				}()
				if result.panicked {
					// the plugins' state is unknown now, and this file's panic is the one that counts for the rest
					return
				}
			}
		}()
	}
	wg.Wait()
	return true
}

// runPluginsOnFile runs the plugins over one file and prints it if needed.
func runPluginsOnFile(program *compiler.Program, c *checker.Checker, plugins []Plugin, opts PluginOptions, file *ast.SourceFile, result *pluginFileResult) {
	emitContext := printer.NewEmitContext()
	tc := &TransformContext{
		Program:          program,
		Checker:          c,
		Options:          program.Options(),
		EmitContext:      emitContext,
		Factory:          emitContext.Factory,
		CurrentDirectory: opts.CurrentDirectory,
		ProjectType:      opts.ProjectType,
		Verbose:          opts.Verbose,
		AddDiagnostic:    func(d *ast.Diagnostic) { result.diagnostics = append(result.diagnostics, d) },
	}
	transformed := file
	for _, plugin := range plugins {
		transformed = plugin.TransformSourceFile(tc, transformed)
	}
	if transformed != file || opts.PrintUnchanged || !reprintKeepsBlockComments(file.Text()) {
		p := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, emitContext)
		result.printed = true
		result.text = removeNonNullAccessParens(file.FileName(), transformed, p.EmitSourceFile(transformed))
	}
}

// proxyHost serves the printed text of transformed files and the original program's (already parsed
// and bound) SourceFile objects for every other file.
type proxyHost struct {
	compiler.CompilerHost
	program     *compiler.Program
	transformed map[string]string
	parse       func(opts ast.SourceFileParseOptions, text string) *ast.SourceFile
}

func (h *proxyHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if text, ok := h.transformed[opts.FileName]; ok {
		if h.parse != nil {
			return h.parse(opts, text)
		}
		return parser.ParseSourceFile(opts, text, core.EnsureScriptKindFromFileName(opts.FileName))
	}
	if file := h.program.GetSourceFileByPath(opts.Path); file != nil && file.ParseOptions() == opts {
		return file
	}
	return h.CompilerHost.GetSourceFile(opts)
}
