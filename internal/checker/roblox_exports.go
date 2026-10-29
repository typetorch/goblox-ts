package checker

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

// Exports for goblox-ts (internal/roblox): checker helpers that TypeScript's public API exposes but
// typescript-go keeps unexported. This is the only upstream package goblox-ts adds a file to; keep
// every such export here so upstream merges conflict in one place.

// GetIndexTypeOfType is TypeScript's checker.getIndexTypeOfType.
func (c *Checker) GetIndexTypeOfType(t *Type, keyType *Type) *Type {
	return c.getIndexTypeOfType(t, keyType)
}

// GetStringLiteralType is TypeScript's checker.getStringLiteralType.
func (c *Checker) GetStringLiteralType(value string) *Type {
	return c.getStringLiteralType(value)
}

// GetConstraintOfType is TypeScript's checker.getConstraintOfType.
func (c *Checker) GetConstraintOfType(t *Type) *Type {
	return c.getConstraintOfType(t)
}

// GetTypeOfAssignmentPattern is TypeScript's checker.getTypeOfAssignmentPattern.
func (c *Checker) GetTypeOfAssignmentPattern(expr *ast.Node) *Type {
	return c.getTypeOfAssignmentPattern(expr)
}

// CreateArrayType is TypeScript's checker.createArrayType.
func (c *Checker) CreateArrayType(elementType *Type) *Type {
	return c.createArrayType(elementType)
}

// GetTrueType is TypeScript's checker.getTrueType (a field in typescript-go).
func (c *Checker) GetTrueType() *Type { return c.trueType }

// GetFalseType is TypeScript's checker.getFalseType (a field in typescript-go).
func (c *Checker) GetFalseType() *Type { return c.falseType }

// Checker is TypeScript's Type.checker: the checker that created t.
func (t *Type) Checker() *Checker { return t.checker }

// GetAwaitedType is TypeScript's checker.getAwaitedType.
func (c *Checker) GetAwaitedType(t *Type) *Type {
	return c.getAwaitedType(t)
}

// GetNumberLiteralType is TypeScript's checker.getNumberLiteralType.
func (c *Checker) GetNumberLiteralType(value jsnum.Number) *Type {
	return c.getNumberLiteralType(value)
}

// Root is TypeScript's ConditionalType.root.
func (t *ConditionalType) Root() *ConditionalRoot { return t.root }

// Node is TypeScript's ConditionalRoot.node.
func (r *ConditionalRoot) Node() *ast.ConditionalTypeNode { return r.node }

// CheckType is TypeScript's ConditionalRoot.checkType.
func (r *ConditionalRoot) CheckType() *Type { return r.checkType }
