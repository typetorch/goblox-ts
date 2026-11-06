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

	mu sync.Mutex
	// working is false when a watch could not be set up or ended; the watcher then polls as without it
	working bool
	watches []fswatch.Watch
	// dirs and files are what the watches cover
	dirs    []string
	files   []string
	ignored []string
}

// newNativeWatch returns a native watch, or one that never wakes the loop when disabled (--usePolling).
func newNativeWatch(disabled bool) *nativeWatch {
	return &nativeWatch{disabled: disabled, wake: make(chan struct{}, 1)}
}

// active is whether the OS reports changes to every watched path, so the poll can be slow.
func (n *nativeWatch) active() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.working
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
	if n.watches != nil && slices.Equal(n.dirs, w.dirs) && slices.Equal(n.files, w.files) && slices.Equal(n.ignored, w.ignored) {
		return
	}
	n.closeLocked()
	n.dirs, n.files, n.ignored = slices.Clone(w.dirs), slices.Clone(w.files), slices.Clone(w.ignored)
	watcher := fswatch.Default()
	if !watcher.Available() {
		return
	}
	n.working = true
	ignore := fswatch.WithIgnore(n.ignoredPath)
	for _, dir := range n.dirs {
		watch, err := watcher.WatchDirectory(dir, n.event, fswatch.WithRecursive(), ignore)
		if err != nil {
			n.working = false
			continue
		}
		n.watches = append(n.watches, watch)
	}
	for _, file := range n.files {
		// a file whose folder doesn't exist yet (a lockfile above the project) is only polled
		watch, err := watcher.WatchFile(file, n.event)
		if err != nil {
			n.working = false
			continue
		}
		n.watches = append(n.watches, watch)
	}
	if n.watches == nil {
		// keeps follow from trying again on every build
		n.watches = []fswatch.Watch{}
	}
}

// event is the watches' callback. An error (an overflow, or a watch that ended) also wakes the loop, and a
// watch that ended leaves polling in charge.
func (n *nativeWatch) event(_ []fswatch.Event, err error) {
	if err != nil && !errors.Is(err, fswatch.ErrOverflow) {
		n.mu.Lock()
		n.working = false
		n.mu.Unlock()
	}
	select {
	case n.wake <- struct{}{}:
	default:
	}
}

// ignoredPath is what a poll's walk skips: node_modules, .git and the build's outputs.
func (n *nativeWatch) ignoredPath(path string) bool {
	path = filepath.Clean(path)
	for _, segment := range strings.Split(filepath.ToSlash(path), "/") {
		if segment == "node_modules" || segment == ".git" {
			return true
		}
	}
	for _, ignored := range n.ignored {
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
	n.working = false
}
