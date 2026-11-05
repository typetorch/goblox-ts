//go:build unix

package main

import (
	"errors"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// handoffDir is $XDG_RUNTIME_DIR/goblox-<uid>, or the same folder in the temporary folder: made by and for
// this user only, so nobody else can listen there or connect.
func handoffDir() (string, bool) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" || !filepath.IsAbs(base) {
		base = os.TempDir()
	}
	uid := os.Getuid()
	dir := filepath.Join(base, "goblox-"+strconv.Itoa(uid))
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", false
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != uid {
		return "", false
	}
	if info.Mode().Perm() != 0o700 {
		if err := os.Chmod(dir, 0o700); err != nil {
			return "", false
		}
	}
	return dir, true
}

// exitSignals end a watcher.
var exitSignals = []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}

// exitOnSignal ends the process by sig, as if it had no handler.
func exitOnSignal(sig os.Signal) {
	signal.Reset()
	if s, ok := sig.(syscall.Signal); ok {
		_ = syscall.Kill(os.Getpid(), s)
		time.Sleep(time.Second)
		os.Exit(128 + int(s))
	}
	os.Exit(1)
}
