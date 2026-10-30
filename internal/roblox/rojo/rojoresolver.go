// Package rojo is a port of @roblox-ts/rojo-resolver 1.3.0 and @roblox-ts/path-translator 1.1.1.
package rojo

import (
	"fmt"
	"io"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

const (
	luaExt  = ".lua"
	luauExt = ".luau"
	jsonExt = ".json"
	tomlExt = ".toml"

	initName = "init"

	serverSubext = ".server"
	clientSubext = ".client"
	pluginSubext = ".plugin"
	moduleSubext = ""
)

var (
	rojoModuleExts = map[string]bool{luauExt: true, jsonExt: true, tomlExt: true}
	rojoScriptExts = map[string]bool{luauExt: true}
)

var rojoFileRegex = regexp.MustCompile(`^.+\.project\.json$`)

const (
	rojoDefaultName = "default.project.json"
	rojoOldName     = "roblox-project.json"
)

// RbxType is the Roblox script class of a file.
type RbxType int

const (
	RbxTypeModuleScript RbxType = iota
	RbxTypeScript
	RbxTypeLocalScript
)

var subExtTypeMap = map[string]RbxType{
	moduleSubext: RbxTypeModuleScript,
	serverSubext: RbxTypeScript,
	clientSubext: RbxTypeLocalScript,
	pluginSubext: RbxTypeScript,
}

// RbxPath is a path in the Roblox instance tree.
type RbxPath = []string

var defaultIsolatedContainers = []RbxPath{
	{"StarterPack"},
	{"StarterGui"},
	{"StarterPlayer", "StarterPlayerScripts"},
	{"StarterPlayer", "StarterCharacterScripts"},
	{"StarterPlayer", "StarterCharacter"},
	{"PluginDebugService"},
}

var (
	clientContainers = []RbxPath{{"StarterPack"}, {"StarterGui"}, {"StarterPlayer"}}
	serverContainers = []RbxPath{{"ServerStorage"}, {"ServerScriptService"}}
)

// RbxPathPart is one step of a RelativeRbxPath: a child name, or the parent (`RbxPathParent`).
type RbxPathPart struct {
	Name     string
	IsParent bool
}

// RbxPathParent is the `RbxPathParent` symbol.
var RbxPathParent = RbxPathPart{IsParent: true}

// RelativeRbxPath is the result of RojoResolver.relative.
type RelativeRbxPath = []RbxPathPart

// PartitionInfo maps a folder on disk to a place in the instance tree.
type PartitionInfo struct {
	RbxPath RbxPath
	FsPath  string
}

// FileRelation is how an importing file and an imported module relate across isolated containers.
type FileRelation int

const (
	FileRelationOutToOut FileRelation = iota // absolute
	FileRelationOutToIn                      // error
	FileRelationInToOut                      // absolute
	FileRelationInToIn                       // relative
)

// NetworkType is where a file runs.
type NetworkType int

const (
	NetworkTypeUnknown NetworkType = iota
	NetworkTypeClient
	NetworkTypeServer
)

func stripRojoExts(filePath string) string {
	ext := nodepath.Extname(filePath)
	if rojoModuleExts[ext] {
		filePath = filePath[:len(filePath)-len(ext)]
		if rojoScriptExts[ext] {
			subext := nodepath.Extname(filePath)
			if subext == serverSubext || subext == clientSubext || subext == pluginSubext {
				filePath = filePath[:len(filePath)-len(subext)]
			}
		}
	}
	return filePath
}

func arrayStartsWith(a, b []string) bool {
	minLength := min(len(a), len(b))
	for i := range minLength {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isPathDescendantOf(filePath, dirPath string) bool {
	return nodepath.IsPathDescendantOf(filePath, dirPath)
}

func convertToLuau(filePath string) string {
	ext := nodepath.Extname(filePath)
	if ext == luaExt {
		return filePath[:len(filePath)-len(ext)] + luauExt
	}
	return filePath
}

// RojoResolver maps files to their place in the Roblox instance tree described by a Rojo project.
type RojoResolver struct {
	fs                   vfs.FS
	warnings             []string
	rbxPath              []string
	partitions           []PartitionInfo
	filePathToRbxPathMap map[string]RbxPath
	isolatedContainers   []RbxPath
	IsGame               bool

	// rbxPathCache memoizes GetRbxPathFromFilePath (file path -> RbxPath). The transform asks for the
	// same files over and over (every import of a module, from every checker goroutine), and the
	// resolver never changes after it is built.
	rbxPathCache sync.Map
}

func newRojoResolver(fs vfs.FS) *RojoResolver {
	return &RojoResolver{
		fs:                   fs,
		filePathToRbxPathMap: map[string]RbxPath{},
		isolatedContainers:   slices.Clone(defaultIsolatedContainers),
	}
}

// cloneRbxPath copies an RbxPath, never returning nil: an empty RbxPath is a valid result (the root),
// while nil means "undefined".
func cloneRbxPath(p RbxPath) RbxPath {
	return append(RbxPath{}, p...)
}

func pathExists(fs vfs.FS, p string) bool {
	return fs.FileExists(p) || fs.DirectoryExists(p)
}

// readdir is fs.readdirSync: the names of the entries of a directory, sorted like libuv's scandir.
func readdir(fs vfs.FS, p string) []string {
	entries := fs.GetAccessibleEntries(p)
	names := append(slices.Clone(entries.Files), entries.Directories...)
	sort.Strings(names)
	return names
}

// FindRojoConfigFilePath is RojoResolver.findRojoConfigFilePath. path is "" when there is none.
func FindRojoConfigFilePath(fs vfs.FS, projectPath string) (path string, warnings []string) {
	defaultPath := nodepath.Join(projectPath, rojoDefaultName)
	if pathExists(fs, defaultPath) {
		return defaultPath, warnings
	}

	var candidates []string
	for _, fileName := range readdir(fs, projectPath) {
		if fileName != rojoDefaultName && (fileName == rojoOldName || rojoFileRegex.MatchString(fileName)) {
			candidates = append(candidates, nodepath.Join(projectPath, fileName))
		}
	}

	if len(candidates) > 1 {
		warnings = append(warnings, "Multiple *.project.json files found, using "+candidates[0])
	}
	if len(candidates) == 0 {
		return "", warnings
	}
	return candidates[0], warnings
}

func (r *RojoResolver) warn(str string) {
	r.warnings = append(r.warnings, str)
}

// GetWarnings returns the warnings collected while parsing.
func (r *RojoResolver) GetWarnings() []string {
	return r.warnings
}

// WriteFingerprint writes everything the resolver answers from, so two resolvers with the same fingerprint
// give the same answers.
func (r *RojoResolver) WriteFingerprint(w io.Writer) {
	fmt.Fprintf(w, "game=%t warnings=%q\n", r.IsGame, r.warnings)
	for _, partition := range r.partitions {
		fmt.Fprintf(w, "partition %q %q\n", partition.FsPath, partition.RbxPath)
	}
	for _, filePath := range slices.Sorted(maps.Keys(r.filePathToRbxPathMap)) {
		fmt.Fprintf(w, "file %q %q\n", filePath, r.filePathToRbxPathMap[filePath])
	}
	for _, container := range r.isolatedContainers {
		fmt.Fprintf(w, "isolated %q\n", container)
	}
}

// FromPath is RojoResolver.fromPath.
func FromPath(fs vfs.FS, rojoConfigFilePath string) *RojoResolver {
	resolver := newRojoResolver(fs)
	resolver.parseConfig(nodepath.Resolve(rojoConfigFilePath), true)
	return resolver
}

// Synthetic creates a synthetic RojoResolver for ProjectType.Package. Forces all imports to be relative.
func Synthetic(fs vfs.FS, basePath string) *RojoResolver {
	resolver := newRojoResolver(fs)
	resolver.parseTree(basePath, "", &RojoTree{Path: &basePath}, true)
	return resolver
}

// FromTree is RojoResolver.fromTree.
func FromTree(fs vfs.FS, basePath string, tree *RojoTree) *RojoResolver {
	resolver := newRojoResolver(fs)
	resolver.parseTree(basePath, "", tree, true)
	return resolver
}

func (r *RojoResolver) parseConfig(rojoConfigFilePath string, doNotPush bool) {
	if pathExists(r.fs, rojoConfigFilePath) {
		realPath := r.fs.Realpath(rojoConfigFilePath)
		text, _ := r.fs.ReadFile(realPath)
		config, err := parseRojoFile(text)
		if err == "" {
			r.parseTree(nodepath.Dirname(rojoConfigFilePath), config.Name, config.Tree, doNotPush)
		} else {
			r.warn("RojoResolver: Invalid configuration! " + err)
		}
	} else {
		r.warn(`RojoResolver: Path does not exist "` + rojoConfigFilePath + `"`)
	}
}

func (r *RojoResolver) parseTree(basePath string, name string, tree *RojoTree, doNotPush bool) {
	if !doNotPush {
		r.rbxPath = append(r.rbxPath, name)
	}

	if tree.Path != nil {
		r.parsePath(nodepath.Resolve(basePath, *tree.Path))
	}

	if tree.ClassName == "DataModel" {
		r.IsGame = true
	}

	for _, child := range tree.Children {
		r.parseTree(basePath, child.Name, child.Tree, false)
	}

	if !doNotPush {
		r.rbxPath = r.rbxPath[:len(r.rbxPath)-1]
	}
}

func (r *RojoResolver) parsePath(itemPath string) {
	itemPath = convertToLuau(itemPath)
	// a project file contributes its tree at the mount point, without adding its project name
	if rojoFileRegex.MatchString(nodepath.Basename(itemPath)) {
		if pathExists(r.fs, itemPath) {
			r.parseConfig(itemPath, true)
		}
		return
	}

	realPath := itemPath
	if pathExists(r.fs, itemPath) {
		realPath = r.fs.Realpath(itemPath)
	}
	ext := nodepath.Extname(itemPath)
	if rojoModuleExts[ext] {
		r.filePathToRbxPathMap[itemPath] = cloneRbxPath(r.rbxPath)
	} else {
		isDirectory := r.fs.DirectoryExists(realPath)
		if isDirectory && slices.Contains(readdir(r.fs, realPath), rojoDefaultName) {
			r.parseConfig(nodepath.Join(itemPath, rojoDefaultName), true)
		} else {
			r.partitions = slices.Insert(r.partitions, 0, PartitionInfo{
				FsPath:  itemPath,
				RbxPath: cloneRbxPath(r.rbxPath),
			})

			if isDirectory {
				r.searchDirectory(itemPath, "")
			}
		}
	}
}

func (r *RojoResolver) searchDirectory(directory string, item string) {
	// the entries follow symlinks, so nothing here needs a Realpath (which opens the file on Windows)
	entries := r.fs.GetAccessibleEntries(directory)

	if slices.Contains(entries.Files, rojoDefaultName) || slices.Contains(entries.Directories, rojoDefaultName) {
		r.parseConfig(nodepath.Join(directory, rojoDefaultName), false)
		return
	}

	if item != "" {
		r.rbxPath = append(r.rbxPath, item)
	}

	// *.project.json
	for _, child := range slices.Sorted(slices.Values(entries.Files)) {
		if rojoFileRegex.MatchString(child) {
			r.parseConfig(nodepath.Join(directory, child), false)
		}
	}

	// folders
	for _, child := range slices.Sorted(slices.Values(entries.Directories)) {
		r.searchDirectory(nodepath.Join(directory, child), child)
	}

	if item != "" {
		r.rbxPath = r.rbxPath[:len(r.rbxPath)-1]
	}
}

// GetRbxPathFromFilePath returns the instance path of a file, or nil when no partition contains it.
// The result is shared between callers and must not be modified.
func (r *RojoResolver) GetRbxPathFromFilePath(filePath string) RbxPath {
	if cached, ok := r.rbxPathCache.Load(filePath); ok {
		return cached.(RbxPath)
	}
	// clip so that an append by a caller copies instead of writing into the cached array
	rbxPath := slices.Clip(r.getRbxPathFromFilePath(filePath))
	r.rbxPathCache.Store(filePath, rbxPath)
	return rbxPath
}

func (r *RojoResolver) getRbxPathFromFilePath(filePath string) RbxPath {
	filePath = nodepath.Resolve(filePath)
	filePath = convertToLuau(filePath)

	if rbxPath, ok := r.filePathToRbxPathMap[filePath]; ok {
		return rbxPath
	}

	ext := nodepath.Extname(filePath)
	for _, partition := range r.partitions {
		if isPathDescendantOf(filePath, partition.FsPath) {
			stripped := stripRojoExts(filePath)
			relativePath := nodepath.Relative(partition.FsPath, stripped)
			var relativeParts []string
			if relativePath != "" {
				relativeParts = strings.Split(relativePath, nodepath.Sep)
			}
			if rojoScriptExts[ext] && len(relativeParts) > 0 && relativeParts[len(relativeParts)-1] == initName {
				relativeParts = relativeParts[:len(relativeParts)-1]
			}
			return append(cloneRbxPath(partition.RbxPath), relativeParts...)
		}
	}
	return nil
}

// GetRbxTypeFromFilePath returns the script class a file becomes.
func (r *RojoResolver) GetRbxTypeFromFilePath(filePath string) RbxType {
	filePath = convertToLuau(filePath)
	ext := nodepath.Extname(filePath)
	subext := nodepath.Extname(nodepath.Basename(filePath, ext))
	if rojoScriptExts[ext] {
		if t, ok := subExtTypeMap[subext]; ok {
			return t
		}
		return RbxTypeModuleScript
	}
	// non-script exts cannot use .server, .client, etc.
	return RbxTypeModuleScript
}

// getContainer returns the index in from of the container rbxPath is in, or -1.
func (r *RojoResolver) getContainer(from []RbxPath, rbxPath RbxPath) int {
	if r.IsGame && rbxPath != nil {
		for i, container := range from {
			if arrayStartsWith(rbxPath, container) {
				return i
			}
		}
	}
	return -1
}

// GetFileRelation is RojoResolver.getFileRelation.
func (r *RojoResolver) GetFileRelation(fileRbxPath RbxPath, moduleRbxPath RbxPath) FileRelation {
	fileContainer := r.getContainer(r.isolatedContainers, fileRbxPath)
	moduleContainer := r.getContainer(r.isolatedContainers, moduleRbxPath)
	switch {
	case fileContainer != -1 && moduleContainer != -1:
		if fileContainer == moduleContainer {
			return FileRelationInToIn
		}
		return FileRelationOutToIn
	case fileContainer != -1:
		return FileRelationInToOut
	case moduleContainer != -1:
		return FileRelationOutToIn
	default:
		return FileRelationOutToOut
	}
}

// IsIsolated reports whether rbxPath is inside an isolated container.
func (r *RojoResolver) IsIsolated(rbxPath RbxPath) bool {
	return r.getContainer(r.isolatedContainers, rbxPath) != -1
}

// GetNetworkType is RojoResolver.getNetworkType.
func (r *RojoResolver) GetNetworkType(rbxPath RbxPath) NetworkType {
	if r.getContainer(serverContainers, rbxPath) != -1 {
		return NetworkTypeServer
	}
	if r.getContainer(clientContainers, rbxPath) != -1 {
		return NetworkTypeClient
	}
	return NetworkTypeUnknown
}

// Relative is RojoResolver.relative.
func Relative(rbxFrom RbxPath, rbxTo RbxPath) RelativeRbxPath {
	maxLength := max(len(rbxFrom), len(rbxTo))
	diffIndex := maxLength
	for i := range maxLength {
		if i >= len(rbxFrom) || i >= len(rbxTo) || rbxFrom[i] != rbxTo[i] {
			diffIndex = i
			break
		}
	}

	result := RelativeRbxPath{}
	if diffIndex < len(rbxFrom) {
		for range len(rbxFrom) - diffIndex {
			result = append(result, RbxPathParent)
		}
	}

	for i := diffIndex; i < len(rbxTo); i++ {
		result = append(result, RbxPathPart{Name: rbxTo[i]})
	}

	return result
}

// GetPartitions is RojoResolver.getPartitions.
func (r *RojoResolver) GetPartitions() []PartitionInfo {
	return r.partitions
}
