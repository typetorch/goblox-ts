package project

import (
	"regexp"
	"slices"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/vfs"
)

// TypeScript 7 compatibility for roblox-ts projects.
//
// 1. @rbxts/compiler-types (3.0.0-types.0 and master as of 2026-10) declares `Iterable<T>`,
//    `IterableIterator<T>`, `AsyncIterable<T>` and `AsyncIterableIterator<T>` with one type parameter.
//    TypeScript 5.x falls back to looking up `[Symbol.iterator]()` structurally, but TypeScript 7 only
//    iterates non-array types when the global `Iterable` has the lib's three type parameters
//    (`Iterable<T, TReturn = any, TNext = any>`), so every `for..of` over an IterableFunction, Map or
//    Set fails with TS2802. compatFS rewrites those four declarations when the file is read.
// 2. roblox-ts requires `"moduleResolution": "Node"` and recommends `"downlevelIteration": true`. Both
//    options were removed in TypeScript 7 (TS5108, TS5102). Node10 resolution falls back to
//    `bundler`, and downlevelIteration has no effect on the checker, so those two diagnostics are
//    dropped by FilterCompatDiagnostics.

var compilerTypesIterableArity = regexp.MustCompile(`(?m)^interface (AsyncIterable|Iterable|AsyncIterableIterator|IterableIterator)<T> `)

const compilerTypesIterableFile = "/@rbxts/compiler-types/types/Iterable.d.ts"

type compatFS struct {
	vfs.FS
}

// WrapCompatFS returns fs with the TypeScript 7 compatibility rewrites applied on read.
func WrapCompatFS(fs vfs.FS) vfs.FS {
	return &compatFS{FS: fs}
}

func (f *compatFS) ReadFile(path string) (string, bool) {
	contents, ok := f.FS.ReadFile(path)
	if ok && strings.HasSuffix(path, compilerTypesIterableFile) {
		contents = compilerTypesIterableArity.ReplaceAllString(contents, "interface $1<T, TReturn = any, TNext = any> ")
	}
	return contents, ok
}

// isCompatDiagnostic reports whether d is a "removed option" error for an option roblox-ts requires.
func isCompatDiagnostic(d *ast.Diagnostic) bool {
	switch d.Code() {
	case 5102: // Option '{0}' has been removed. Please remove it from your configuration.
		return slices.Contains(d.MessageArgs(), "downlevelIteration")
	case 5108: // Option '{0}={1}' has been removed. Please remove it from your configuration.
		return len(d.MessageArgs()) > 0 && d.MessageArgs()[0] == "moduleResolution"
	}
	return false
}

// FilterCompatDiagnostics removes the diagnostics that FilterCompatDiagnostics documents.
func FilterCompatDiagnostics(diags []*ast.Diagnostic) []*ast.Diagnostic {
	return slices.DeleteFunc(slices.Clone(diags), isCompatDiagnostic)
}
