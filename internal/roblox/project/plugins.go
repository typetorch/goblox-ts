package project

import (
	"context"
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

// PluginOptions configures RunPlugins.
type PluginOptions struct {
	CurrentDirectory string
	ProjectType      string
	Verbose          bool
	// Files are the files to transform. Defaults to CompilableSourceFiles(program).
	Files []*ast.SourceFile
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
// `before` transformers. Plugins run sequentially; a file's nodes may be replaced by several plugins.
//
// Transformed files are printed and parsed again (roblox-ts does the same: "transformed nodes don't
// have symbol or type information, there's no way to rebind an existing file, so we have to reprint
// it"). Positions in a changed file therefore refer to the printed text, not the original source.
func RunPlugins(ctx context.Context, program *compiler.Program, plugins []Plugin, opts PluginOptions) PluginResult {
	result := PluginResult{Program: program}
	if len(plugins) == 0 {
		return result
	}
	files := opts.Files
	if files == nil {
		files = CompilableSourceFiles(program)
	}

	var diagnosticsMu sync.Mutex
	addDiagnostic := func(d *ast.Diagnostic) {
		diagnosticsMu.Lock()
		result.Diagnostics = append(result.Diagnostics, d)
		diagnosticsMu.Unlock()
	}

	result.Transformed = map[string]string{}
	for _, file := range files {
		c, done := program.GetTypeCheckerForFile(ctx, file)
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
			AddDiagnostic:    addDiagnostic,
		}
		transformed := file
		for _, plugin := range plugins {
			transformed = plugin.TransformSourceFile(tc, transformed)
		}
		done()
		if transformed != file {
			p := printer.NewPrinter(printer.PrinterOptions{NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, emitContext)
			result.Transformed[file.FileName()] = p.EmitSourceFile(transformed)
		}
	}

	if len(result.Transformed) == 0 {
		return result
	}
	host := &proxyHost{CompilerHost: program.Host(), program: program, transformed: result.Transformed}
	result.Program = compiler.NewProgram(compiler.ProgramOptions{Config: program.CommandLine(), Host: host})
	return result
}

// proxyHost serves the printed text of transformed files and the original program's (already parsed
// and bound) SourceFile objects for every other file.
type proxyHost struct {
	compiler.CompilerHost
	program     *compiler.Program
	transformed map[string]string
}

func (h *proxyHost) GetSourceFile(opts ast.SourceFileParseOptions) *ast.SourceFile {
	if text, ok := h.transformed[opts.FileName]; ok {
		return parser.ParseSourceFile(opts, text, core.EnsureScriptKindFromFileName(opts.FileName))
	}
	if file := h.program.GetSourceFileByPath(opts.Path); file != nil && file.ParseOptions() == opts {
		return file
	}
	return h.CompilerHost.GetSourceFile(opts)
}
