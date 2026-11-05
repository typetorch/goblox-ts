package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

// handoffDir is the goblox folder of the user's temporary folder.
func handoffDir() (string, bool) {
	dir := filepath.Join(os.TempDir(), "goblox")
	if err := os.Mkdir(dir, 0o700); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", false
	}
	// not a link or junction to somewhere else
	if info, err := os.Lstat(dir); err != nil || !info.IsDir() {
		return "", false
	}
	return dir, true
}

// exitSignals end a watcher: Ctrl+C or Ctrl+Break.
var exitSignals = []os.Signal{os.Interrupt}

// statusControlCExit is the exit code of a console program that Ctrl+C ends without a handler.
var statusControlCExit uint32 = 0xC000013A

func exitOnSignal(os.Signal) {
	os.Exit(int(statusControlCExit))
}
