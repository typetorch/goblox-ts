package main

// Native file watching for watch mode: the OS reports a change (fanotify or inotify on Linux, FSEvents on
// macOS, ReadDirectoryChangesW on Windows, see internal/fswatch) and the watcher looks at once, instead of at
// its next poll. The events only wake the watcher: what changed is still decided by comparing the poll's
// snapshot, so a missed or spurious event can't change a build. The poll keeps running, slower, as a net for
// file systems that drop events. When the OS watch can't be set up, or one ends, polling is as before.

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/typescript-go/internal/fswatch"
)

const (
	// nativePollInterval is the poll while the OS reports changes: only a net for events it drops
	nativePollInterval = time.Second
	// nativeDebounce is how long the files must stay the same before a rebuild, when the OS reports changes:
	// an editor saving several files, or a git checkout, still makes one build
	nativeDebounce = 30 * time.Millisecond
)

// nativeWatch wakes the watch loop when the OS reports a change to a watched path.
type nativeWatch struct {
	disabled bool
	// wake has room for one signal: a change while one is pending adds nothing
	wake chan struct{}
	// working is false when a watch could not be set up or ended; the watcher then polls as without it
	working atomic.Bool
	// paths is what the watches cover. The callbacks read it without a lock: it is replaced, never changed.
	paths atomic.Pointer[watchPaths]

	// mu guards watches
	mu      sync.Mutex
	watches []fswatch.Watch
}

// watchPaths are a fileWatcher's paths.
type watchPaths struct {
	dirs    []string
	files   []string
	ignored []string
}

// newNativeWatch returns a native watch, or one that never wakes the loop when disabled (--usePolling).
func newNativeWatch(disabled bool) *nativeWatch {
	n := &nativeWatch{disabled: disabled, wake: make(chan struct{}, 1)}
	n.paths.Store(&watchPaths{})
	return n
}

// active is whether the OS reports changes, so the poll can be slow.
func (n *nativeWatch) active() bool {
	return n.working.Load()
}

// pollInterval and debounce are the watch loop's timings: slower polls and a shorter settle with native
// events, the polling ones without.
func (n *nativeWatch) pollInterval() time.Duration {
	if n.active() {
		return nativePollInterval
	}
	return watchPollInterval
}

func (n *nativeWatch) debounce() time.Duration {
	if n.active() {
		return nativeDebounce
	}
	return watchDebounce
}

// follow watches what w polls. It does nothing when that did not change since the last call.
func (n *nativeWatch) follow(w *fileWatcher) {
	if n.disabled {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	old := n.paths.Load()
	if n.watches != nil && slices.Equal(old.dirs, w.dirs) && slices.Equal(old.files, w.files) && slices.Equal(old.ignored, w.ignored) {
		return
	}
	n.closeLocked()
	paths := &watchPaths{dirs: slices.Clone(w.dirs), files: slices.Clone(w.files), ignored: slices.Clone(w.ignored)}
	n.paths.Store(paths)
	n.watches = []fswatch.Watch{}
	watcher := fswatch.Default()
	if !watcher.Available() {
		return
	}
	working := true
	ignore := fswatch.WithIgnore(paths.ignoredPath)
	for _, dir := range paths.dirs {
		watch, err := watcher.WatchDirectory(dir, n.event, fswatch.WithRecursive(), ignore)
		if err != nil {
			working = false
			continue
		}
		n.watches = append(n.watches, watch)
	}
	for _, file := range paths.files {
		// a file above the project (package.json and the lockfiles of the folders above it) is only polled:
		// a file watch watches its folder, and a home folder or / changes all the time
		if !paths.inProject(file) {
			continue
		}
		watch, err := watcher.WatchFile(file, n.event)
		if err != nil {
			working = false
			continue
		}
		n.watches = append(n.watches, watch)
	}
	n.working.Store(working)
}

// event is the watches' callback. It wakes the loop for an event on a watched path. An error (an overflow,
// or a watch that ended) also wakes it, and a watch that ended leaves polling in charge.
func (n *nativeWatch) event(events []fswatch.Event, err error) {
	if err != nil {
		if !errors.Is(err, fswatch.ErrOverflow) {
			n.working.Store(false)
		}
	} else {
		paths := n.paths.Load()
		if !slices.ContainsFunc(events, func(event fswatch.Event) bool { return paths.watched(event.Path) }) {
			return
		}
	}
	select {
	case n.wake <- struct{}{}:
	default:
	}
}

// inProject is whether path is in the folder the source folders are in (the project folder, for a rootDir of
// src), or below it.
func (p *watchPaths) inProject(path string) bool {
	path = filepath.Clean(path)
	for _, dir := range p.dirs {
		if root := filepath.Dir(dir); sameWatchPath(filepath.Dir(path), root) || isWatchPathDescendant(path, root) {
			return true
		}
	}
	return false
}

// watched is whether an event for path concerns a poll: a watched file, or a file in a watched folder that
// the walk doesn't skip.
func (p *watchPaths) watched(path string) bool {
	path = filepath.Clean(path)
	if slices.ContainsFunc(p.files, func(file string) bool { return sameWatchPath(file, path) }) {
		return true
	}
	if p.ignoredPath(path) {
		return false
	}
	return slices.ContainsFunc(p.dirs, func(dir string) bool { return sameWatchPath(path, dir) || isWatchPathDescendant(path, dir) })
}

// ignoredPath is what a poll's walk skips: node_modules, .git and the build's outputs.
func (p *watchPaths) ignoredPath(path string) bool {
	path = filepath.Clean(path)
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == "node_modules" || segment == ".git" {
			return true
		}
	}
	for _, ignored := range p.ignored {
		if sameWatchPath(path, ignored) || isWatchPathDescendant(path, ignored) {
			return true
		}
	}
	return false
}

func isWatchPathDescendant(path string, dir string) bool {
	prefix := dir + string(filepath.Separator)
	if len(path) <= len(prefix) {
		return false
	}
	return sameWatchPath(path[:len(prefix)], prefix)
}

// close stops the watches.
func (n *nativeWatch) close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.closeLocked()
}

func (n *nativeWatch) closeLocked() {
	for _, watch := range n.watches {
		_ = watch.Close()
	}
	n.watches = nil
	n.working.Store(false)
}
