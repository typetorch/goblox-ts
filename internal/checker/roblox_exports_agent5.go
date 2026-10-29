package checker

import "github.com/microsoft/typescript-go/internal/ast"

// Small exports used by goblox's roblox transform (agent 5). Kept apart from roblox_exports.go so that
// agents do not conflict on one file.

// Root is TypeScript's `ConditionalType.root`.
func (t *ConditionalType) Root() *ConditionalRoot { return t.root }

// Node is TypeScript's `ConditionalRoot.node`.
func (r *ConditionalRoot) Node() *ast.ConditionalTypeNode { return r.node }

// CheckType is TypeScript's `ConditionalRoot.checkType`.
func (r *ConditionalRoot) CheckType() *Type { return r.checkType }
