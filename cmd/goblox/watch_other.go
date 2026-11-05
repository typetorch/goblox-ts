//go:build !linux

package main

import (
	"io/fs"
	"path/filepath"
)

type dirWalker struct{}

// walk calls fn for every file below root, without the folders skipDir skips. It returns false when fn
// stopped it.
func (w *fileWatcher) walk(root string, fn func(path string, file watchedFile) bool) bool {
	stopped := false
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			// a file deleted while walking; the next poll sees the deletion
			return nil
		}
		if entry.IsDir() {
			if path != root && w.skipDir(path, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		if !fn(path, watchedFile{modTime: info.ModTime().UnixNano(), size: info.Size(), exists: true}) {
			stopped = true
			return filepath.SkipAll
		}
		return nil
	})
	return !stopped
}
