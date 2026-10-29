// Package transform is the port of roblox-ts's src/TSTransformer: TypeScript AST -> Luau AST.
//
// Files and functions keep the roblox-ts names (transformSourceFile.go has transformSourceFile, ...)
// so that changes to roblox-ts can be ported by diffing file against file. roblox-ts's TSTransformer
// is one big cycle of mutually recursive modules, so this is one Go package. Files of the
// util/evaluation directory are prefixed with evaluation_, and macros/types.ts is macros_types.go.
//
// Conventions of the port (see docs/goblox/PORTING.md):
//   - ts.Node and all its subtypes are *ast.Node; ts.SourceFile is *ast.SourceFile; ts.Type is
//     *checker.Type; ts.Symbol is *ast.Symbol. TypeScript's Type methods (getProperty, getConstraint,
//     ...) are the typeX helpers of types.go.
//   - luau.Expression, luau.Statement, ... are aliases of luau.Node; luau.List<T> is *luau.List.
//   - TypeScript `number` is float64 where it is a JavaScript number value and int where it is a count
//     or an index; `T | undefined` is nil for pointers and interfaces and an extra bool otherwise.
//   - MultiTransformState is per checker, so its caches need no locking.
package transform

// COMPILER_VERSION is the roblox-ts version goblox-ts is a port of (package.json of roblox-ts
// 68648b0); it is printed in the header of every compiled file.
const COMPILER_VERSION = "3.0.0"

const (
	PARENT_FIELD = "Parent"
	RBXTS_SCOPE  = "@rbxts"
)
