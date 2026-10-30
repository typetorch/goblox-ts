package transform

// Port of TSTransformer/util/createImportExpression.ts.

import (
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// NODE_MODULES is Shared/constants.ts NODE_MODULES.
const NODE_MODULES = "node_modules"

func getAbsoluteImport(moduleRbxPath rojo.RbxPath) []luau.Expression {
	var pathExpressions []luau.Expression
	assert(len(moduleRbxPath) > 0)
	serviceName := moduleRbxPath[0]
	assert(serviceName != "")
	pathExpressions = append(pathExpressions, createGetService(serviceName))
	for i := 1; i < len(moduleRbxPath); i++ {
		pathExpressions = append(pathExpressions, luau.String(moduleRbxPath[i]))
	}
	return pathExpressions
}

func getRelativeImport(sourceRbxPath rojo.RbxPath, moduleRbxPath rojo.RbxPath) []luau.Expression {
	relativePath := rojo.Relative(sourceRbxPath, moduleRbxPath)

	// create descending path pieces
	var path []string
	i := 0
	for i < len(relativePath) && relativePath[i].IsParent {
		path = append(path, PARENT_FIELD)
		i++
	}

	pathExpressions := []luau.Expression{propertyAccessExpressionChain(luau.Globals.Script, path)}

	// create descending path pieces
	for ; i < len(relativePath); i++ {
		pathPart := relativePath[i]
		assert(!pathPart.IsParent)
		pathExpressions = append(pathExpressions, luau.String(pathPart.Name))
	}

	return pathExpressions
}

func validateModule(state *TransformState, scope string) bool {
	scopedModules := nodepath.Join(state.Data.NodeModulesPath, scope)
	assert(state.CompilerOptions.TypeRoots != nil)
	for _, typeRoot := range state.CompilerOptions.TypeRoots {
		if nodepath.Normalize(scopedModules) == nodepath.Normalize(typeRoot) {
			return true
		}
	}
	return false
}

func findRelativeRbxPath(moduleOutPath string, pkgRojoResolvers []*rojo.RojoResolver) rojo.RbxPath {
	for _, pkgRojoResolver := range pkgRojoResolvers {
		relativeRbxPath := pkgRojoResolver.GetRbxPathFromFilePath(moduleOutPath)
		if relativeRbxPath != nil {
			return relativeRbxPath
		}
	}
	return nil
}

func getNodeModulesImportParts(
	state *TransformState,
	sourceFile *ast.SourceFile,
	moduleSpecifier *ast.Node,
	moduleOutPath string,
	moduleFilename string,
) []luau.Expression {
	relativePath := nodepath.Relative(state.Data.NodeModulesPath, moduleOutPath)
	moduleScope, _, _ := strings.Cut(relativePath, nodepath.Sep)
	assert(moduleScope != "")

	if !isPathDescendantOf(moduleOutPath, state.Data.NodeModulesPath) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.FailedSymlinkResolve.Create(
			moduleSpecifier,
			state.Data.NodeModulesPath,
			moduleFilename,
			state.guessVirtualPath(moduleFilename),
			relativePath,
		))
		return []luau.Expression{luau.NewNone()}
	} else if !strings.HasPrefix(moduleScope, "@") {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoUnscopedModule.Create(moduleSpecifier))
		return []luau.Expression{luau.NewNone()}
	}

	if !validateModule(state, moduleScope) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoInvalidScope.Create(moduleSpecifier))
		return []luau.Expression{luau.NewNone()}
	}

	if state.ProjectType == ProjectTypePackage {
		relativeRbxPath := findRelativeRbxPath(moduleOutPath, state.PkgRojoResolvers)
		if relativeRbxPath == nil {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRojoData.Create(
				moduleSpecifier, nodepath.Relative(state.Data.ProjectPath, moduleOutPath), true,
			))
			return []luau.Expression{luau.NewNone()}
		}

		assert(len(relativeRbxPath) > 0)
		moduleName := relativeRbxPath[0]
		assert(moduleName != "")

		return []luau.Expression{
			propertyAccessExpressionChain(
				luau.Call(state.TS(moduleSpecifier.Parent, "getModule"),
					luau.Globals.Script,
					luau.String(moduleScope),
					luau.String(moduleName),
				),
				relativeRbxPath[1:],
			),
		}
	}

	moduleRbxPath := state.RojoResolver.GetRbxPathFromFilePath(moduleOutPath)
	if moduleRbxPath == nil {
		// Rojo follows symlinks, so a config may mount a package by its real location (e.g. the pnpm store)
		// resolve the directory, since the output extension (.luau) may differ from the shipped file (init.lua)
		realModuleDir := state.realPathExistsSync(nodepath.Dirname(moduleOutPath))
		if realModuleDir != "" {
			moduleRbxPath = state.RojoResolver.GetRbxPathFromFilePath(
				nodepath.Join(realModuleDir, nodepath.Basename(moduleOutPath)),
			)
		}
	}
	if moduleRbxPath == nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRojoData.Create(
			moduleSpecifier, nodepath.Relative(state.Data.ProjectPath, moduleOutPath), true,
		))
		return []luau.Expression{luau.NewNone()}
	}

	indexOfScope := slices.Index(moduleRbxPath, moduleScope)
	if indexOfScope == -1 || indexOfScope == 0 || moduleRbxPath[indexOfScope-1] != NODE_MODULES {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoPackageImportWithoutScope.Create(
			moduleSpecifier,
			nodepath.Relative(state.Data.ProjectPath, moduleOutPath),
			moduleRbxPath,
		))
		return []luau.Expression{luau.NewNone()}
	}

	return getProjectImportParts(state, sourceFile, moduleSpecifier, moduleOutPath, moduleRbxPath)
}

func getProjectImportParts(
	state *TransformState,
	sourceFile *ast.SourceFile,
	moduleSpecifier *ast.Node,
	moduleOutPath string,
	moduleRbxPath rojo.RbxPath,
) []luau.Expression {
	moduleRbxType := state.RojoResolver.GetRbxTypeFromFilePath(moduleOutPath)
	if moduleRbxType == rojo.RbxTypeScript || moduleRbxType == rojo.RbxTypeLocalScript {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoNonModuleImport.Create(moduleSpecifier))
		return []luau.Expression{luau.NewNone()}
	}

	sourceOutPath := state.PathTranslator.GetOutputPath(sourceFile.FileName())
	sourceRbxPath := state.RojoResolver.GetRbxPathFromFilePath(sourceOutPath)
	if sourceRbxPath == nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRojoData.CreateWithRange(
			sourceFile, sourceFile.Loc, nodepath.Relative(state.Data.ProjectPath, sourceOutPath), false,
		))
		return []luau.Expression{luau.NewNone()}
	}

	if state.ProjectType == ProjectTypeGame {
		// in the case of `import("")`, don't do network type check
		// as the call may be guarded by runtime RunService checks
		if !ast.IsImportCall(moduleSpecifier.Parent) &&
			state.RojoResolver.GetNetworkType(moduleRbxPath) == rojo.NetworkTypeServer &&
			state.RojoResolver.GetNetworkType(sourceRbxPath) != rojo.NetworkTypeServer {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoServerImport.Create(moduleSpecifier))
			return []luau.Expression{luau.NewNone()}
		}

		fileRelation := state.RojoResolver.GetFileRelation(sourceRbxPath, moduleRbxPath)
		if fileRelation == rojo.FileRelationOutToOut || fileRelation == rojo.FileRelationInToOut {
			return getAbsoluteImport(moduleRbxPath)
		} else if fileRelation == rojo.FileRelationInToIn {
			return getRelativeImport(sourceRbxPath, moduleRbxPath)
		} else {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIsolatedImport.Create(moduleSpecifier))
			return []luau.Expression{luau.NewNone()}
		}
	}
	return getRelativeImport(sourceRbxPath, moduleRbxPath)
}

// getImportParts is TSTransformer/util/createImportExpression.ts getImportParts.
func getImportParts(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) []luau.Expression {
	moduleFile := getSourceFileFromModuleSpecifier(state, moduleSpecifier)
	if moduleFile == nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoModuleSpecifierFile.Create(moduleSpecifier))
		return []luau.Expression{luau.NewNone()}
	}

	virtualPath := state.guessVirtualPath(moduleFile.FileName())
	if virtualPath == "" {
		virtualPath = moduleFile.FileName()
	}
	referencePath, hasReferencePath := state.Data.ProjectReferencePaths[state.getCanonicalFileName(nodepath.Normalize(virtualPath))]

	if !hasReferencePath && isInsideNodeModules(virtualPath) {
		mappedPath, ok := state.NodeModulesPathMapping[state.getCanonicalFileName(nodepath.Normalize(virtualPath))]
		if !ok {
			mappedPath = virtualPath
		}
		moduleOutPath := state.PathTranslator.GetImportPath(mappedPath, true /* isNodeModule */)
		return getNodeModulesImportParts(state, sourceFile, moduleSpecifier, moduleOutPath, moduleFile.FileName())
	}

	moduleOutPath := referencePath
	if !hasReferencePath {
		moduleOutPath = state.PathTranslator.GetImportPath(virtualPath, false)
	}

	if !hasReferencePath && !isPathDescendantOf(virtualPath, state.PathTranslator.RootDir) {
		// hand-written declarations and JSON outside the source roots describe files mapped directly by Rojo
		if tspath.IsDeclarationFileName(virtualPath) {
			ext := ".lua"
			if state.Data.ProjectOptions.Luau {
				ext = ".luau"
			}
			moduleOutPath = virtualPath[:len(virtualPath)-len(".d.ts")] + ext
		} else {
			// TypeScript rejects non-declaration source files outside rootDir before this transform
			assert(ast.IsJsonSourceFile(moduleFile))
			moduleOutPath = virtualPath
		}
	}

	moduleRbxPath := state.RojoResolver.GetRbxPathFromFilePath(moduleOutPath)
	if moduleRbxPath == nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRojoData.Create(
			moduleSpecifier, nodepath.Relative(state.Data.ProjectPath, moduleOutPath), false,
		))
		return []luau.Expression{luau.NewNone()}
	}

	return getProjectImportParts(state, sourceFile, moduleSpecifier, moduleOutPath, moduleRbxPath)
}

// createImportExpression is TSTransformer/util/createImportExpression.ts createImportExpression.
func createImportExpression(state *TransformState, sourceFile *ast.SourceFile, moduleSpecifier *ast.Node) luau.IndexableExpression {
	parts := getImportParts(state, sourceFile, moduleSpecifier)
	parts = append([]luau.Expression{luau.Globals.Script}, parts...)
	return luau.Call(state.TS(moduleSpecifier.Parent, "import"), parts...)
}
