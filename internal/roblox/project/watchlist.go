package project

// What a build reads from the file system, for watch mode. roblox-ts's watcher (setupProjectWatchProgram)
// watches the rootDir/rootDirs; goblox also watches the files outside them that the build reads, so a change
// to an extended tsconfig, the Rojo project, package.json or a lockfile rebuilds too.

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/bundled"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

// rojoDefaultProjectName is the Rojo project file a project without one would use first.
const rojoDefaultProjectName = "default.project.json"

// packageLockfiles are the lockfiles incrementalKey reads beside every package.json.
var packageLockfiles = []string{"bun.lock", "bun.lockb", "package-lock.json", "yarn.lock", "pnpm-lock.yaml"}

// WatchList is what one build read. Every build that is given a WatchList replaces its contents, also when
// the build fails, with what it read before the failure.
type WatchList struct {
	// Dirs are the source folders (rootDir or rootDirs), whose files are all inputs: compiled, or copied to
	// the output. A watcher walks them without node_modules, .git and Ignored, like getProjectOutputs.
	Dirs []string
	// Files are single files outside Dirs: the tsconfig and the configs it extends or references, the Rojo
	// project files, typetorch.json, package.json and the lockfiles, and the program's files outside Dirs
	// that are not in a node_modules folder (such as a typings folder). Some of them may not exist yet.
	Files []string
	// Ignored are the build's own output folders (outDir, declarationDir, the include folder).
	Ignored []string
}

func (w *WatchList) reset() {
	if w != nil {
		*w = WatchList{}
	}
}

func (w *WatchList) addDir(dir string) {
	if w != nil && dir != "" {
		w.Dirs = append(w.Dirs, nodepath.Normalize(dir))
	}
}

func (w *WatchList) addFile(file string) {
	if w != nil && file != "" && !bundled.IsBundled(file) {
		w.Files = append(w.Files, nodepath.Normalize(file))
	}
}

func (w *WatchList) addIgnored(dir string) {
	if w != nil && dir != "" {
		w.Ignored = append(w.Ignored, nodepath.Normalize(dir))
	}
}

// addPackageFiles adds package.json and the lockfiles of directory and every folder above it, which
// findPackageJson and incrementalKey read.
func (w *WatchList) addPackageFiles(directory string) {
	if w == nil {
		return
	}
	for {
		w.addFile(nodepath.Join(directory, "package.json"))
		for _, lockfile := range packageLockfiles {
			w.addFile(nodepath.Join(directory, lockfile))
		}
		parent := nodepath.Dirname(directory)
		if parent == directory {
			return
		}
		directory = parent
	}
}

// finish sorts the lists and drops duplicates, nested Dirs, and Files a walk of Dirs already sees or that
// are outputs.
func (w *WatchList) finish(fs vfs.FS) {
	if w == nil {
		return
	}
	w.Ignored = sortedUniquePaths(fs, w.Ignored)
	dirs := sortedUniquePaths(fs, w.Dirs)
	w.Dirs = dirs[:0]
	for _, dir := range dirs {
		if !w.covers(dir) {
			w.Dirs = append(w.Dirs, dir)
		}
	}
	files := sortedUniquePaths(fs, w.Files)
	w.Files = files[:0]
	for _, file := range files {
		if !w.covers(file) && !w.isIgnored(file) {
			w.Files = append(w.Files, file)
		}
	}
}

// covers is whether a walk of Dirs sees path. A walk skips node_modules, .git and Ignored folders.
func (w *WatchList) covers(path string) bool {
	for _, dir := range w.Dirs {
		if path == dir || !nodepath.IsPathDescendantOf(path, dir) || w.isIgnored(path) {
			continue
		}
		segments := strings.Split(nodepath.Relative(dir, path), "/")
		if !slices.Contains(segments, NODE_MODULES) && !slices.Contains(segments, ".git") {
			return true
		}
	}
	return false
}

func (w *WatchList) isIgnored(path string) bool {
	for _, dir := range w.Ignored {
		if nodepath.IsPathDescendantOf(path, dir) {
			return true
		}
	}
	return false
}

func sortedUniquePaths(fs vfs.FS, paths []string) []string {
	seen := make(map[string]bool, len(paths))
	unique := make([]string, 0, len(paths))
	for _, path := range paths {
		if key := projectPathKey(fs, path); !seen[key] {
			seen[key] = true
			unique = append(unique, path)
		}
	}
	slices.Sort(unique)
	return unique
}

// isNodeModulesPath is whether path is inside a node_modules folder: those change with an install, which
// changes package.json or a lockfile.
func isNodeModulesPath(path string) bool {
	return strings.Contains(nodepath.Normalize(path), "/"+NODE_MODULES+"/")
}
