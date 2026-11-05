//go:build linux

package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unsafe"

	"golang.org/x/sys/unix"
)

// linux_dirent64, which unix.ReadDirent returns on every architecture
var (
	direntReclen = int(unsafe.Offsetof(unix.Dirent{}.Reclen))
	direntType   = int(unsafe.Offsetof(unix.Dirent{}.Type))
	direntName   = int(unsafe.Offsetof(unix.Dirent{}.Name))
)

// dirWalker keeps the buffers of the walks between polls: a getdents buffer per folder depth, the path of
// the current entry and a stat result. filepath.WalkDir allocates a DirEntry, a path and an lstat result for
// every file; this walk allocates only the name fstatat copies.
type dirWalker struct {
	buffers [][]byte
	path    []byte
	stat    unix.Stat_t
}

// walk calls fn for every file below root, without the folders skipDir skips. It returns false when fn
// stopped it. Like filepath.WalkDir, it does not follow symbolic links.
func (w *fileWatcher) walk(root string, fn func(path string, file watchedFile) bool) bool {
	fd, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		// a missing source folder: its files show up once it is created
		return true
	}
	w.walker.path = append(w.walker.path[:0], root...)
	return w.walkDir(fd, 0, fn)
}

// walkDir walks the open folder fd, whose path is walker.path, and closes it.
func (w *fileWatcher) walkDir(fd int, depth int, fn func(path string, file watchedFile) bool) bool {
	defer unix.Close(fd)
	if depth == len(w.walker.buffers) {
		w.walker.buffers = append(w.walker.buffers, make([]byte, 16<<10))
	}
	buffer := w.walker.buffers[depth]
	dirLength := len(w.walker.path)
	for {
		n, err := unix.ReadDirent(fd, buffer)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil || n <= 0 {
			// the end, or a folder deleted while walking; the next poll sees the deletion
			return true
		}
		for offset := 0; offset < n; {
			record := buffer[offset:]
			offset += int(binary.NativeEndian.Uint16(record[direntReclen:]))
			name := record[direntName:]
			end := bytes.IndexByte(name, 0)
			if end < 0 {
				return true
			}
			name = name[:end]
			if string(name) == "." || string(name) == ".." {
				continue
			}
			nameString := unsafe.String(unsafe.SliceData(name), len(name))
			w.walker.path = append(append(w.walker.path[:dirLength], '/'), name...)
			path := unsafe.String(unsafe.SliceData(w.walker.path), len(w.walker.path))

			kind := record[direntType]
			// files need their stat anyway, and DT_UNKNOWN (some file systems) could be a folder
			if kind != unix.DT_DIR {
				if unix.Fstatat(fd, nameString, &w.walker.stat, unix.AT_SYMLINK_NOFOLLOW) != nil {
					continue
				}
				if w.walker.stat.Mode&unix.S_IFMT == unix.S_IFDIR {
					kind = unix.DT_DIR
				}
			}
			if kind == unix.DT_DIR {
				if w.skipDir(path, nameString) {
					continue
				}
				child, err := unix.Openat(fd, nameString, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
				if err != nil {
					continue
				}
				if !w.walkDir(child, depth+1, fn) {
					return false
				}
				continue
			}
			if !fn(path, watchedFile{modTime: w.walker.stat.Mtim.Nano(), size: w.walker.stat.Size, exists: true}) {
				return false
			}
		}
	}
}
