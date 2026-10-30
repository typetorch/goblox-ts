package goblox

import (
	"context"
	"io"

	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/diagnosticwriter"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// LoadOptions configures LoadProgram.
//
//	type LoadOptions struct {
//		Project          string // tsconfig.json or the folder that contains it
//		CurrentDirectory string // defaults to the working directory
//		FS               vfs.FS // defaults to the OS file system with the bundled libs and the TS 7 compat rewrites
//		Checkers         int    // number of type checkers, 0 = min(4, physical cores); 4 with 8+ threads
//		SingleThreaded   bool
//	}
type LoadOptions = project.LoadOptions

// LoadProgram parses the tsconfig and creates the program. It returns nil and the config errors when
// the tsconfig cannot be read. Files from @rbxts/compiler-types are patched for TypeScript 7 on read
// (see internal/roblox/project/compat.go).
func LoadProgram(opts LoadOptions) (*Program, []*Diagnostic) {
	return project.LoadProgram(opts)
}

// GetConfigDiagnostics returns the config, syntactic and compiler options diagnostics of the program.
func GetConfigDiagnostics(ctx context.Context, program *Program) []*Diagnostic {
	return project.GetConfigDiagnostics(ctx, program)
}

// GetPreEmitDiagnostics is TypeScript's `ts.getPreEmitDiagnostics(program, sourceFile)`; nil means
// every file.
func GetPreEmitDiagnostics(ctx context.Context, program *Program, sourceFile *SourceFile) []*Diagnostic {
	return project.GetPreEmitDiagnostics(ctx, program, sourceFile)
}

// PluginOptions configures RunPlugins.
//
//	type PluginOptions struct {
//		CurrentDirectory string
//		ProjectType      string
//		Verbose          bool
//		Files            []*SourceFile // defaults to CompilableSourceFiles(program)
//	}
type PluginOptions = project.PluginOptions

// PluginResult is the outcome of RunPlugins.
//
//	type PluginResult struct {
//		Program     *Program          // the program to compile (re-parsed when a plugin changed a file)
//		Transformed map[string]string // file name -> printed text of every changed file
//		Diagnostics []*Diagnostic     // reported through TransformContext.AddDiagnostic
//	}
type PluginResult = project.PluginResult

// RunPlugins runs every plugin over every file, in order, then prints the changed files and parses
// them into a new program (see the package documentation).
func RunPlugins(ctx context.Context, program *Program, plugins []Plugin, opts PluginOptions) PluginResult {
	return project.RunPlugins(ctx, program, plugins, opts)
}

// CompilableSourceFiles are the program's .ts/.tsx files outside of node_modules.
func CompilableSourceFiles(program *Program) []*SourceFile {
	return project.CompilableSourceFiles(program)
}

// SortAndDeduplicateDiagnostics sorts diagnostics by file and position and removes duplicates.
func SortAndDeduplicateDiagnostics(diags []*Diagnostic) []*Diagnostic {
	return compiler.SortAndDeduplicateDiagnostics(diags)
}

// WriteDiagnostics writes diagnostics in tsc's plain `file(line,col): error TSxxxx: message` format,
// with paths relative to currentDirectory.
func WriteDiagnostics(w io.Writer, diags []*Diagnostic, currentDirectory string) {
	formatOpts := &diagnosticwriter.FormattingOptions{
		NewLine:             "\n",
		ComparePathsOptions: tspath.ComparePathsOptions{CurrentDirectory: currentDirectory, UseCaseSensitiveFileNames: true},
	}
	for _, d := range diags {
		diagnosticwriter.WriteFormatDiagnostic(w, diagnosticwriter.WrapASTDiagnostic(d), formatOpts)
	}
}

// PluginEntries returns the tsconfig's `compilerOptions.plugins` entries as plain Go values
// (map[string]any, []any, string, float64, bool, nil), taken from the nearest config of the `extends`
// chain that sets `plugins`.
func PluginEntries(program *Program) []map[string]any {
	return project.PluginEntries(program)
}

// PluginFactory creates a plugin from its tsconfig entry (`{ "transform": "<name>", ...options }`).
type PluginFactory = project.PluginFactory

// RegisterPlugin makes a Go plugin available to the tsconfig `plugins` entries whose `transform` is
// name, the npm package of the JavaScript transformer it replaces (e.g. "@typetorch/transformer").
// Call it from an init function; the goblox binary runs the plugins a project's tsconfig lists.
func RegisterPlugin(name string, create PluginFactory) {
	project.RegisterPlugin(name, create)
}

// RegisteredPlugins are the names passed to RegisterPlugin, sorted.
func RegisteredPlugins() []string {
	return project.RegisteredPlugins()
}

// PluginsForProgram creates the registered plugins for the program's tsconfig `plugins` entries, in
// order. Entries without `transform` (language service plugins) are skipped; a transform without a
// registered plugin is an error.
func PluginsForProgram(program *Program) ([]Plugin, []error) {
	return project.PluginsForProgram(program)
}
