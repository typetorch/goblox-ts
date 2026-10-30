// Package project is the port of roblox-ts's src/Project: loading the tsconfig and the program,
// validating options, and (later) the build loop.
package project

import (
	"context"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/tsc"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

// LoadOptions configures LoadProgram.
type LoadOptions struct {
	// Project is the path to a tsconfig.json or to the folder that contains it.
	Project string
	// CurrentDirectory resolves relative paths. Defaults to the process's working directory.
	CurrentDirectory string
	// FS defaults to the OS file system with the bundled TypeScript libs.
	FS vfs.FS
	// Checkers is the number of type checkers (0 = typescript-go's default of 4). Each checker holds
	// its own copy of the types it has resolved, so fewer checkers use less memory.
	Checkers int
	// SingleThreaded disables parallel parsing, binding and checking.
	SingleThreaded bool
}

type parseHost struct {
	fs  vfs.FS
	cwd string
}

func (h *parseHost) FS() vfs.FS                  { return h.fs }
func (h *parseHost) GetCurrentDirectory() string { return h.cwd }

// ResolveConfigFileName returns the absolute path of the tsconfig.json for project.
func ResolveConfigFileName(fs vfs.FS, cwd string, project string) string {
	configFileName := tspath.GetNormalizedAbsolutePath(project, cwd)
	if fs.DirectoryExists(configFileName) {
		configFileName = tspath.CombinePaths(configFileName, "tsconfig.json")
	}
	return configFileName
}

// DefaultFS is the OS file system with the bundled libs and the TypeScript 7 compatibility rewrites.
func DefaultFS() vfs.FS {
	return WrapCompatFS(bundled.WrapFS(osvfs.FS()))
}

// LoadProgram parses the tsconfig and creates the program. It returns nil and the config errors when
// the tsconfig cannot be read.
func LoadProgram(opts LoadOptions) (*compiler.Program, []*ast.Diagnostic) {
	config, errs := loadConfig(opts)
	if len(errs) > 0 {
		return nil, errs
	}
	return config.newProgram(), nil
}

// loadedConfig is a parsed tsconfig, before the program is created.
type loadedConfig struct {
	host                *parseHost
	configFileName      string
	config              *tsoptions.ParsedCommandLine
	extendedConfigCache *tsc.ExtendedConfigCache
}

func loadConfig(opts LoadOptions) (*loadedConfig, []*ast.Diagnostic) {
	fs := opts.FS
	if fs == nil {
		fs = DefaultFS()
	}
	cwd := opts.CurrentDirectory
	if cwd == "" {
		cwd = tspath.NormalizePath(mustGetwd())
	}
	host := &parseHost{fs: fs, cwd: cwd}
	configFileName := ResolveConfigFileName(fs, cwd, opts.Project)

	extendedConfigCache := &tsc.ExtendedConfigCache{}
	cliOptions := &core.CompilerOptions{}
	if opts.Checkers > 0 {
		checkers := opts.Checkers
		cliOptions.Checkers = &checkers
	}
	if opts.SingleThreaded {
		cliOptions.SingleThreaded = core.TSTrue
	}
	config, errs := tsoptions.GetParsedCommandLineOfConfigFile(configFileName, cliOptions, nil, host, extendedConfigCache)
	if len(errs) > 0 {
		return nil, errs
	}
	if config.CompilerOptions().Types == nil {
		// TypeScript 5 includes every package of the type roots when `types` is not set
		// (getAutomaticTypeDirectiveNames); TypeScript 7 includes none unless `types` has "*". roblox-ts's
		// project templates don't set `types` and rely on @rbxts/compiler-types being global.
		cliOptions.Types = []string{"*"}
		config, errs = tsoptions.GetParsedCommandLineOfConfigFile(configFileName, cliOptions, nil, host, extendedConfigCache)
		if len(errs) > 0 {
			return nil, errs
		}
	}
	applyBaseUrlCompat(config.CompilerOptions(), tspath.GetDirectoryPath(configFileName))
	return &loadedConfig{host: host, configFileName: configFileName, config: config, extendedConfigCache: extendedConfigCache}, nil
}

func (c *loadedConfig) newProgram() *compiler.Program {
	return c.newProgramWithCache(nil)
}

// newProgramWithCache creates the program, reusing the unchanged files of cache when it is not nil.
func (c *loadedConfig) newProgramWithCache(cache *SourceFileCache) *compiler.Program {
	var compilerHost compiler.CompilerHost = compiler.NewCachedFSCompilerHost(c.host.cwd, c.host.fs, bundled.LibPath(), c.extendedConfigCache, nil, nil)
	if cache != nil {
		compilerHost = &cachingCompilerHost{CompilerHost: compilerHost, cache: cache}
	}
	program := compiler.NewProgram(compiler.ProgramOptions{Config: c.config, Host: compilerHost})
	if cache != nil {
		cache.prune(program)
	}
	return program
}

// GetConfigDiagnostics returns the config, syntactic and options diagnostics of the program, minus
// the ones FilterCompatDiagnostics removes.
func GetConfigDiagnostics(ctx context.Context, program *compiler.Program) []*ast.Diagnostic {
	var diags []*ast.Diagnostic
	diags = append(diags, program.GetConfigFileParsingDiagnostics()...)
	diags = append(diags, program.GetSyntacticDiagnostics(ctx, nil)...)
	diags = append(diags, program.GetProgramDiagnostics()...)
	return FilterCompatDiagnostics(diags)
}

// GetPreEmitDiagnostics is `ts.getPreEmitDiagnostics(program, sourceFile)` as roblox-ts calls it for
// each file it compiles: the global, syntactic and semantic diagnostics of sourceFile (or of every
// file when sourceFile is nil).
func GetPreEmitDiagnostics(ctx context.Context, program *compiler.Program, sourceFile *ast.SourceFile) []*ast.Diagnostic {
	var diags []*ast.Diagnostic
	diags = append(diags, program.GetSyntacticDiagnostics(ctx, sourceFile)...)
	diags = append(diags, program.GetGlobalDiagnostics(ctx)...)
	diags = append(diags, program.GetSemanticDiagnostics(ctx, sourceFile)...)
	return FilterCompatDiagnostics(compiler.SortAndDeduplicateDiagnostics(diags))
}
