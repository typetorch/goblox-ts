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

// GetTrueType is TypeScript's checker.getTrueType() (a field in typescript-go). Without the `fresh`
// argument TypeScript returns the regular literal type, which is what declared `true` types resolve to.
func (c *Checker) GetTrueType() *Type { return c.regularTrueType }

// GetFalseType is TypeScript's checker.getFalseType() (a field in typescript-go), the regular literal type.
func (c *Checker) GetFalseType() *Type { return c.regularFalseType }

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

// The EmitResolver methods lock the checker's mutex, which the goblox transform already holds through
// Program.GetTypeCheckerForFileExclusive, so it calls these unlocked equivalents instead.

// GetJsxFactoryEntity is EmitResolver.GetJsxFactoryEntity without the checker lock.
func (c *Checker) GetJsxFactoryEntity(location *ast.Node) *ast.Node {
	return c.getJsxFactoryEntity(location)
}

// GetJsxFragmentFactoryEntity is EmitResolver.GetJsxFragmentFactoryEntity without the checker lock.
func (c *Checker) GetJsxFragmentFactoryEntity(location *ast.Node) *ast.Node {
	return c.getJsxFragmentFactoryEntity(location)
}

// IsReferencedAliasDeclaration is EmitResolver.IsReferencedAliasDeclaration without the checker lock.
func (c *Checker) IsReferencedAliasDeclaration(node *ast.Node) bool {
	if !c.canCollectSymbolAliasAccessibilityData || !ast.IsParseTreeNode(node) {
		return true
	}
	if ast.IsAliasSymbolDeclaration(node) {
		if symbol := c.getSymbolOfDeclaration(node); symbol != nil {
			aliasLinks := c.aliasSymbolLinks.Get(symbol)
			if aliasLinks.referenced {
				return true
			}
			target := aliasLinks.aliasTarget
			if target != nil && node.ModifierFlags()&ast.ModifierFlagsExport != 0 &&
				c.getSymbolFlags(target)&ast.SymbolFlagsValue != 0 &&
				(c.compilerOptions.ShouldPreserveConstEnums() || !isConstEnumOrConstEnumOnlyModule(target)) {
				return true
			}
		}
	}
	return false
}

// GetContextualSignature is TypeScript's checker.getContextualSignature.
func (c *Checker) GetContextualSignature(node *ast.Node) *Signature {
	return c.getContextualSignature(node)
}
