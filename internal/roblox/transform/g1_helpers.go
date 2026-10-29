package transform

// Small helpers for the import/export, namespace and enum ports that have no file of their own in
// roblox-ts (Shared/util functions, TransformState methods and TypeScript internals that goblox
// does not expose elsewhere).

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// getCanonicalFileName is Shared/util/getCanonicalFileName.ts with the program's case sensitivity.
func (s *TransformState) getCanonicalFileName(fileName string) string {
	return tspath.GetCanonicalFileName(fileName, s.Program.UseCaseSensitiveFileNames())
}

// guessVirtualPath is TransformState.guessVirtualPath: the path of fsPath through the symlink that
// resolved to one of its parent directories, or "" (undefined) when there is none.
func (s *TransformState) guessVirtualPath(fsPath string) string {
	reverseSymlinkMap := s.Program.GetSymlinkCache().DirectoriesByRealpath()

	original := fsPath
	for {
		// reverseSymlinkMap always has trailing slashes
		// as it is constructed from `SymlinkedDirectory.real`
		parent := tspath.EnsureTrailingDirectorySeparator(nodepath.Dirname(fsPath))
		if fsPath == parent {
			break
		}
		fsPath = parent
		key := tspath.ToPath(fsPath, s.Program.GetCurrentDirectory(), s.Program.UseCaseSensitiveFileNames()).EnsureTrailingDirectorySeparator()
		if symlinks, ok := reverseSymlinkMap.Load(key); ok {
			symlink := ""
			symlinks.Range(func(value string) bool {
				symlink = value
				return false
			})
			if symlink != "" {
				return nodepath.Join(symlink, nodepath.Relative(fsPath, original))
			}
		}
	}
	return ""
}

// isInsideNodeModules is TypeScript's isInsideNodeModules (pathContainsNodeModules).
func isInsideNodeModules(fileOrDirectory string) bool {
	return strings.Contains(fileOrDirectory, "/node_modules/")
}

// realPathExistsSync is Shared/util/realPathExistsSync.ts; it returns "" (undefined) when fsPath
// does not exist.
func (s *TransformState) realPathExistsSync(fsPath string) string {
	fs := s.Program.Host().FS()
	if fs.FileExists(fsPath) || fs.DirectoryExists(fsPath) {
		return fs.Realpath(fsPath)
	}
	return ""
}
