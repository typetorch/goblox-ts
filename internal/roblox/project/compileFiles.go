package project

// Port of Project/functions/compileFiles.ts, checkRojoConfig.ts, checkFileName.ts,
// createNodeModulesPathMapping.ts and Project/preEmitDiagnostics/fileUsesCommentDirectives.ts.
//
// roblox-ts transforms the files one after the other with a single checker. goblox transforms them in
// parallel, one goroutine per checker of the program (each with its own MultiTransformState and
// MacroManager, since types and merged symbols belong to one checker), then replays the per-file
// results in file order so the reported diagnostics and written files match the sequential build.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
	"github.com/microsoft/typescript-go/internal/vfs/cachedvfs"
)

// EmitResult is ts.EmitResult, plus the files whose transform panicked.
type EmitResult struct {
	EmitSkipped bool
	// WatchRoot is the directory of the project's tsconfig.json, and OutputRoots are the directories the
	// build writes (outDir, the include folder, ...). Watch mode polls WatchRoot without OutputRoots.
	WatchRoot    string
	OutputRoots  []string
	Diagnostics  []*ast.Diagnostic
	EmittedFiles []string
	// Panics maps the file name of every file whose transform panicked (an unported stub or an internal
	// error) to the panic value and stack. Those files also get an error diagnostic.
	Panics map[string]string
	// needsPrintedPositions: see compileFilesPrinting
	needsPrintedPositions bool
}

// FILENAME_WARNINGS is Shared/constants.ts FILENAME_WARNINGS.
var FILENAME_WARNINGS = func() map[string]string {
	m := map[string]string{}
	for _, scriptType := range []string{".server", ".client", ""} {
		for _, fileType := range []string{".ts", ".tsx", ".d.ts"} {
			m["init"+scriptType+fileType] = "index" + scriptType + fileType
		}
	}
	return m
}()

func checkFileName(filePath string) {
	baseName := nodepath.Basename(filePath)
	if nameWarning, ok := FILENAME_WARNINGS[baseName]; ok {
		transform.DiagnosticService.AddDiagnostic(diagnostics.Errors.IncorrectFileName(baseName, nameWarning, filePath))
	}
}

func checkRojoConfig(data *transform.ProjectData, rojoResolver *rojo.RojoResolver, rootDirs []string, inputFileNames []string, pathTranslator *rojo.PathTranslator) {
	if data.RojoConfigPath != "" {
		// inspect all inputs, including imports, independently of the files changed in this build
		var inputs []string
		for _, fileName := range inputFileNames {
			for _, rootDir := range rootDirs {
				if transform.IsPathDescendantOf(fileName, rootDir) {
					inputs = append(inputs, fileName)
					break
				}
			}
		}
		for _, partition := range rojoResolver.GetPartitions() {
			// only flag partitions that would sync compiler input
			for _, fileName := range inputs {
				if transform.IsPathDescendantOf(fileName, partition.FsPath) {
					rojoConfigDir := nodepath.Dirname(data.RojoConfigPath)
					outPath := pathTranslator.GetOutputPath(partition.FsPath)

					inputPath := nodepath.Relative(rojoConfigDir, partition.FsPath)
					suggestedPath := nodepath.Relative(rojoConfigDir, outPath)
					transform.DiagnosticService.AddDiagnostic(diagnostics.Errors.RojoPathInSrc(inputPath, suggestedPath))
					break
				}
			}
		}
	}
}

func createNodeModulesPathMapping(fs vfs.FS, typeRoots []string) map[string]string {
	nodeModulesPathMapping := map[string]string{}
	// go through each org
	for _, scopePath := range typeRoots {
		if !fs.DirectoryExists(scopePath) {
			continue
		}
		// map module paths
		entries := fs.GetAccessibleEntries(scopePath)
		for _, pkgName := range append(entries.Directories, entries.Files...) {
			pkgPath := nodepath.Join(scopePath, pkgName)
			pkgJsonPath := nodepath.Join(pkgPath, "package.json")
			text, ok := fs.ReadFile(pkgJsonPath)
			if !ok {
				continue
			}
			var pkgJson struct {
				Main    *string `json:"main"`
				Typings *string `json:"typings"`
				Types   *string `json:"types"`
			}
			if json.Unmarshal([]byte(text), &pkgJson) != nil {
				continue
			}
			// both "types" and "typings" are valid
			typesPath := "index.d.ts"
			if pkgJson.Types != nil {
				typesPath = *pkgJson.Types
			} else if pkgJson.Typings != nil {
				typesPath = *pkgJson.Typings
			}
			if pkgJson.Main != nil && *pkgJson.Main != "" {
				key := tspath.GetCanonicalFileName(nodepath.Resolve(pkgPath, typesPath), fs.UseCaseSensitiveFileNames())
				nodeModulesPathMapping[key] = nodepath.Resolve(pkgPath, *pkgJson.Main)
			}
		}
	}
	return nodeModulesPathMapping
}

func fileUsesCommentDirectives(data *transform.ProjectData, sourceFile *ast.SourceFile) []*ast.Diagnostic {
	if data.ProjectOptions.AllowCommentDirectives {
		return nil
	}

	var result []*ast.Diagnostic

	for _, commentDirective := range sourceFile.CommentDirectives {
		result = append(result, diagnostics.Errors.NoCommentDirectives.CreateWithRange(sourceFile, commentDirective.Loc))
	}

	for _, pragma := range sourceFile.Pragmas {
		if pragma.Name == "ts-nocheck" {
			result = append(result, diagnostics.Errors.NoCommentDirectives.CreateWithRange(sourceFile, pragma.TextRange))
		}
	}

	return result
}

func getCustomPreEmitDiagnostics(data *transform.ProjectData, sourceFile *ast.SourceFile) []*ast.Diagnostic {
	return fileUsesCommentDirectives(data, sourceFile)
}

func inferProjectType(data *transform.ProjectData, rojoResolver *rojo.RojoResolver) transform.ProjectType {
	if data.IsPackage {
		return transform.ProjectTypePackage
	} else if rojoResolver.IsGame {
		return transform.ProjectTypeGame
	}
	return transform.ProjectTypeModel
}

func emitResultFailure(messageText string) *EmitResult {
	return &EmitResult{
		EmitSkipped: true,
		Diagnostics: []*ast.Diagnostic{diagnostics.CreateTextDiagnostic(messageText, tsdiagnostics.CategoryError)},
	}
}

func getRootDirs(compilerOptions *core.CompilerOptions) []string {
	if compilerOptions.RootDir != "" {
		return []string{compilerOptions.RootDir}
	}
	return compilerOptions.RootDirs
}

// fileResult is what the parallel phase computes for one file.
type fileResult struct {
	preEmitDiagnostics   []*ast.Diagnostic
	transformDiagnostics []*ast.Diagnostic
	source               string
	panic                string
}

// getPreEmitDiagnostics is `ts.getPreEmitDiagnostics(program, sourceFile)` for every file, checking
// the files in parallel. configDiagnostics are the program-wide config and options diagnostics.
func getPreEmitDiagnostics(ctx context.Context, program *compiler.Program, files []*ast.SourceFile, configDiagnostics []*ast.Diagnostic) [][]*ast.Diagnostic {
	semantic := program.GetSemanticDiagnosticsWithoutNoEmitFiltering(ctx, files)
	global := program.GetGlobalDiagnostics(ctx)
	result := make([][]*ast.Diagnostic, len(files))
	for i, file := range files {
		var diags []*ast.Diagnostic
		diags = append(diags, configDiagnostics...)
		diags = append(diags, program.GetSyntacticDiagnostics(ctx, file)...)
		diags = append(diags, global...)
		diags = append(diags, compiler.FilterNoEmitSemanticDiagnostics(semantic[file], program.Options())...)
		diags = append(diags, program.GetIncludeProcessorDiagnostics(file)...)
		result[i] = FilterCompatDiagnostics(compiler.SortAndDeduplicateDiagnostics(diags))
	}
	return result
}

func panicMessage(value any) string {
	return fmt.Sprintf("%v\n%s", value, debug.Stack())
}

// transformFiles transforms and renders files, one goroutine per checker. results[i] must already hold
// the pre-emit diagnostics of files[i]; files with pre-emit errors are not transformed.
func transformFiles(ctx context.Context, program *compiler.Program, renderTime *atomic.Int64, newState func(c *checker.Checker, services *transform.TransformServices, mts *transform.MultiTransformState, file *ast.SourceFile) *transform.TransformState, files []*ast.SourceFile, results []*fileResult) {
	type group struct {
		checker *checker.Checker
		indices []int
	}
	var groups []*group
	byChecker := map[*checker.Checker]*group{}
	for i, file := range files {
		if hasErrors(results[i].preEmitDiagnostics) {
			continue
		}
		c, done := program.GetTypeCheckerForFile(ctx, file)
		done()
		g := byChecker[c]
		if g == nil {
			g = &group{checker: c}
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

			var services *transform.TransformServices
			var servicesPanic string
			func() {
				defer func() {
					if r := recover(); r != nil {
						servicesPanic = panicValueMessage(r)
					}
				}()
				services = transform.CreateTransformServices(c)
			}()
			multiTransformState := transform.NewMultiTransformState()

			for _, i := range g.indices {
				file := files[i]
				result := results[i]
				if servicesPanic != "" {
					result.panic = servicesPanic
					continue
				}
				func() {
					defer func() {
						if r := recover(); r != nil {
							result.panic = panicValueMessage(r)
						}
					}()
					state := newState(c, services, multiTransformState, file)
					luauAST := transform.TransformSourceFile(state, file)
					result.transformDiagnostics = transform.DiagnosticService.TakeFileDiagnostics(file)
					if !hasErrors(result.transformDiagnostics) {
						renderStart := time.Now()
						result.source = luau.RenderAST(luauAST)
						renderTime.Add(int64(time.Since(renderStart)))
					}
				}()
				if result.panic != "" {
					// drop the partial diagnostics of a failed transform
					transform.DiagnosticService.TakeFileDiagnostics(file)
				}
			}
		}()
	}
	wg.Wait()
}

func panicValueMessage(r any) string {
	switch e := r.(type) {
	case *diagnostics.ProjectError:
		return e.Message
	}
	return panicMessage(r)
}

func hasErrors(ds []*ast.Diagnostic) bool {
	for _, d := range ds {
		if d.Category() == tsdiagnostics.CategoryError {
			return true
		}
	}
	return false
}

// compileFiles 'transpiles' TypeScript project into a logically identical Luau project.
//
// writes rendered Luau source to the out directory.
func compileFiles(ctx context.Context, fs vfs.FS, program *compiler.Program, data *transform.ProjectData, pathTranslator *rojo.PathTranslator, sourceFiles []*ast.SourceFile, keepGoing bool) *EmitResult {
	result := compileFilesPrinting(ctx, fs, program, data, pathTranslator, sourceFiles, keepGoing, data.ProjectOptions.WriteTransformedFiles)
	if result.needsPrintedPositions {
		result = compileFilesPrinting(ctx, fs, program, data, pathTranslator, sourceFiles, keepGoing, true)
	}
	return result
}

// compileFilesPrinting compiles the files. rbxtsc reports a diagnostic the compile finds at its position
// in the printed text of the file, so a diagnostic in a file that wasn't printed (printUnchanged is off
// and no plugin changed it) can't be reported as is: the result then only has needsPrintedPositions, and
// nothing was written. Builds without such a diagnostic, nearly all of them, don't pay for the prints.
func compileFilesPrinting(ctx context.Context, fs vfs.FS, program *compiler.Program, data *transform.ProjectData, pathTranslator *rojo.PathTranslator, sourceFiles []*ast.SourceFile, keepGoing bool, printUnchanged bool) *EmitResult {
	compilerOptions := program.Options()

	outDir := compilerOptions.OutDir

	rojoStart := time.Now()
	// The resolvers walk their directories and ask for the real path of every entry several times
	// (Realpath opens the file on Windows). Cache those answers while the resolvers are built; the cache
	// is dropped before the transform, so later queries and watch rebuilds see the file system as it is.
	rojoFS := cachedvfs.From(fs)
	var rojoResolver *rojo.RojoResolver
	if data.RojoConfigPath != "" {
		rojoResolver = rojo.FromPath(rojoFS, data.RojoConfigPath)
	} else {
		rojoResolver = rojo.Synthetic(rojoFS, outDir)
	}

	for _, warning := range rojoResolver.GetWarnings() {
		LogService.Warn(warning)
	}

	var inputFileNames []string
	for _, sourceFile := range program.GetSourceFiles() {
		if !sourceFile.IsDeclarationFile && !program.IsSourceFileFromExternalLibrary(sourceFile) {
			inputFileNames = append(inputFileNames, sourceFile.FileName())
		}
	}
	benchmarkIfVerbose("check rojo config and file names", func() {
		checkRojoConfig(data, rojoResolver, getRootDirs(compilerOptions), inputFileNames, pathTranslator)

		for _, sourceFile := range program.GetSourceFiles() {
			if !strings.HasPrefix(nodepath.Normalize(sourceFile.FileName()), data.NodeModulesPath) {
				checkFileName(sourceFile.FileName())
			}
		}
	})

	var pkgRojoResolvers []*rojo.RojoResolver
	for _, typeRoot := range compilerOptions.TypeRoots {
		pkgRojoResolvers = append(pkgRojoResolvers, rojo.Synthetic(rojoFS, typeRoot))
	}
	nodeModulesPathMapping := createNodeModulesPathMapping(fs, compilerOptions.TypeRoots)
	rojoFS.DisableAndClearCache()
	if LogService.Verbose {
		LogService.Write("load rojo configs and package paths")
		benchmarkEnd(rojoStart)
	}

	projectType := data.ProjectOptions.Type
	if projectType == "" {
		projectType = inferProjectType(data, rojoResolver)
	}

	if projectType != transform.ProjectTypePackage && data.RojoConfigPath == "" {
		return emitResultFailure("Non-package projects must have a Rojo project file!")
	}

	var runtimeLibRbxPath rojo.RbxPath
	if projectType != transform.ProjectTypePackage {
		runtimeLibRbxPath = rojoResolver.GetRbxPathFromFilePath(nodepath.Join(data.ProjectOptions.IncludePath, "RuntimeLib.lua"))
		if runtimeLibRbxPath == nil {
			return emitResultFailure("Rojo project contained no data for include folder!")
		} else if rojoResolver.GetNetworkType(runtimeLibRbxPath) != rojo.NetworkTypeUnknown {
			return emitResultFailure("Runtime library cannot be in a server-only or client-only container!")
		} else if rojoResolver.IsIsolated(runtimeLibRbxPath) {
			return emitResultFailure("Runtime library cannot be in an isolated container!")
		}
	}

	if transform.DiagnosticService.HasErrors() {
		return &EmitResult{EmitSkipped: true, Diagnostics: transform.DiagnosticService.Flush()}
	}

	LogService.WriteLineIfVerbose("compiling as " + string(projectType) + "..")

	proxyProgram := program
	var transformed map[string]string
	if len(PluginEntries(program)) > 0 {
		plugins, errs := PluginsForProgram(program)
		for _, err := range errs {
			if message, isError := pluginErrorMessage(err); message != "" {
				category := tsdiagnostics.CategoryWarning
				if isError {
					category = tsdiagnostics.CategoryError
				}
				transform.DiagnosticService.AddDiagnostic(diagnostics.CreateTextDiagnostic(message, category))
			}
		}
		data.TransformersRan = len(plugins) > 0
		if len(plugins) > 0 {
			var result PluginResult
			benchmarkIfVerbose("running transformers..", func() {
				result = RunPlugins(ctx, program, plugins, PluginOptions{
					CurrentDirectory: data.ProjectPath,
					ProjectType:      string(data.ProjectOptions.Type),
					Verbose:          data.ProjectOptions.Verbose,
					Files:            sourceFiles,
					PrintUnchanged:   printUnchanged,
				})
			})
			if len(result.Transformed) > 0 {
				LogService.WriteLineIfVerbose(fmt.Sprintf("%d/%d files re-printed", len(result.Transformed), len(sourceFiles)))
			}
			transform.DiagnosticService.AddDiagnostics(result.Diagnostics)
			transformed = result.Transformed
			if data.ProjectOptions.WriteTransformedFiles {
				for fileName, source := range result.Transformed {
					if err := outputFile(pathTranslator.GetOutputTransformedPath(fileName), []byte(source)); err != nil {
						return emitResultFailure(err.Error())
					}
				}
			}
			proxyProgram = result.Program
		}
	}

	if transform.DiagnosticService.HasErrors() {
		return &EmitResult{EmitSkipped: true, Diagnostics: transform.DiagnosticService.Flush()}
	}
	diags := transform.DiagnosticService.Flush()

	// pre-emit diagnostics of the files to compile, in parallel
	proxyFiles := make([]*ast.SourceFile, len(sourceFiles))
	for i, sourceFile := range sourceFiles {
		proxyFiles[i] = proxyProgram.GetSourceFile(sourceFile.FileName())
		if proxyFiles[i] == nil {
			panic("compileFiles: " + sourceFile.FileName() + " is not in the program")
		}
	}
	var preEmit [][]*ast.Diagnostic
	benchmarkIfVerbose(fmt.Sprintf("type check %d files", len(proxyFiles)), func() {
		configDiagnostics := GetConfigDiagnostics(ctx, proxyProgram)
		preEmit = getPreEmitDiagnostics(ctx, proxyProgram, proxyFiles, configDiagnostics)
	})
	// see compileFilesPrinting
	needsPrintedPositions := false
	checkPositions := func(diagnostics []*ast.Diagnostic) {
		if data.TransformersRan && !printUnchanged && inFileNotPrinted(diagnostics, transformed) {
			needsPrintedPositions = true
		}
	}
	results := make([]*fileResult, len(sourceFiles))
	for i, sourceFile := range proxyFiles {
		proxyDiagnostics := preEmit[i]
		original := false
		if data.TransformersRan && hasErrors(proxyDiagnostics) {
			// The transformed source file has diagnostics, but we want to present the original diagnostic spans.
			// If the original file doesn't have diagnostics, the diagnostics are caused by a transformer and we display them instead.
			sourceDiagnostics := GetPreEmitDiagnostics(ctx, program, sourceFiles[i])
			if hasErrors(sourceDiagnostics) {
				proxyDiagnostics = sourceDiagnostics
				original = true
			}
		}
		if !original {
			checkPositions(proxyDiagnostics)
		}
		customDiagnostics := getCustomPreEmitDiagnostics(data, sourceFile)
		checkPositions(customDiagnostics)
		results[i] = &fileResult{
			preEmitDiagnostics: append(proxyDiagnostics, customDiagnostics...),
		}
	}

	// transform in parallel
	var renderTime atomic.Int64
	transformStart := time.Now()
	benchmarkIfVerbose(fmt.Sprintf("transform and render %d files", len(proxyFiles)), func() {
		transformFiles(ctx, proxyProgram, &renderTime, func(c *checker.Checker, services *transform.TransformServices, mts *transform.MultiTransformState, file *ast.SourceFile) *transform.TransformState {
			return transform.NewTransformState(
				proxyProgram,
				data,
				services,
				pathTranslator,
				mts,
				compilerOptions,
				rojoResolver,
				pkgRojoResolvers,
				nodeModulesPathMapping,
				runtimeLibRbxPath,
				c,
				projectType,
				file,
			)
		}, proxyFiles, results)
	})
	if LogService.Verbose {
		LogService.WriteLine(fmt.Sprintf("  of which luau render: %d ms of %d ms (summed over the checkers' goroutines)",
			time.Duration(renderTime.Load()).Milliseconds(), time.Since(transformStart).Milliseconds()))
	}

	// replay in file order: once a file has an error, roblox-ts transforms no further files
	type fileWrite struct {
		sourceFile *ast.SourceFile
		source     string
	}
	var fileWriteQueue []fileWrite
	var panics map[string]string
	failed := hasErrors(diags)
	for i, sourceFile := range proxyFiles {
		result := results[i]
		diags = append(diags, result.preEmitDiagnostics...)
		if (failed && !keepGoing) || hasErrors(result.preEmitDiagnostics) {
			failed = true
			continue
		}
		if result.panic != "" {
			if panics == nil {
				panics = map[string]string{}
			}
			panics[sourceFile.FileName()] = result.panic
			message := result.panic
			if index := strings.IndexByte(message, '\n'); index >= 0 {
				message = message[:index]
			}
			diags = append(diags, diagnostics.CreateTextDiagnostic(
				"goblox could not compile "+sourceFile.FileName()+": "+message, tsdiagnostics.CategoryError))
			failed = true
			continue
		}
		checkPositions(result.transformDiagnostics)
		diags = append(diags, result.transformDiagnostics...)
		if hasErrors(result.transformDiagnostics) {
			failed = true
			continue
		}
		fileWriteQueue = append(fileWriteQueue, fileWrite{sourceFile, result.source})
	}
	// diagnostics without a file, or reported for another file
	lateDiagnostics := transform.DiagnosticService.Flush()
	checkPositions(lateDiagnostics)
	diags = append(diags, lateDiagnostics...)

	if needsPrintedPositions {
		return &EmitResult{needsPrintedPositions: true}
	}

	if hasErrors(diags) && !keepGoing {
		return &EmitResult{EmitSkipped: true, Diagnostics: diags, Panics: panics}
	}

	if compilerOptions.GetEmitDeclarations() {
		LogService.Warn("goblox does not emit declaration files yet")
	}

	var emittedFiles []string
	var writeErr error
	benchmarkIfVerbose("writing compiled files", func() {
		for _, write := range fileWriteQueue {
			outPath := pathTranslator.GetOutputPath(write.sourceFile.FileName())
			if data.ProjectOptions.WriteOnlyChanged {
				if existing, err := os.ReadFile(outPath); err == nil && string(existing) == write.source {
					continue
				}
			}
			if err := outputFile(outPath, []byte(write.source)); err != nil {
				writeErr = err
				return
			}
			emittedFiles = append(emittedFiles, outPath)
		}
	})
	if writeErr != nil {
		return emitResultFailure(writeErr.Error())
	}

	return &EmitResult{EmittedFiles: emittedFiles, EmitSkipped: hasErrors(diags), Diagnostics: diags, Panics: panics}
}

// inFileNotPrinted is whether a diagnostic has a position in a file that isn't in printed.
func inFileNotPrinted(diagnostics []*ast.Diagnostic, printed map[string]string) bool {
	for _, d := range diagnostics {
		if d.File() != nil {
			if _, ok := printed[d.File().FileName()]; !ok {
				return true
			}
		}
	}
	return false
}

// TYPETORCH_TRANSFORMER is the npm package of the transformer whose Go port goblox links by default.
const TYPETORCH_TRANSFORMER = "@typetorch/transformer"

var (
	skippedPluginsMu sync.Mutex
	skippedPlugins   = map[string]bool{}
)

// pluginErrorMessage is the diagnostic for an error of PluginsForProgram. A JavaScript transformer
// without a Go port is skipped with a warning, once per process (watch mode rebuilds don't repeat it);
// "" means it was already reported. @typetorch/transformer missing is an error: its macros would
// compile to calls of functions that don't exist.
func pluginErrorMessage(err error) (message string, isError bool) {
	var unregistered *UnregisteredPluginError
	if !errors.As(err, &unregistered) {
		return err.Error(), true
	}
	if unregistered.Name == TYPETORCH_TRANSFORMER {
		return "This goblox binary was built without the Go port of " + TYPETORCH_TRANSFORMER + " (build it with -tags typetorch, see docs/goblox/CLI.md)", true
	}
	skippedPluginsMu.Lock()
	defer skippedPluginsMu.Unlock()
	if skippedPlugins[unregistered.Name] {
		return "", false
	}
	skippedPlugins[unregistered.Name] = true
	return "goblox skips the tsconfig plugin \"" + unregistered.Name + "\": JavaScript transformers can't run, only " + TYPETORCH_TRANSFORMER + " (built in)", false
}
