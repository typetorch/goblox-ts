package transform

// Port of TSTransformer/util/evaluation/bindings.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

func enclosingFunction(node *ast.Node) *ast.Node {
	return getAncestor(node.Parent, ast.IsFunctionLike)
}

type BindingUsage struct {
	Captured         map[*ast.Symbol]bool
	WrittenByClosure map[*ast.Symbol]bool
}

func getBindingUsage(state *TransformState, sourceFile *ast.SourceFile) *BindingUsage {
	cache := state.MultiTransformState.BindingUsageBySourceFile
	usage := cache[sourceFile]
	if usage == nil {
		captured := map[*ast.Symbol]bool{}
		writtenByClosure := map[*ast.Symbol]bool{}
		var visit func(node *ast.Node) bool
		visit = func(node *ast.Node) bool {
			if ast.IsIdentifier(node) {
				var symbol *ast.Symbol
				if ast.IsShorthandPropertyAssignment(node.Parent) {
					symbol = state.TypeChecker.GetShorthandAssignmentValueSymbol(node.Parent)
				} else {
					symbol = state.TypeChecker.GetSymbolAtLocation(node)
				}
				var declaration *ast.Node
				if symbol != nil {
					declaration = symbol.ValueDeclaration
				}
				if symbol != nil &&
					declaration != nil &&
					ast.GetSourceFileOfNode(declaration) == sourceFile &&
					enclosingFunction(declaration) != enclosingFunction(node) {
					captured[symbol] = true
					if ast.IsAssignmentTarget(node) {
						writtenByClosure[symbol] = true
					}
				}
			}
			node.ForEachChild(visit)
			return false
		}
		visit(sourceFile.AsNode())
		usage = &BindingUsage{Captured: captured, WrittenByClosure: writtenByClosure}
		cache[sourceFile] = usage
	}
	return usage
}

// getBindingKey is TSTransformer/util/evaluation/bindings.ts getBindingKey: unknown callees can only
// rebind a local if some closure has access to it.
func getBindingKey(state *TransformState, symbol *ast.Symbol, reference *ast.Node) *BindingKey {
	cache := state.MultiTransformState.BindingKeys
	if existing := cache[symbol]; existing != nil {
		return existing
	}
	sourceFile := ast.GetSourceFileOfNode(reference)
	usage := getBindingUsage(state, sourceFile)
	declaration := symbol.ValueDeclaration
	external := declaration == nil || ast.GetSourceFileOfNode(declaration) != sourceFile
	key := &BindingKey{
		Captured:         external || usage.Captured[symbol],
		WrittenByClosure: external || usage.WrittenByClosure[symbol],
	}
	cache[symbol] = key
	return key
}
