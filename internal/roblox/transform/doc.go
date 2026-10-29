// Package transform is the port of roblox-ts's src/TSTransformer: TypeScript AST -> Luau AST.
//
// Files and functions keep the roblox-ts names (transformSourceFile.go has transformSourceFile, ...)
// so that changes to roblox-ts can be ported by diffing file against file. roblox-ts's TSTransformer
// is one big cycle of mutually recursive modules, so this is one Go package.
//
// Ported so far (Phase 1 slice): source files and exports, statement lists with comments and
// hoisting, imports (named, default, namespace), variable statements with identifier names,
// function declarations and expressions, calls, property access, literals, template strings,
// binary/unary/logical operators and assignments, if/while/do/for, return, break/continue.
// Anything else reports errors.notYetSupported (see unsupported.go) instead of guessing.
package transform

// COMPILER_VERSION is the roblox-ts version goblox-ts is a port of (package.json of roblox-ts
// 68648b0); it is printed in the header of every compiled file.
const COMPILER_VERSION = "3.0.0"

const (
	PARENT_FIELD = "Parent"
	RBXTS_SCOPE  = "@rbxts"
)
