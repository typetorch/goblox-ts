package transform

// Port of TSTransformer/nodes/expressions/transformThisExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformThisExpression is TSTransformer/nodes/expressions/transformThisExpression.ts transformThisExpression.
func transformThisExpression(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	symbol := state.TypeChecker.GetSymbolAtLocation(node)
	if symbol == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.GlobalThis) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoGlobalThis.Create(node))
	}

	if symbol != nil {
		container := ast.GetThisContainer(node, false, false)

		// ts.hasStaticModifier doesn't work on static blocks
		isStatic := ast.HasStaticModifier(container) || ast.IsClassStaticBlockDeclaration(container)

		// MethodDeclaration creates it's own implicit this
		if isStatic && !ast.IsMethodDeclaration(container) && ast.IsClassLike(container.Parent) {
			identifier := state.ClassIdentifierMap[container.Parent]
			assert(identifier != nil)
			return identifier
		}
	}

	return luau.Globals.Self
}
