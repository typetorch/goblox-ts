package project

// Incremental builds: which files a build has to process, and what it can reuse from the last good build.
//
// There are two states, because the plugins depend on types and the compiled text depends on the plugins:
//
//	original program -> affected files -> plugins run on those, the others reuse their printed text
//	proxy program    -> affected files -> check, transform and render those, the others keep their output
//
// A file stops being "clean" when its text changes or when the d.ts signature of something it depends on
// changes (internal/execute/incremental). A state is only replaced by a build that succeeded: a build with
// errors writes nothing and commits nothing, so the next build compares with the same last good state, finds
// the same files and reports the same errors, until they are fixed. Files with any diagnostic are not clean
// either, so their warnings are reported by every build.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/execute/incremental"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

// IncrementalSession keeps the last good state between builds. Persist reads and writes the cache file next
// to tsBuildInfoFile, when the tsconfig has `incremental` or `composite`. Memory keeps the state for the next
// build of the session in any case (watch mode). With neither, builds are not incremental.
type IncrementalSession struct {
	Persist bool
	Memory  bool

	state *incrementalState
	// proxyFiles keeps the parsed and bound text of re-printed files between the builds of a session
	proxyFiles sync.Map // tspath.Path -> *ast.SourceFile
	// lastCompiled counts the files the last build checked and transformed
	lastCompiled int
}

type incrementalState struct {
	key string
	// original is nil when the project has no plugins: proxy is then the only program
	original *incremental.Program
	proxy    *incremental.Program
	// files has an entry for every compiled file, by file name
	files           map[string]*incrementalFile
	rojoFingerprint string
	rojoWarnings    []string
}

// incrementalFile is what the last good build knows about one compiled file.
type incrementalFile struct {
	// Printed is the plugins' output, when they changed the file
	Printed    string `json:"printed,omitempty"`
	Reprinted  bool   `json:"reprinted,omitempty"`
	OutputHash string `json:"outputHash"`
	// Clean is false when the file had diagnostics, which the next build has to report again
	Clean bool `json:"clean,omitempty"`
}

// incrementalBuild is the work of one build.
type incrementalBuild struct {
	session *IncrementalSession
	key     string
	// cachePath is "" when the state stays in memory
	cachePath string
	old       *incrementalState
	// staleOutputs are the compiled files whose output is missing or is not what the last build wrote
	staleOutputs map[string]bool

	original *incremental.Program
	proxy    *incremental.Program
	// blocked is set by a diagnostic that belongs to no compiled file, which only a full build reproduces
	blocked         bool
	rojoFingerprint string
	rojoWarnings    []string
}

// incrementalEligible is false for what the state can't describe. KeepGoing writes the good files of a
// failed build. writeTransformedFiles and logTruthyChanges are about every file, and declaration outputs
// aren't emitted yet. removeComments takes the JSDoc @metadata tags out of the d.ts signatures, though
// macros read them from other files. assumeChangesOnlyAffectDirectDependencies skips dependents on purpose.
func incrementalEligible(opts BuildOptions, data *transform.ProjectData, options *core.CompilerOptions) bool {
	if opts.Incremental == nil || opts.KeepGoing {
		return false
	}
	if data.ProjectOptions.WriteTransformedFiles || data.ProjectOptions.LogTruthyChanges {
		return false
	}
	if options.GetEmitDeclarations() || options.RemoveComments.IsTrue() || options.AssumeChangesOnlyAffectDirectDependencies.IsTrue() {
		return false
	}
	// a persisted state follows the tsconfig, like tsc's .tsbuildinfo
	return opts.Incremental.Memory || opts.Incremental.Persist && options.IsIncremental()
}

// incrementalCachePath is the cache file next to the build info file: out/tsconfig.tsbuildinfo has
// out/tsconfig.goblox.tsbuildinfo.
func incrementalCachePath(buildInfoPath string) string {
	if buildInfoPath == "" {
		return ""
	}
	extension := nodepath.Extname(buildInfoPath)
	return strings.TrimSuffix(buildInfoPath, extension) + ".goblox" + extension
}

// begin picks the state to compare with and checks the outputs it describes.
func (s *IncrementalSession) begin(fs vfs.FS, loaded *loadedConfig, data *transform.ProjectData, pathTranslator *rojo.PathTranslator, program *compiler.Program, sourceFiles []*ast.SourceFile, outputs *projectOutputs) *incrementalBuild {
	b := &incrementalBuild{session: s}
	b.key = incrementalKey(fs, loaded, data, pathTranslator, program, outputs)
	if b.key == "" {
		return nil
	}
	if s.Persist && program.Options().IsIncremental() {
		b.cachePath = incrementalCachePath(pathTranslator.BuildInfoOutputPath)
		if b.cachePath == "" && !s.Memory {
			return nil
		}
		if b.cachePath != "" && s.state == nil {
			s.state = readIncrementalCache(b.cachePath, b.key, loaded, program)
		}
	}
	if s.state != nil && s.state.key == b.key {
		b.old = s.state
		b.staleOutputs = staleOutputs(b.old, pathTranslator, sourceFiles)
	}
	return b
}

// staleOutputs hashes the output of every compiled file that has one in the state.
func staleOutputs(state *incrementalState, pathTranslator *rojo.PathTranslator, sourceFiles []*ast.SourceFile) map[string]bool {
	stale := make([]bool, len(sourceFiles))
	wg := core.NewWorkGroup(false)
	for i, sourceFile := range sourceFiles {
		entry := state.files[sourceFile.FileName()]
		if entry == nil {
			continue
		}
		wg.Queue(func() {
			text, err := os.ReadFile(pathTranslator.GetOutputPath(sourceFile.FileName()))
			stale[i] = err != nil || incremental.ComputeHash(string(text), false) != entry.OutputHash
		})
	}
	wg.RunAndWait()
	result := map[string]bool{}
	for i, isStale := range stale {
		if isStale {
			result[sourceFiles[i].FileName()] = true
		}
	}
	return result
}

// upToDate is the no-change build: every input and output is what the last good build saw, and it left no
// diagnostic to report again.
func (b *incrementalBuild) upToDate(program *compiler.Program, sourceFiles []*ast.SourceFile) bool {
	if b.old == nil || len(b.staleOutputs) > 0 {
		return false
	}
	for _, sourceFile := range sourceFiles {
		entry := b.old.files[sourceFile.FileName()]
		if entry == nil || !entry.Clean {
			return false
		}
	}
	base := b.old.original
	if base == nil {
		base = b.old.proxy
	}
	return base.SameFiles(program)
}

// checkRojo drops the old state when a project file changed what the resolvers answer.
func (b *incrementalBuild) checkRojo(rojoResolver *rojo.RojoResolver, pkgRojoResolvers []*rojo.RojoResolver, nodeModulesPathMapping map[string]string) {
	hash := sha256.New()
	rojoResolver.WriteFingerprint(hash)
	for _, resolver := range pkgRojoResolvers {
		resolver.WriteFingerprint(hash)
	}
	for _, key := range slices.Sorted(maps.Keys(nodeModulesPathMapping)) {
		fmt.Fprintf(hash, "%q=%q\n", key, nodeModulesPathMapping[key])
	}
	b.rojoFingerprint = hex.EncodeToString(hash.Sum(nil))
	b.rojoWarnings = rojoResolver.GetWarnings()
	if b.old != nil && b.old.rojoFingerprint != b.rojoFingerprint {
		LogService.WriteLineIfVerbose("incremental: the Rojo project changed, compiling everything")
		b.old = nil
		b.staleOutputs = nil
	}
}

func (b *incrementalBuild) newProgram(ctx context.Context, program *compiler.Program, old *incremental.Program) *incremental.Program {
	state := incremental.NewProgram(program, old, nil, nil, false)
	state.SetSignatureSalt(decoratorText)
	state.CollectAffectedFiles(ctx)
	return state
}

// pluginWork returns the files the plugins have to run on and the printed text of the others.
func (b *incrementalBuild) pluginWork(ctx context.Context, program *compiler.Program, sourceFiles []*ast.SourceFile) ([]*ast.SourceFile, map[string]string) {
	var old *incremental.Program
	if b.old != nil {
		old = b.old.original
	}
	b.original = b.newProgram(ctx, program, old)
	files := make([]*ast.SourceFile, 0, len(sourceFiles))
	reused := map[string]string{}
	for _, file := range sourceFiles {
		var entry *incrementalFile
		if b.old != nil {
			entry = b.old.files[file.FileName()]
		}
		if entry == nil || !b.original.IsClean(file) {
			files = append(files, file)
		} else if entry.Reprinted {
			reused[file.FileName()] = entry.Printed
		}
	}
	return files, reused
}

// parseTransformed serves the plugins' text from the session when it did not change.
func (b *incrementalBuild) parseTransformed(opts ast.SourceFileParseOptions, text string) *ast.SourceFile {
	if value, ok := b.session.proxyFiles.Load(opts.Path); ok {
		if file := value.(*ast.SourceFile); file.Text() == text && file.ParseOptions() == opts {
			return file
		}
	}
	file := parser.ParseSourceFile(opts, text, core.EnsureScriptKindFromFileName(opts.FileName))
	b.session.proxyFiles.Store(opts.Path, file)
	return file
}

// filesToProcess narrows the parallel lists of original and proxy files to the ones to check and transform.
// configDiagnostics belong to the first compiled file of a full build, so with any of them there is no subset
// whose diagnostics are the same.
func (b *incrementalBuild) filesToProcess(ctx context.Context, proxyProgram *compiler.Program, sourceFiles []*ast.SourceFile, proxyFiles []*ast.SourceFile, configDiagnostics []*ast.Diagnostic) ([]*ast.SourceFile, []*ast.SourceFile) {
	var old *incremental.Program
	if b.old != nil {
		old = b.old.proxy
	}
	b.proxy = b.newProgram(ctx, proxyProgram, old)
	if b.old == nil || len(configDiagnostics) > 0 {
		b.blocked = b.blocked || len(configDiagnostics) > 0
		return sourceFiles, proxyFiles
	}
	var subsetSources, subsetProxies []*ast.SourceFile
	for i, proxyFile := range proxyFiles {
		entry := b.old.files[proxyFile.FileName()]
		if entry == nil || !entry.Clean || !b.proxy.IsClean(proxyFile) || b.staleOutputs[proxyFile.FileName()] {
			subsetSources = append(subsetSources, sourceFiles[i])
			subsetProxies = append(subsetProxies, proxyFile)
		}
	}
	return subsetSources, subsetProxies
}

// unchangedOutput is true when the file's output is on disk and already has this text.
func (b *incrementalBuild) unchangedOutput(fileName string, source string) bool {
	if b.old == nil || b.staleOutputs[fileName] {
		return false
	}
	entry := b.old.files[fileName]
	return entry != nil && entry.OutputHash == incremental.ComputeHash(source, false)
}

// commit makes this build the last good state. processed and results are the parallel lists of the files
// this build compiled.
func (b *incrementalBuild) commit(ctx context.Context, program *compiler.Program, proxyProgram *compiler.Program, sourceFiles []*ast.SourceFile, transformed map[string]string, processed []*ast.SourceFile, results []*fileResult) {
	if b.blocked || len(proxyProgram.GetGlobalDiagnostics(ctx)) > 0 {
		return
	}
	state := &incrementalState{
		key:             b.key,
		files:           make(map[string]*incrementalFile, len(sourceFiles)),
		rojoFingerprint: b.rojoFingerprint,
		rojoWarnings:    b.rojoWarnings,
	}
	for _, sourceFile := range sourceFiles {
		fileName := sourceFile.FileName()
		printed, reprinted := transformed[fileName]
		entry := &incrementalFile{Printed: printed, Reprinted: reprinted, Clean: true}
		if b.old != nil && b.old.files[fileName] != nil {
			entry.OutputHash = b.old.files[fileName].OutputHash
		}
		state.files[fileName] = entry
	}

	// the proxy's clean files are the ones compiled without a diagnostic, and whatever is not compiled
	isProcessed := make(map[*ast.SourceFile]bool, len(processed))
	for i, file := range processed {
		isProcessed[file] = true
		entry := state.files[file.FileName()]
		entry.OutputHash = incremental.ComputeHash(results[i].source, false)
		if len(results[i].preEmitDiagnostics) == 0 && len(results[i].transformDiagnostics) == 0 {
			b.proxy.MarkClean(file)
		} else {
			b.proxy.MarkNotClean(file)
			entry.Clean = false
		}
	}
	for _, file := range proxyProgram.GetSourceFiles() {
		if !isProcessed[file] && state.files[file.FileName()] == nil {
			b.proxy.MarkClean(file)
		}
	}
	if b.original != nil {
		// the plugins' diagnostics block the commit, so what they ran on is clean
		for _, file := range program.GetSourceFiles() {
			b.original.MarkClean(file)
		}
	}

	var wg sync.WaitGroup
	if b.original != nil && b.original.GetProgram() != proxyProgram {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.original.ComputeMissingSignatures(ctx)
		}()
	}
	b.proxy.ComputeMissingSignatures(ctx)
	wg.Wait()

	if b.cachePath != "" {
		if err := writeIncrementalCache(b.cachePath, state, b.original, b.proxy); err != nil {
			// the next build compiles everything, as without a cache
			LogService.WriteLineIfVerbose("incremental: could not write the cache: " + err.Error())
			_ = os.Remove(b.cachePath)
		}
	}

	if b.original != nil {
		state.original = b.original.Detached()
	}
	state.proxy = b.proxy.Detached()
	b.session.state = state
}

// decoratorText is what a file says about its declarations outside of its d.ts. Plugins read the decorators
// of declarations in other files (isUserMacro), so changing one has to count as a change of shape.
func decoratorText(file *ast.SourceFile) string {
	var text strings.Builder
	var visit func(node *ast.Node) bool
	visit = func(node *ast.Node) bool {
		switch node.Kind {
		case ast.KindDecorator:
			text.WriteString(strings.TrimSpace(file.Text()[node.Pos():node.End()]))
			text.WriteByte('\n')
		case ast.KindClassDeclaration, ast.KindClassExpression, ast.KindModuleDeclaration, ast.KindModuleBlock, ast.KindMethodDeclaration,
			ast.KindPropertyDeclaration, ast.KindGetAccessor, ast.KindSetAccessor, ast.KindConstructor, ast.KindParameter,
			ast.KindVariableStatement, ast.KindVariableDeclarationList, ast.KindVariableDeclaration:
			node.ForEachChild(visit)
		}
		return false
	}
	file.AsNode().ForEachChild(visit)
	return text.String()
}

// incrementalKey covers everything outside the program's files that the output depends on. "" means the key
// could not be computed, so nothing can be reused.
func incrementalKey(fs vfs.FS, loaded *loadedConfig, data *transform.ProjectData, pathTranslator *rojo.PathTranslator, program *compiler.Program, outputs *projectOutputs) string {
	hash := sha256.New()
	writeString := func(label string, value string) { fmt.Fprintf(hash, "%s %d\n%s\n", label, len(value), value) }
	writeJSON := func(label string, value any) bool {
		text, err := json.Marshal(value)
		if err != nil {
			return false
		}
		writeString(label, string(text))
		return true
	}
	writeFile := func(path string) {
		if text, ok := fs.ReadFile(path); ok {
			writeString("file "+path, text)
		} else {
			writeString("missing "+path, "")
		}
	}

	writeString("goblox", gobloxBuildIdentity())
	writeString("TYPETORCH_DEBUG_MACROS", os.Getenv("TYPETORCH_DEBUG_MACROS"))
	writeString("rootDir", pathTranslator.RootDir)

	// the options the output depends on: not the number of checkers, and not how the build logs or writes
	options := *loaded.config.CompilerOptions()
	options.Checkers = nil
	options.SingleThreaded = core.TSUnknown
	if !writeJSON("options", &options) {
		return ""
	}
	projectData := *data
	projectData.ProjectOptions.Verbose = false
	projectData.ProjectOptions.WriteOnlyChanged = false
	projectData.ProjectOptions.Watch = false
	projectData.ProjectOptions.UsePolling = false
	if !writeJSON("project", &projectData) || !writeJSON("plugins", PluginEntries(program)) || !writeJSON("registered", RegisteredPlugins()) {
		return ""
	}

	writeFile(data.TsConfigPath)
	for _, extended := range loaded.config.ExtendedSourceFiles() {
		writeFile(extended)
	}
	if data.RojoConfigPath != "" {
		writeFile(data.RojoConfigPath)
	}
	writeFile(nodepath.Join(data.ProjectPath, "typetorch.json"))
	// package.json and the lockfile can be above the project, in a monorepo
	for directory := data.ProjectPath; ; {
		writeFile(nodepath.Join(directory, "package.json"))
		for _, lockfile := range []string{"bun.lock", "bun.lockb", "package-lock.json", "yarn.lock", "pnpm-lock.yaml"} {
			writeFile(nodepath.Join(directory, lockfile))
		}
		parent := nodepath.Dirname(directory)
		if parent == directory {
			break
		}
		directory = parent
	}

	// a project file copied to the output is a nested Rojo project of the next build
	for _, key := range slices.Sorted(maps.Keys(outputs.assets)) {
		if input := outputs.assets[key].input; deploymentConfigRegex.MatchString(nodepath.Basename(input)) {
			writeFile(input)
		}
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// gobloxBuildIdentity changes with the compiler, including a local build with the same version.
func gobloxBuildIdentity() string {
	identity := core.Version()
	if info, ok := debug.ReadBuildInfo(); ok {
		identity += " " + info.Main.Version
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" || setting.Key == "vcs.modified" || setting.Key == "-tags" {
				identity += " " + setting.Value
			}
		}
	}
	if executable, err := os.Executable(); err == nil {
		if stat, err := os.Stat(executable); err == nil {
			identity += fmt.Sprintf(" %d %d", stat.Size(), stat.ModTime().UnixNano())
		}
	}
	return identity
}

// The cache file: a magic line, the SHA-256 of the rest, then the JSON of incrementalCache. Anything else
// is no cache at all.
const incrementalCacheMagic = "goblox-incremental-1\n"

type incrementalCache struct {
	Key             string                      `json:"key"`
	Original        json.RawMessage             `json:"original,omitempty"`
	Proxy           json.RawMessage             `json:"proxy"`
	Files           map[string]*incrementalFile `json:"files"`
	RojoFingerprint string                      `json:"rojoFingerprint"`
	RojoWarnings    []string                    `json:"rojoWarnings,omitempty"`
}

func writeIncrementalCache(path string, state *incrementalState, original *incremental.Program, proxy *incremental.Program) error {
	cache := incrementalCache{Key: state.key, Files: state.files, RojoFingerprint: state.rojoFingerprint, RojoWarnings: state.rojoWarnings}
	buildInfoPath := strings.TrimSuffix(path, ".goblox"+nodepath.Extname(path)) + nodepath.Extname(path)
	var err error
	if original != nil {
		if cache.Original, err = original.MarshalState(buildInfoPath); err != nil {
			return err
		}
	}
	if cache.Proxy, err = proxy.MarshalState(buildInfoPath); err != nil {
		return err
	}
	body, err := json.Marshal(&cache)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	var contents strings.Builder
	contents.WriteString(incrementalCacheMagic)
	contents.WriteString(hex.EncodeToString(sum[:]))
	contents.WriteByte('\n')
	contents.Write(body)

	temporary := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := outputFile(temporary, []byte(contents.String())); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		_ = os.Remove(temporary)
		return err
	}
	return nil
}

// readIncrementalCache returns nil for a missing, corrupt or old cache, or one with another key.
func readIncrementalCache(path string, key string, loaded *loadedConfig, program *compiler.Program) *incrementalState {
	state, err := parseIncrementalCache(path, key, loaded, program)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			LogService.WriteLineIfVerbose("incremental: compiling everything: " + err.Error())
		}
		return nil
	}
	return state
}

func parseIncrementalCache(path string, key string, loaded *loadedConfig, program *compiler.Program) (*incrementalState, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := string(contents)
	if !strings.HasPrefix(text, incrementalCacheMagic) {
		return nil, errors.New("the cache is from another version")
	}
	text = text[len(incrementalCacheMagic):]
	checksum, body, found := strings.Cut(text, "\n")
	sum := sha256.Sum256([]byte(body))
	if !found || checksum != hex.EncodeToString(sum[:]) {
		return nil, errors.New("the cache is corrupt")
	}
	var cache incrementalCache
	if err := json.Unmarshal([]byte(body), &cache); err != nil {
		return nil, err
	}
	if cache.Key != key {
		return nil, errors.New("the options, the project files or goblox changed")
	}
	state := &incrementalState{key: cache.Key, files: cache.Files, rojoFingerprint: cache.RojoFingerprint, rojoWarnings: cache.RojoWarnings}
	if state.files == nil {
		return nil, errors.New("the cache is corrupt")
	}
	for fileName, entry := range state.files {
		if entry == nil {
			return nil, errors.New("the cache is corrupt: " + fileName)
		}
	}
	host := program.Host()
	if len(cache.Original) > 0 {
		if state.original, err = incremental.UnmarshalState(cache.Original, loaded.config, host); err != nil {
			return nil, err
		}
	}
	if state.proxy, err = incremental.UnmarshalState(cache.Proxy, loaded.config, host); err != nil {
		return nil, err
	}
	return state, nil
}

var _ = tspath.Path("")
