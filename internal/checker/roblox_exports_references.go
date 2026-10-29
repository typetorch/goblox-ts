package checker

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/astnav"
	"github.com/microsoft/typescript-go/internal/scanner"
)

// Exports for goblox-ts (internal/roblox), like roblox_exports.go: a separate file only so that
// concurrent goblox agents do not conflict on its import block.

// EachSymbolReferenceInFile is TypeScript's FindAllReferences.Core.eachSymbolReferenceInFile. It calls
// cb for every reference to the symbol of definition inside searchContainer (the whole file when nil),
// except definition itself, and returns true as soon as cb returns true.
func (c *Checker) EachSymbolReferenceInFile(definition *ast.Node, sourceFile *ast.SourceFile, cb func(token *ast.Node) bool, searchContainer *ast.Node) bool {
	var symbol *ast.Symbol
	if definition.Parent != nil && definition.Parent.Parent != nil && ast.IsParameterPropertyDeclaration(definition.Parent, definition.Parent.Parent) {
		symbol, _ = c.GetSymbolsOfParameterPropertyDeclaration(definition.Parent, definition.Text())
	} else {
		symbol = c.GetSymbolAtLocation(definition)
	}
	if symbol == nil {
		return false
	}
	if searchContainer == nil {
		searchContainer = sourceFile.AsNode()
	}
	for _, token := range robloxPossibleSymbolReferenceNodes(sourceFile, symbol.Name, searchContainer) {
		if !ast.IsIdentifier(token) || token == definition || token.Text() != definition.Text() {
			continue
		}
		referenceSymbol := c.GetSymbolAtLocation(token)
		if referenceSymbol == symbol ||
			c.GetShorthandAssignmentValueSymbol(token.Parent) == symbol ||
			(ast.IsExportSpecifier(token.Parent) &&
				c.getLocalSymbolForExportSpecifier(token.AsIdentifier(), referenceSymbol, token.Parent.AsExportSpecifier()) == symbol) {
			if cb(token) {
				return true
			}
		}
	}
	return false
}

// IsSymbolReferencedInFileWithin is TypeScript's FindAllReferences.Core.isSymbolReferencedInFile with
// its searchContainer argument (nil searches the whole file).
func (c *Checker) IsSymbolReferencedInFileWithin(definition *ast.Node, sourceFile *ast.SourceFile, searchContainer *ast.Node) bool {
	return c.EachSymbolReferenceInFile(definition, sourceFile, func(*ast.Node) bool { return true }, searchContainer)
}

// robloxPossibleSymbolReferenceNodes is getPossibleSymbolReferenceNodes with absolute positions for a
// container that is not the source file (getPossibleSymbolReferencePositions only handles the file).
func robloxPossibleSymbolReferenceNodes(sourceFile *ast.SourceFile, symbolName string, container *ast.Node) []*ast.Node {
	var result []*ast.Node
	if symbolName == "" {
		return result
	}
	text := sourceFile.Text()
	sourceLength := len(text)
	symbolNameLength := len(symbolName)
	endPos := container.End()
	position := container.Pos()
	for position <= sourceLength {
		found := strings.Index(text[position:], symbolName)
		if found < 0 {
			break
		}
		position += found
		if position >= endPos {
			break
		}
		endPosition := position + symbolNameLength
		if (position == 0 || !scanner.IsIdentifierPart(rune(text[position-1]))) &&
			(endPosition == sourceLength || !scanner.IsIdentifierPart(rune(text[endPosition]))) {
			if referenceLocation := astnav.GetTouchingPropertyName(sourceFile, position); referenceLocation != sourceFile.AsNode() {
				result = append(result, referenceLocation)
			}
		}
		position += symbolNameLength + 1
	}
	return result
}
