package project

// Port of Project/functions/copyInclude.ts and Project/functions/getIncludeFiles.ts. The roblox-ts
// include folder (RuntimeLib and Promise, MIT, roblox-ts 68648b0) is embedded in the binary.

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
)

//go:embed include
var includeFS embed.FS

type includeFile struct {
	// input is the path inside includeFS
	input  string
	output string
}

func getIncludeFiles(options *transform.ProjectOptions) []includeFile {
	var files []includeFile
	_ = fs.WalkDir(includeFS, "include", func(input string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		outputName := strings.TrimPrefix(input, "include/")
		if !options.Luau && strings.HasSuffix(outputName, LUAU_EXT) {
			outputName = strings.TrimSuffix(outputName, LUAU_EXT) + LUA_EXT
		}
		files = append(files, includeFile{input: input, output: nodepath.Join(options.IncludePath, outputName)})
		return nil
	})
	return files
}

func copyInclude(data *transform.ProjectData) error {
	if !data.ProjectOptions.NoInclude &&
		data.ProjectOptions.Type != transform.ProjectTypePackage &&
		!(data.ProjectOptions.Type == "" && data.IsPackage) {
		for _, file := range getIncludeFiles(&data.ProjectOptions) {
			if strings.HasSuffix(file.input, LUAU_EXT) {
				// Rojo must not see both extensions as competing instances after an option change
				extension, alternateExtension := LUAU_EXT, LUA_EXT
				if !data.ProjectOptions.Luau {
					extension, alternateExtension = LUA_EXT, LUAU_EXT
				}
				alternate := strings.TrimSuffix(file.output, extension) + alternateExtension
				if _, err := os.Stat(alternate); err == nil {
					if err := os.Remove(alternate); err != nil {
						return err
					}
				}
			}
			contents, err := includeFS.ReadFile(file.input)
			if err != nil {
				return err
			}
			if err := outputFile(file.output, contents); err != nil {
				return err
			}
		}
	}
	return nil
}

// outputFile is fs-extra's outputFileSync: it creates the parent directories. The contents go to a
// temporary file beside path that is then renamed over it, so a build that is killed (or a watcher that a
// handed-off build loses) leaves the old file or the new one, never part of one. A symbolic link is written
// through, as fs-extra does.
func outputFile(path string, contents []byte) error {
	if err := os.MkdirAll(nodepath.Dirname(path), 0o755); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return os.WriteFile(path, contents, 0o644)
	}
	temporary, err := writeTemporary(path, contents)
	if err != nil {
		return err
	}
	if err := renameOver(temporary, path); err != nil {
		_ = os.Remove(temporary)
		// a file another program holds open without FILE_SHARE_DELETE (Windows) can't be replaced, only
		// rewritten, as before
		return os.WriteFile(path, contents, 0o644)
	}
	return nil
}

// temporaryCounter tells apart the temporary files of one process.
var temporaryCounter atomic.Uint64

// writeTemporary writes contents to a new hidden file next to path. Its name ends in .tmp, which Rojo
// does not sync, and the next build's clean step removes one that a killed build left behind.
func writeTemporary(path string, contents []byte) (string, error) {
	directory, base := nodepath.Dirname(path), nodepath.Basename(path)
	for {
		temporary := nodepath.Join(directory, fmt.Sprintf(".%s.%d-%d.tmp", base, os.Getpid(), temporaryCounter.Add(1)))
		file, err := os.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = file.Write(contents)
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(temporary)
			return "", err
		}
		return temporary, nil
	}
}

// renameOver renames from to to, replacing it. On Windows a reader that has the file open (Rojo, an
// editor, a virus scanner) makes the rename fail for a moment, so it is retried a few times.
func renameOver(from string, to string) error {
	var err error
	for attempt := range 5 {
		if err = os.Rename(from, to); err == nil {
			return nil
		}
		time.Sleep(time.Duration(1<<attempt) * time.Millisecond)
	}
	return err
}
