package project

// Port of Project/functions/copyInclude.ts and Project/functions/getIncludeFiles.ts. The roblox-ts
// include folder (RuntimeLib and Promise, MIT, roblox-ts 68648b0) is embedded in the binary.

import (
	"embed"
	"io/fs"
	"os"
	"strings"

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

// outputFile is fs-extra's outputFileSync: it creates the parent directories.
func outputFile(path string, contents []byte) error {
	if err := os.MkdirAll(nodepath.Dirname(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, contents, 0o644)
}
