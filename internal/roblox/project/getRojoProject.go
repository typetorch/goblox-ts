package project

// Port of Project/functions/getRojoProject.ts: only the read and parse errors. The config dependencies it
// collects are for roblox-ts's watch mode, which goblox does not port.

import (
	"encoding/json"
	"maps"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

var (
	rojoProjectFilePattern = regexp.MustCompile(`^.+\.project\.json$`)
	rojoLeafFilePattern    = regexp.MustCompile(`\.(?:lua|luau|json|toml)$`)
)

func checkRojoProject(fs vfs.FS, configPath string) error {
	files := map[string]bool{}
	visited := map[string]bool{}

	var visitConfig func(filePath string) error
	var visitTree func(basePath string, tree any) error

	// rojo-resolver does not expose its config dependencies, so follow its directory discovery rules
	var visitDirectory func(directory string) error
	visitDirectory = func(directory string) error {
		if !fs.DirectoryExists(directory) {
			return nil
		}
		realPath := fs.Realpath(directory)
		if visited[realPath] {
			return nil
		}
		visited[realPath] = true

		entries := fs.GetAccessibleEntries(directory)
		if slices.Contains(entries.Files, "default.project.json") {
			return visitConfig(nodepath.Join(directory, "default.project.json"))
		}
		for _, entry := range entries.Directories {
			if err := visitDirectory(nodepath.Join(directory, entry)); err != nil {
				return err
			}
		}
		for _, entry := range entries.Files {
			if rojoProjectFilePattern.MatchString(entry) {
				if err := visitConfig(nodepath.Join(directory, entry)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	visitTree = func(basePath string, tree any) error {
		object, ok := tree.(map[string]any)
		if !ok {
			return nil
		}
		// Object.entries keeps the file's order; only which of several broken configs is reported depends on it
		for _, key := range slices.Sorted(maps.Keys(object)) {
			value := object[key]
			if key == "$path" {
				directory, isString := value.(string)
				if optional, isObject := value.(map[string]any); isObject {
					directory, isString = optional["optional"].(string)
				}
				if !isString {
					continue
				}
				resolved := nodepath.Resolve(basePath, directory)
				if rojoProjectFilePattern.MatchString(nodepath.Basename(resolved)) {
					// Rojo nests a referenced project file
					if fs.FileExists(resolved) {
						if err := visitConfig(resolved); err != nil {
							return err
						}
					}
				} else if !rojoLeafFilePattern.MatchString(resolved) {
					// explicit module files are leaves, including JSON files
					if err := visitDirectory(resolved); err != nil {
						return err
					}
				}
			} else if !strings.HasPrefix(key, "$") {
				if err := visitTree(basePath, value); err != nil {
					return err
				}
			}
		}
		return nil
	}

	visitConfig = func(filePath string) error {
		if files[filePath] {
			return nil
		}
		files[filePath] = true
		contents, ok := fs.ReadFile(filePath)
		if !ok {
			native := filepath.FromSlash(filePath)
			if fs.DirectoryExists(filePath) {
				return &rojoProjectReadError{"Error: EISDIR: illegal operation on a directory, read"}
			}
			return &rojoProjectReadError{"Error: ENOENT: no such file or directory, open '" + native + "'"}
		}
		var config struct {
			Tree any `json:"tree"`
		}
		if err := json.Unmarshal([]byte(contents), &config); err != nil {
			// JSON.parse's messages are V8's; this keeps the kind of error
			return &rojoProjectReadError{"SyntaxError: " + err.Error()}
		}
		return visitTree(nodepath.Dirname(filePath), config.Tree)
	}

	if err := visitConfig(configPath); err != nil {
		message := `Unable to read Rojo project "` + filepath.FromSlash(configPath) + `": ` + err.Error()
		return &diagnostics.DiagnosticError{Diagnostics: []*ast.Diagnostic{
			diagnostics.CreateTextDiagnostic(message, tsdiagnostics.CategoryError),
		}}
	}
	return nil
}

// rojoProjectReadError is String(error) of what fs.readFileSync or JSON.parse threw.
type rojoProjectReadError struct{ message string }

func (e *rojoProjectReadError) Error() string { return e.message }
