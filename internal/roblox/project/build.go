package project

// Port of CLI/commands/build.ts (findTsConfigPath), Project/classes/ProjectBuild.ts,
// Project/functions/getParsedCommandLine.ts, createPathTranslator.ts, getProjectOutputs.ts and
// Shared/util/findAncestorDir.ts, for a single project: goblox does not build project references and
// has no watch mode or incremental builds yet, so every build is a cold build of the root project.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/outputpaths"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

// BuildOptions configures Build.
type BuildOptions struct {
	// Project is the `-p` flag: a tsconfig.json, or a folder at or below the one that contains it.
	// Defaults to ".".
	Project string
	// CurrentDirectory resolves relative paths. Defaults to the process's working directory.
	CurrentDirectory string
	// FS defaults to DefaultFS(). Outputs are always written to the OS file system.
	FS vfs.FS
	// Overrides are the explicit CLI options; they win over the tsconfig's "rbxts" options.
	Overrides PartialProjectOptions
	// KeepGoing transforms and writes every file without errors, instead of stopping at the first file
	// with an error as roblox-ts does. The fixture harness uses it so one unported construct does not
	// hide the results of the other files.
	KeepGoing bool
	// Checkers and SingleThreaded are as in LoadOptions.
	Checkers       int
	SingleThreaded bool
	// SourceFileCache, when set, keeps parsed and bound source files between builds (watch mode).
	SourceFileCache *SourceFileCache
	// Incremental, when set, compiles only what changed since the last good build of the session.
	Incremental *IncrementalSession
	// EmitFirst writes the outputs of the files without errors and reports the errors afterwards, instead of
	// blocking the emit (`--emitFirst`, watch mode only). What a build with errors writes is not part of
	// the incremental state, so those files are compiled again by the next build.
	EmitFirst bool
}

// CLIError is CLI/errors/CLIError.ts.
type CLIError struct{ Message string }

func (e *CLIError) Error() string { return e.Message }

// findTsConfigPath is build.ts findTsConfigPath.
func findTsConfigPath(fs vfs.FS, cwd string, projectPath string) (string, error) {
	tsConfigPath := nodepath.Resolve(cwd, projectPath)
	if !fs.FileExists(tsConfigPath) {
		// ts.findConfigFile
		found := ""
		for directory := tsConfigPath; ; {
			candidate := nodepath.Join(directory, "tsconfig.json")
			if fs.FileExists(candidate) {
				found = candidate
				break
			}
			parent := nodepath.Dirname(directory)
			if parent == directory {
				break
			}
			directory = parent
		}
		if found == "" {
			return "", &CLIError{Message: "Unable to find tsconfig.json!"}
		}
		tsConfigPath = found
	}
	return nodepath.Resolve(cwd, tsConfigPath), nil
}

// findAncestorDir is Shared/util/findAncestorDir.ts.
func findAncestorDir(dirs []string) string {
	for i, dir := range dirs {
		dir = nodepath.Normalize(dir)
		if !strings.HasSuffix(dir, "/") {
			dir += "/"
		}
		dirs[i] = dir
	}
	currentDir := dirs[0]
	for {
		all := true
		for _, dir := range dirs {
			if !strings.HasPrefix(dir, currentDir) {
				all = false
				break
			}
		}
		if all {
			return currentDir
		}
		// keep a directory boundary so src-extra cannot match src
		currentDir = nodepath.Join(currentDir, "..", "/")
	}
}

// projectPathKey is ProjectGraph.ts projectPathKey.
func projectPathKey(fs vfs.FS, filePath string) string {
	return tspath.GetCanonicalFileName(nodepath.Normalize(filePath), fs.UseCaseSensitiveFileNames())
}

// createPathTranslator is the path translator of ProjectGraph.ts for a parsed config.
func createPathTranslator(config *tsoptions.ParsedCommandLine, luau bool) *rojo.PathTranslator {
	compilerOptions := config.CompilerOptions()
	rootDir := findAncestorDir(append([]string{config.CommonSourceDirectory()}, getRootDirs(compilerOptions)...))
	buildInfoPath := outputpaths.GetBuildInfoFileName(compilerOptions, tspath.ComparePathsOptions{
		UseCaseSensitiveFileNames: config.UseCaseSensitiveFileNames(),
		CurrentDirectory:          config.GetCurrentDirectory(),
	})
	if buildInfoPath != "" {
		buildInfoPath = nodepath.Normalize(buildInfoPath)
	}
	return rojo.NewPathTranslator(rootDir, nodepath.Normalize(compilerOptions.OutDir), buildInfoPath, compilerOptions.GetEmitDeclarations(), luau)
}

// isSolutionConfig is ts.isSolutionConfig.
func isSolutionConfig(config *tsoptions.ParsedCommandLine) bool {
	return len(config.FileNames()) == 0 && len(config.ProjectReferences()) > 0
}

// mapReferencePaths is ProjectGraph.ts mapReferencePaths for the root project: the import path of every
// file of the (transitively) referenced projects. Referenced projects are not built; build them first.
func mapReferencePaths(fs vfs.FS, cwd string, root *tsoptions.ParsedCommandLine, projectOptions func(own PartialProjectOptions) transform.ProjectOptions) (map[string]string, error) {
	paths := map[string]string{}
	visited := map[string]bool{}

	var visit func(configPath string) error
	visit = func(configPath string) error {
		key := projectPathKey(fs, configPath)
		if visited[key] {
			return nil
		}
		visited[key] = true

		loaded, errs := loadConfig(LoadOptions{Project: configPath, CurrentDirectory: cwd, FS: fs})
		if len(errs) > 0 {
			return &diagnostics.DiagnosticError{Diagnostics: errs}
		}
		config := loaded.config
		for _, reference := range config.ResolvedProjectReferencePaths() {
			if err := visit(reference); err != nil {
				return err
			}
		}
		if isSolutionConfig(config) {
			return nil
		}

		own, err := readProjectOptions(loaded.host, configPath, map[string]PartialProjectOptions{})
		if err != nil {
			return err
		}
		compilerOptions := config.CompilerOptions()
		if compilerOptions.Composite.IsTrue() && compilerOptions.RootDir == "" {
			compilerOptions.RootDir = nodepath.Dirname(configPath)
		}
		translator := createPathTranslator(config, projectOptions(own).Luau)

		for _, fileName := range config.FileNames() {
			if tspath.IsDeclarationFileName(fileName) && !transform.IsPathDescendantOf(fileName, translator.RootDir) {
				continue
			}

			outputPath := translator.GetImportPath(fileName, false)
			paths[projectPathKey(fs, fileName)] = outputPath

			if !tspath.IsDeclarationFileName(fileName) && !strings.HasSuffix(fileName, ".json") {
				declarationPath := outputpaths.GetOutputDeclarationFileNameWorker(fileName, compilerOptions, config)
				paths[projectPathKey(fs, declarationPath)] = outputPath
			}
		}
		return nil
	}

	for _, reference := range root.ResolvedProjectReferencePaths() {
		if err := visit(reference); err != nil {
			return nil, err
		}
	}
	return paths, nil
}

// projectOutputs is getProjectOutputs.ts ProjectOutputs.
type projectOutputs struct {
	roots []string
	// files maps the key of every output path the build owns to its input ("" for owned outputs
	// without an input, such as build info).
	files  map[string]string
	assets map[string]asset
}

type asset struct{ input, output string }

func getOutputRoots(pathTranslator *rojo.PathTranslator, compilerOptions *core.CompilerOptions) []string {
	roots := []string{pathTranslator.OutDir}
	if compilerOptions.GetEmitDeclarations() && compilerOptions.DeclarationDir != "" {
		declarationDir := nodepath.Normalize(compilerOptions.DeclarationDir)
		if declarationDir != roots[0] {
			roots = append(roots, declarationDir)
		}
	}
	return roots
}

func validateProjectOutputs(pathTranslator *rojo.PathTranslator, compilerOptions *core.CompilerOptions) error {
	for _, root := range getOutputRoots(pathTranslator, compilerOptions) {
		for _, source := range getRootDirs(compilerOptions) {
			if transform.IsPathDescendantOf(source, root) {
				return &diagnostics.ProjectError{Message: `Output directory "` + root + `" contains source directory "` + source + `".`}
			}
		}
	}
	return nil
}

var deploymentConfigRegex = regexp.MustCompile(`^.+\.project\.json$`)

func getProjectOutputs(fs vfs.FS, program *compiler.Program, data *transform.ProjectData, pathTranslator *rojo.PathTranslator, configPaths map[string]bool) (*projectOutputs, error) {
	compilerOptions := program.Options()
	outputs := &projectOutputs{
		roots:  getOutputRoots(pathTranslator, compilerOptions),
		files:  map[string]string{},
		assets: map[string]asset{},
	}

	var err error
	addOutput := func(output string, input string) {
		key := projectPathKey(fs, output)
		if previous, ok := outputs.files[key]; ok && previous != input && err == nil {
			err = &diagnostics.ProjectError{Message: `Files "` + previous + `" and "` + input + `" both emit to "` + output + `".`}
		}
		outputs.files[key] = input
	}

	// imports can pull sources into the program even when files or exclude omit them
	for _, sourceFile := range program.GetSourceFiles() {
		if !program.SourceFileMayBeEmitted(sourceFile, false) {
			continue
		}
		fileName := sourceFile.FileName()
		if strings.HasSuffix(fileName, ".json") {
			continue
		}
		addOutput(pathTranslator.GetOutputPath(fileName), fileName)
		if data.ProjectOptions.WriteTransformedFiles && len(PluginEntries(program)) > 0 {
			addOutput(pathTranslator.GetOutputTransformedPath(fileName), fileName)
		}
		if pathTranslator.Declaration {
			addOutput(pathTranslator.GetOutputDeclarationPath(fileName), fileName)
		}
	}

	excluded := outputs.roots
	if !isPackageProject(data) {
		excluded = append(excluded[:len(excluded):len(excluded)], data.ProjectOptions.IncludePath)
	}
	buildInfoPath, cachePath := "", ""
	if pathTranslator.BuildInfoOutputPath != "" {
		buildInfoPath = projectPathKey(fs, pathTranslator.BuildInfoOutputPath)
		cachePath = projectPathKey(fs, incrementalCachePath(pathTranslator.BuildInfoOutputPath))
	}

	var walk func(input string)
	walk = func(input string) {
		for _, root := range excluded {
			if transform.IsPathDescendantOf(input, root) {
				return
			}
		}
		info, statErr := os.Stat(input)
		if statErr != nil {
			return
		}
		if info.IsDir() {
			if nodepath.Basename(input) == "node_modules" || nodepath.Basename(input) == ".git" {
				return
			}
			entries, _ := os.ReadDir(input)
			for _, entry := range entries {
				walk(nodepath.Join(input, entry.Name()))
			}
		} else {
			// a composite root can include deployment configs beside tsconfig, not just source assets
			isDeploymentConfig := nodepath.Dirname(input) == data.ProjectPath && deploymentConfigRegex.MatchString(nodepath.Basename(input))
			if configPaths[projectPathKey(fs, input)] || (buildInfoPath != "" && (projectPathKey(fs, input) == buildInfoPath || projectPathKey(fs, input) == cachePath)) || isDeploymentConfig {
				return
			}
			if strings.HasSuffix(input, ".ts") || strings.HasSuffix(input, ".tsx") {
				if !tspath.IsDeclarationFileName(input) || !pathTranslator.Declaration {
					return
				}
			}
			checkFileName(input)
			output := pathTranslator.GetOutputPath(input)
			addOutput(output, input)
			outputs.assets[projectPathKey(fs, output)] = asset{input, output}
		}
	}
	for _, root := range getRootDirs(compilerOptions) {
		walk(root)
	}

	// runtimes can be placed inside an output directory, including at its root
	if !isPackageProject(data) {
		for _, file := range getIncludeFiles(&data.ProjectOptions) {
			addOutput(file.output, "include:"+file.input)
		}
	}
	if pathTranslator.BuildInfoOutputPath != "" {
		outputs.files[buildInfoPath] = ""
		outputs.files[cachePath] = ""
	}
	if data.RojoConfigPath != "" {
		key := projectPathKey(fs, data.RojoConfigPath)
		if _, ok := outputs.files[key]; !ok {
			outputs.files[key] = ""
		}
	}
	return outputs, err
}

func isPackageProject(data *transform.ProjectData) bool {
	return data.ProjectOptions.Type == transform.ProjectTypePackage || (data.ProjectOptions.Type == "" && data.IsPackage)
}

func syncProjectOutputs(fs vfs.FS, outputs *projectOutputs, writeOnlyChanged bool) error {
	for _, a := range outputs.assets {
		if tspath.IsDeclarationFileName(a.input) {
			// TODO: transformDeclarationFile
			LogService.Warn("goblox does not copy declaration files yet: " + a.input)
			continue
		}
		contents, err := os.ReadFile(a.input)
		if err != nil {
			return err
		}
		if writeOnlyChanged {
			if existing, err := os.ReadFile(a.output); err == nil && string(existing) == string(contents) {
				continue
			}
		}
		if err := outputFile(a.output, contents); err != nil {
			return err
		}
	}

	var clean func(directory string) error
	clean = func(directory string) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return nil
		}
		for _, entry := range entries {
			if entry.Name() == ".git" {
				continue
			}
			output := nodepath.Join(directory, entry.Name())
			info, err := os.Lstat(output)
			if err != nil {
				continue
			}
			if info.IsDir() {
				if err := clean(output); err != nil {
					return err
				}
				if rest, _ := os.ReadDir(output); len(rest) == 0 {
					if err := os.Remove(output); err != nil {
						return err
					}
				}
			} else if _, ok := outputs.files[projectPathKey(fs, output)]; !ok {
				if err := os.RemoveAll(output); err != nil {
					return err
				}
			}
		}
		return nil
	}
	for _, root := range outputs.roots {
		if err := clean(root); err != nil {
			return err
		}
	}
	return nil
}

// Build is `rbxtsc build` for one project. It returns an error for the failures roblox-ts reports as a
// LoggableError (CLIError, ProjectError); everything else is in the result's diagnostics.
func Build(ctx context.Context, opts BuildOptions) (*EmitResult, error) {
	fs := opts.FS
	if fs == nil {
		fs = DefaultFS()
	}
	cwd := opts.CurrentDirectory
	if cwd == "" {
		cwd = tspath.NormalizePath(mustGetwd())
	}
	projectPath := opts.Project
	if projectPath == "" {
		projectPath = "."
	}
	tsConfigPath, err := findTsConfigPath(fs, cwd, projectPath)
	if err != nil {
		return nil, err
	}

	result, err := build(ctx, fs, cwd, tsConfigPath, opts)
	var diagnosticError *diagnostics.DiagnosticError
	if errors.As(err, &diagnosticError) {
		return &EmitResult{EmitSkipped: true, Diagnostics: diagnosticError.Diagnostics, WatchRoot: nodepath.Dirname(tsConfigPath)}, nil
	}
	if err != nil {
		return nil, err
	}
	result.WatchRoot = nodepath.Dirname(tsConfigPath)
	result.Diagnostics = compiler.SortAndDeduplicateDiagnostics(result.Diagnostics)
	result.EmitSkipped = hasErrors(result.Diagnostics)
	return result, nil
}

func build(ctx context.Context, fs vfs.FS, cwd string, tsConfigPath string, opts BuildOptions) (result *EmitResult, err error) {
	defer func() {
		// ProjectError and DiagnosticError thrown from deep inside the transform helpers
		if r := recover(); r != nil {
			switch e := r.(type) {
			case *diagnostics.ProjectError:
				err = e
			case *diagnostics.DiagnosticError:
				err = e
			default:
				panic(r)
			}
		}
	}()

	// roblox-ts does not time this; goblox does (reported once --verbose is known), since watch rebuilds pay it every time
	setupStart := time.Now()
	loaded, errs := loadConfig(LoadOptions{
		Project:          tsConfigPath,
		CurrentDirectory: cwd,
		FS:               fs,
		Checkers:         opts.Checkers,
		SingleThreaded:   opts.SingleThreaded,
	})
	if len(errs) > 0 {
		return nil, &diagnostics.DiagnosticError{Diagnostics: errs}
	}

	// parseProjectConfig: raw.rbxts
	ownOptions, err := readProjectOptions(loaded.host, tsConfigPath, map[string]PartialProjectOptions{})
	if err != nil {
		return nil, err
	}
	projectOptions := transform.DEFAULT_PROJECT_OPTIONS
	ApplyProjectOptions(&projectOptions, ownOptions)
	ApplyProjectOptions(&projectOptions, opts.Overrides)
	// options of a referenced project: `{ ...DEFAULT_PROJECT_OPTIONS, ...inheritedOptions, ...config.raw.rbxts, ...referenceOverrides }`
	referenceOptions := func(own PartialProjectOptions) transform.ProjectOptions {
		options := transform.DEFAULT_PROJECT_OPTIONS
		ApplyProjectOptions(&options, ownOptions)
		ApplyProjectOptions(&options, own)
		ApplyProjectOptions(&options, opts.Overrides)
		return options
	}
	LogService.Verbose = projectOptions.Verbose

	data, err := createProjectData(fs, tsConfigPath, projectOptions)
	if err != nil {
		return nil, err
	}

	// getParsedCommandLine
	compilerOptions := loaded.config.CompilerOptions()
	if compilerOptions.Composite.IsTrue() && compilerOptions.RootDir == "" {
		// composite projects default to the config directory, rather than the common source directory
		compilerOptions.RootDir = nodepath.Dirname(data.TsConfigPath)
	}
	if err := validateCompilerOptions(fs, compilerOptions, data.ProjectPath); err != nil {
		return nil, err
	}

	configPaths := map[string]bool{projectPathKey(fs, tsConfigPath): true}
	for _, extended := range loaded.config.ExtendedSourceFiles() {
		configPaths[projectPathKey(fs, extended)] = true
	}

	data.ProjectReferencePaths, err = mapReferencePaths(fs, cwd, loaded.config, referenceOptions)
	if err != nil {
		return nil, err
	}

	LogService.WriteLineIfVerbose("Building project: " + data.TsConfigPath)
	if LogService.Verbose {
		LogService.Write("parse tsconfig and project data")
		benchmarkEnd(setupStart)
	}
	var program *compiler.Program
	benchmarkIfVerbose("create program", func() {
		program = loaded.newProgramWithCache(opts.SourceFileCache)
		// parse and bind now, so the steps below are timed apart from them
		program.BindSourceFiles()
	})
	pathTranslator := createPathTranslator(loaded.config, data.ProjectOptions.Luau)
	if err := validateProjectOutputs(pathTranslator, program.Options()); err != nil {
		return nil, err
	}

	benchmarkIfVerbose("copy include files", func() { err = copyInclude(data) })
	if err != nil {
		return nil, err
	}

	var outputs *projectOutputs
	benchmarkIfVerbose("collect project outputs", func() {
		outputs, err = getProjectOutputs(fs, program, data, pathTranslator, configPaths)
	})
	if err != nil {
		return nil, err
	}

	var sourceFiles []*ast.SourceFile
	for _, sourceFile := range program.GetSourceFiles() {
		if program.SourceFileMayBeEmitted(sourceFile, false) && !sourceFile.IsDeclarationFile && !isJsonSourceFile(sourceFile) {
			sourceFiles = append(sourceFiles, sourceFile)
		}
	}

	var inc *incrementalBuild
	if incrementalEligible(opts, data, program.Options()) {
		benchmarkIfVerbose("compare with the last build", func() {
			inc = opts.Incremental.begin(fs, loaded, data, pathTranslator, program, sourceFiles, outputs)
		})
	}

	// compileFiles checks the rebound plugin output, which can differ from the original TypeScript
	keepGoing := opts.KeepGoing || opts.EmitFirst
	result = compileFiles(ctx, fs, program, data, pathTranslator, sourceFiles, keepGoing, inc)
	result.OutputRoots = append(getOutputRoots(pathTranslator, program.Options()), data.ProjectOptions.IncludePath)
	if !result.EmitSkipped || keepGoing {
		benchmarkIfVerbose("copy non-compiled files", func() {
			err = syncProjectOutputs(fs, outputs, data.ProjectOptions.WriteOnlyChanged || inc != nil)
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func isJsonSourceFile(sourceFile *ast.SourceFile) bool {
	return sourceFile.ScriptKind == core.ScriptKindJSON || strings.EqualFold(filepath.Ext(sourceFile.FileName()), ".json")
}
