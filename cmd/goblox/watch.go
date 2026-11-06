package main

// Watch mode: Project/functions/setupProjectWatchProgram.ts. roblox-ts watches with chokidar and rebuilds
// the changed files; goblox polls modification times (stdlib only, so no new module dependency) and
// compiles what the changes can reach (internal/roblox/project/incremental.go). The status lines are
// ts.createWatchStatusReporter(ts.sys, true)'s, which `typetorch serve` parses.

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"time"

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

// runWatch is the watch loop. After hibernateAfter without a build (0 = never), the watcher drops what it
// keeps in memory between builds and only polls; the next change rebuilds from the cache file. usePolling
// turns the OS's change events off (watch_native.go), so only the poll notices changes.
func runWatch(opts project.BuildOptions, handoff *handoffServer, hibernateAfter time.Duration, usePolling bool) int {
	session := newWatchSession(opts)
	opts = session.opts
	// the inputs the last build read: the first build reports them
	watcher := &fileWatcher{}
	native := newNativeWatch(usePolling)
	defer native.close()
	compile := func(initial bool) {
		if initial {
			reportWatchStatus("Starting compilation in watch mode...")
		} else {
			reportWatchStatus("File change detected. Starting incremental compilation...")
		}
		result, errorCount, _ := session.build(opts, stdOutput)
		// a build that failed before it read everything keeps watching what the last one read too
		watcher.setList(*opts.Watch, result == nil || result.Fatal)
		native.follow(watcher)
		plural := "s"
		if errorCount == 1 {
			plural = ""
		}
		reportWatchStatus(fmt.Sprintf("Found %d error%s. Watching for file changes.", errorCount, plural))
	}

	compile(true)
	if !projectPathExists(opts) || watcher.empty() {
		// "Unable to find tsconfig.json!": nothing to watch
		return 1
	}
	// one-shot builds of the project can be handed to this watcher from now on (handoff.go)
	if handoff.listen() {
		defer handoff.close()
		handoff.closeOnSignal()
	}
	watcher.take()
	poll := time.NewTimer(native.pollInterval())
	for {
		select {
		case job := <-handoff.requests():
			session.handOff(job)
			continue
		case <-native.wake:
			// the OS reported a change: look now, without waiting for the poll
		case <-poll.C:
			poll.Reset(native.pollInterval())
		}
		if !watcher.changed() {
			if hibernateAfter > 0 && session.idleFor() >= hibernateAfter && session.hibernate() {
				reportWatchStatus("Idle for " + hibernateAfter.String() + ": dropped the in-memory state until the next change.")
			}
			continue
		}
		// wait until the files stop changing (an editor saving several files, a git checkout)
		for {
			watcher.take()
			time.Sleep(native.debounce())
			if !watcher.changed() {
				break
			}
		}
		compile(false)
		// outputs are ignored, but a build can also touch inputs (none today); start from after the build
		watcher.take()
	}
}

// newWatchSession sets opts up for watch mode.
func newWatchSession(opts project.BuildOptions) *watchSession {
	// a full rebuild rewrites every output; only the changed ones should reach Rojo
	if _, given := opts.Overrides["writeOnlyChanged"]; !given {
		opts.Overrides["writeOnlyChanged"] = true
	}

	// reuse the parsed and bound files that did not change, like tsc -w
	opts.SourceFileCache = &project.SourceFileCache{}
	// and compile only what a change can reach
	if opts.Incremental != nil {
		opts.Incremental.Memory = true
		opts.Incremental.Background = true
	}
	opts.Watch = &project.WatchList{}
	return &watchSession{opts: opts}
}

// watchSession runs the builds of a watcher: its own, and the ones handed off to it.
type watchSession struct {
	opts project.BuildOptions
	// gcMu guards builds, which counts the builds; a save that finishes after the next build started leaves
	// the GC alone
	gcMu   sync.Mutex
	builds int
	// lastBuild is when the last build ended, and hibernated is whether the state was dropped since
	lastBuild  time.Time
	hibernated bool
}

// idleFor is how long ago the last build ended.
func (w *watchSession) idleFor() time.Duration {
	w.gcMu.Lock()
	defer w.gcMu.Unlock()
	return time.Since(w.lastBuild)
}

// hibernate drops the parsed files and the incremental state kept for the next build, keeping the cache file,
// and hands the memory back. It is false when the state was already dropped.
func (w *watchSession) hibernate() bool {
	w.gcMu.Lock()
	if w.hibernated {
		w.gcMu.Unlock()
		return false
	}
	w.hibernated = true
	w.gcMu.Unlock()
	if w.opts.Incremental != nil {
		w.opts.Incremental.Hibernate()
	}
	w.opts.SourceFileCache.Clear()
	debug.FreeOSMemory()
	return true
}

// build is buildOnce with the GC tuned for a build, and back to Go's default once the state is saved.
func (w *watchSession) build(opts project.BuildOptions, out consoleOutput) (*project.EmitResult, int, bool) {
	w.gcMu.Lock()
	w.builds++
	build := w.builds
	w.hibernated = false
	project.SetBuildGC(true)
	w.gcMu.Unlock()
	result, errorCount, ok := buildOnce(opts, out)
	w.gcMu.Lock()
	w.lastBuild = time.Now()
	w.gcMu.Unlock()
	// the watcher idles between builds: Go's default GOGC, and the build's garbage handed back once the
	// state is saved
	go func() {
		if opts.Incremental != nil {
			opts.Incremental.WaitForSave()
		}
		w.gcMu.Lock()
		defer w.gcMu.Unlock()
		if build == w.builds {
			project.SetBuildGC(false)
			debug.FreeOSMemory()
		}
	}()
	return result, errorCount, ok
}

// projectPathExists is whether the -p path (a tsconfig, or a folder at or below the one that has it) exists.
func projectPathExists(opts project.BuildOptions) bool {
	_, err := os.Stat(tspath.GetNormalizedAbsolutePath(opts.Project, opts.CurrentDirectory))
	return err == nil
}

// watchedFile is what a poll compares. exists is false for a watched file that is missing.
type watchedFile struct {
	modTime int64
	size    int64
	exists  bool
}

// fileWatcher polls the inputs of the last build (project.WatchList): it walks the source folders, without
// node_modules, .git and the build's outputs (setupProjectWatchProgram's `ignored`), and checks the single
// files. The snapshot is filled once and then compared in place, so an idle poll allocates only what the
// file system calls do.
type fileWatcher struct {
	dirs    []string
	files   []string
	ignored []string
	// snapshot has every path the last take visited
	snapshot map[string]watchedFile
	// visits counts the paths the last take visited, so a deleted file is noticed without a second map
	visits int
	// walker is the platform's walk state (watch_linux.go)
	walker dirWalker
}

// setList replaces the paths to watch, or adds to them when keep is set.
func (w *fileWatcher) setList(list project.WatchList, keep bool) {
	if !keep {
		w.dirs, w.files, w.ignored = nil, nil, nil
	}
	w.dirs = appendWatchPaths(w.dirs, list.Dirs)
	w.files = appendWatchPaths(w.files, list.Files)
	w.ignored = appendWatchPaths(w.ignored, list.Ignored)
}

func appendWatchPaths(paths []string, add []string) []string {
	for _, path := range add {
		path = filepath.Clean(filepath.FromSlash(path))
		if !slices.ContainsFunc(paths, func(other string) bool { return sameWatchPath(other, path) }) {
			paths = append(paths, path)
		}
	}
	return paths
}

func (w *fileWatcher) empty() bool {
	return len(w.dirs) == 0 && len(w.files) == 0
}

// take fills the snapshot with the current state of the watched paths.
func (w *fileWatcher) take() {
	if w.snapshot == nil {
		w.snapshot = map[string]watchedFile{}
	}
	clear(w.snapshot)
	w.visits = 0
	w.visit(func(path string, file watchedFile) bool {
		// a walk's path can be a view of its buffer
		w.snapshot[strings.Clone(path)] = file
		w.visits++
		return true
	})
}

// changed is whether a watched path differs from the snapshot. It stops at the first difference.
func (w *fileWatcher) changed() bool {
	visits := 0
	differs := false
	w.visit(func(path string, file watchedFile) bool {
		visits++
		if old, ok := w.snapshot[path]; !ok || old != file {
			differs = true
			return false
		}
		return true
	})
	return differs || visits != w.visits
}

// visit calls fn for every watched file, until fn returns false. path is only valid during the call.
func (w *fileWatcher) visit(fn func(path string, file watchedFile) bool) {
	for _, path := range w.files {
		file := watchedFile{}
		if info, err := os.Stat(path); err == nil {
			file = watchedFile{modTime: info.ModTime().UnixNano(), size: info.Size(), exists: true}
		}
		if !fn(path, file) {
			return
		}
	}
	for _, root := range w.dirs {
		if !w.walk(root, fn) {
			return
		}
	}
}

func (w *fileWatcher) skipDir(path string, name string) bool {
	if name == "node_modules" || name == ".git" {
		return true
	}
	for _, ignored := range w.ignored {
		if sameWatchPath(path, ignored) {
			return true
		}
	}
	return false
}

// sameWatchPath compares two clean paths, ignoring case on Windows.
func sameWatchPath(a, b string) bool {
	if os.PathSeparator == '\\' {
		return strings.EqualFold(a, b)
	}
	return a == b
}
