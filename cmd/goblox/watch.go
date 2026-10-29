package main

// Watch mode: Project/functions/setupProjectWatchProgram.ts. roblox-ts watches with chokidar and rebuilds
// the changed files; goblox polls modification times (stdlib only, so no new module dependency) and
// rebuilds the whole project, which is as fast as rbxtsc's incremental rebuild of one file. The status
// lines are ts.createWatchStatusReporter(ts.sys, true)'s, which `typetorch serve` parses.

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
)

const (
	watchPollInterval = 250 * time.Millisecond
	// setupProjectWatchProgram's setTimeout(compile, 100): changes settle before a rebuild
	watchDebounce = 100 * time.Millisecond
)

// reportWatchStatus is ts.createWatchStatusReporter(ts.sys, true) for a message diagnostic: the time
// in grey (toLocaleTimeString, en-US), then the message and a blank line.
func reportWatchStatus(message string) {
	fmt.Fprintf(os.Stdout, "[\x1b[90m%s\x1b[0m] %s\n\n", time.Now().Format("3:04:05 PM"), message)
}

func runWatch(opts project.BuildOptions, report func([]*ast.Diagnostic)) int {
	// a full rebuild rewrites every output; only the changed ones should reach Rojo
	if _, given := opts.Overrides["writeOnlyChanged"]; !given {
		opts.Overrides["writeOnlyChanged"] = true
	}

	root := watchRootFallback(opts)
	var ignored []string
	compile := func(initial bool) {
		if initial {
			reportWatchStatus("Starting compilation in watch mode...")
		} else {
			reportWatchStatus("File change detected. Starting incremental compilation...")
		}
		result, errorCount, _ := buildOnce(opts, report)
		if result != nil {
			if result.WatchRoot != "" {
				root = result.WatchRoot
			}
			if len(result.OutputRoots) > 0 {
				ignored = result.OutputRoots
			}
		}
		plural := "s"
		if errorCount == 1 {
			plural = ""
		}
		reportWatchStatus(fmt.Sprintf("Found %d error%s. Watching for file changes.", errorCount, plural))
	}

	compile(true)
	if root == "" {
		// "Unable to find tsconfig.json!": nothing to watch
		return 1
	}
	previous := snapshotWatchedFiles(root, ignored)
	for {
		time.Sleep(watchPollInterval)
		current := snapshotWatchedFiles(root, ignored)
		if sameSnapshot(previous, current) {
			continue
		}
		// wait until the files stop changing (an editor saving several files, a git checkout)
		for {
			time.Sleep(watchDebounce)
			settled := snapshotWatchedFiles(root, ignored)
			if sameSnapshot(current, settled) {
				break
			}
			current = settled
		}
		compile(false)
		// outputs are ignored, but a build can also touch inputs (none today); start from after the build
		previous = snapshotWatchedFiles(root, ignored)
	}
}

// watchRootFallback is the folder to watch before the first build reports its tsconfig's folder: the
// -p path's folder (or the path itself when it is a folder). "" when it does not exist.
func watchRootFallback(opts project.BuildOptions) string {
	p := tspath.GetNormalizedAbsolutePath(opts.Project, opts.CurrentDirectory)
	info, err := os.Stat(p)
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		return tspath.GetDirectoryPath(p)
	}
	return p
}

type watchedFile struct {
	modTime time.Time
	size    int64
}

// snapshotWatchedFiles lists the files below root with their modification times, without
// node_modules, .git and the build's own outputs (setupProjectWatchProgram's `ignored`).
func snapshotWatchedFiles(root string, ignored []string) map[string]watchedFile {
	files := map[string]watchedFile{}
	ignoredKeys := make(map[string]bool, len(ignored))
	for _, dir := range ignored {
		if dir != "" {
			ignoredKeys[watchPathKey(dir)] = true
		}
	}
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// a file deleted while walking; the next poll sees the deletion
			return nil
		}
		if entry.IsDir() {
			name := entry.Name()
			if path != root && (name == "node_modules" || name == ".git" || ignoredKeys[watchPathKey(path)]) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		files[path] = watchedFile{modTime: info.ModTime(), size: info.Size()}
		return nil
	})
	return files
}

func watchPathKey(path string) string {
	key := filepath.Clean(filepath.FromSlash(path))
	if os.PathSeparator == '\\' {
		key = strings.ToLower(key)
	}
	return key
}

func sameSnapshot(a, b map[string]watchedFile) bool {
	if len(a) != len(b) {
		return false
	}
	for path, file := range a {
		other, ok := b[path]
		if !ok || !other.modTime.Equal(file.modTime) || other.size != file.size {
			return false
		}
	}
	return true
}
