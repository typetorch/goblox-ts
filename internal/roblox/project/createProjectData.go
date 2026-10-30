package project

// Port of Project/functions/createProjectData.ts and Project/functions/validateCompilerOptions.ts.

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/json"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/roblox/transform"
	"github.com/microsoft/typescript-go/internal/vfs"
)

const (
	NODE_MODULES = "node_modules"
	RBXTS_SCOPE  = "@rbxts"
	DTS_EXT      = ".d.ts"
	LUA_EXT      = ".lua"
	LUAU_EXT     = ".luau"
)

// npm allows any non-empty scope, including `.`, `_`, and `~`
var packageRegex = regexp.MustCompile(`^@[^/]+/`)

// findPackageJson is ts.findPackageJson: the nearest package.json at or above directory, "" if none.
func findPackageJson(fs vfs.FS, directory string) string {
	for {
		candidate := nodepath.Join(directory, "package.json")
		if fs.FileExists(candidate) {
			return candidate
		}
		parent := nodepath.Dirname(directory)
		if parent == directory {
			return ""
		}
		directory = parent
	}
}

func createProjectData(fs vfs.FS, tsConfigPath string, projectOptions transform.ProjectOptions) (*transform.ProjectData, error) {
	projectPath := nodepath.Dirname(tsConfigPath)

	pkgJsonPath := findPackageJson(fs, projectPath)
	if pkgJsonPath == "" {
		return nil, &diagnostics.ProjectError{Message: "Unable to find package.json"}
	}

	isPackage := false
	if text, ok := fs.ReadFile(pkgJsonPath); ok {
		var pkgJson struct {
			Name string `json:"name,case:ignore"`
		}
		// errors if no pkgJson, so assume not a package
		if json.Unmarshal([]byte(text), &pkgJson, json.AllowDuplicateNames(true)) == nil {
			isPackage = packageRegex.MatchString(pkgJson.Name)
		}
	}

	// intentionally use || here for empty string case
	includePath := projectOptions.IncludePath
	if includePath == "" {
		includePath = nodepath.Join(projectPath, "include")
	}
	projectOptions.IncludePath = nodepath.Resolve(includePath)

	nodeModulesPath := nodepath.Join(nodepath.Dirname(pkgJsonPath), NODE_MODULES)

	var rojoConfigPath string
	// Checking truthiness covers empty string case
	if projectOptions.Rojo != "" {
		rojoConfigPath = nodepath.Resolve(projectOptions.Rojo)
	} else {
		path, warnings := rojo.FindRojoConfigFilePath(fs, projectPath)
		rojoConfigPath = path
		for _, warning := range warnings {
			LogService.Warn(warning)
		}
	}

	return &transform.ProjectData{
		TsConfigPath:    tsConfigPath,
		IsPackage:       isPackage,
		NodeModulesPath: nodeModulesPath,
		ProjectOptions:  projectOptions,
		ProjectPath:     projectPath,
		RojoConfigPath:  rojoConfigPath,
	}, nil
}

/** shorthand for kleur.yellow */
func y(str string) string { return diagnostics.Yellow(str) }

func validateTypeRoots(typeRoots []string) bool {
	for _, typeRoot := range typeRoots {
		resolvedPath := nodepath.Resolve(typeRoot)

		// allow hoisted dependencies without requiring VirtualProject's directories to exist on disk
		if nodepath.Basename(resolvedPath) == RBXTS_SCOPE && nodepath.Basename(nodepath.Dirname(resolvedPath)) == NODE_MODULES {
			return true
		}
	}
	return false
}

// validateCompilerOptions is validateCompilerOptions.ts. TypeScript 7 removed `moduleResolution: node10`
// and `importsNotUsedAsValues`: typescript-go still parses "Node" as Node10 (and reports TS5108, which
// FilterCompatDiagnostics drops), and rejects importsNotUsedAsValues itself.
func validateCompilerOptions(fs vfs.FS, opts *core.CompilerOptions, projectPath string) error {
	var errors []string

	// required compiler options
	if !opts.NoLib.IsTrue() {
		errors = append(errors, y(`"noLib"`)+" must be "+y(`true`))
	}

	if !opts.Strict.IsTrue() {
		errors = append(errors, y(`"strict"`)+" must be "+y(`true`))
	}

	if opts.Module != core.ModuleKindCommonJS {
		errors = append(errors, y(`"module"`)+" must be "+y(`commonjs`))
	}

	if opts.ModuleDetection != core.ModuleDetectionKindForce {
		errors = append(errors, y(`"moduleDetection"`)+" must be "+y(`"force"`))
	}

	if opts.ModuleResolution != core.ModuleResolutionKindNode10 {
		errors = append(errors, y(`"moduleResolution"`)+" must be "+y(`"Node"`))
	}

	if !opts.AllowSyntheticDefaultImports.IsTrue() {
		errors = append(errors, y(`"allowSyntheticDefaultImports"`)+" must be "+y(`true`))
	}

	rbxtsModules := nodepath.Join(projectPath, NODE_MODULES, RBXTS_SCOPE)
	if opts.TypeRoots == nil || !validateTypeRoots(opts.TypeRoots) {
		errors = append(errors, y(`"typeRoots"`)+" must contain a node_modules/@rbxts directory, such as "+y(filepath.FromSlash(rbxtsModules)))
	}

	for _, typesLocation := range opts.Types {
		if typesLocation == "*" {
			// goblox sets types to "*" for TypeScript 5's automatic type directives (see LoadProgram)
			continue
		}
		// Technically checked to exist above
		// But if that's an error, we still want this error too,
		// To avoid "fix one error, get a new one on the next compile"
		typeRoots := opts.TypeRoots
		if typeRoots == nil {
			typeRoots = []string{"node_modules/@rbxts"}
		}

		found := false
		for _, typeRoot := range typeRoots {
			typesPath := nodepath.Resolve(projectPath, typeRoot, typesLocation)
			if fs.FileExists(typesPath) || fs.DirectoryExists(typesPath) || fs.FileExists(typesPath+DTS_EXT) {
				found = true
				break
			}
		}
		if !found {
			errors = append(errors, y(`"types"`)+" "+y(typesLocation)+" were not found. Make sure the path is relative to `typeRoots`")
		}
	}

	// configurable compiler options
	if opts.RootDir == "" && opts.RootDirs == nil {
		errors = append(errors, y(`"rootDir"`)+" or "+y(`"rootDirs"`)+" must be defined")
	}

	if opts.OutDir == "" {
		errors = append(errors, y(`"outDir"`)+" must be defined")
	}

	// throw if errors
	if len(errors) > 0 {
		var list strings.Builder
		for _, e := range errors {
			list.WriteString("- " + e + "\n")
		}
		return &diagnostics.ProjectError{Message: strings.Join([]string{
			`Invalid "tsconfig.json" configuration!`,
			`https://roblox-ts.com/docs/quick-start#project-folder-setup`,
			list.String(),
		}, "\n")}
	}
	return nil
}
