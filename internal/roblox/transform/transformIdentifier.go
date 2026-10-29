package transform

// Port of TSTransformer/nodes/expressions/transformIdentifier.ts.

import (
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformIdentifierDefined is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifierDefined.
func transformIdentifierDefined(state *TransformState, node *ast.Node) luau.AnyIdentifier {
	var symbol *ast.Symbol
	if ast.IsShorthandPropertyAssignment(node.Parent) {
		symbol = state.TypeChecker.GetShorthandAssignmentValueSymbol(node.Parent)
	} else {
		symbol = state.TypeChecker.GetSymbolAtLocation(node)
	}
	assert(symbol != nil)

	if replacementId, ok := state.SymbolToIdMap[symbol]; ok {
		identifier := luau.ShallowClone(replacementId)
		markBinding(identifier, symbol, getBindingKey(state, symbol, node))
		return identifier
	}

	identifier := luau.Create(&luau.Identifier{
		Name: node.Text(),
	})
	markBinding(identifier, symbol, getBindingKey(state, symbol, node))
	tryMarkBuiltinLibrary(state, node, identifier)
	if effects, ok := state.MultiTransformState.FunctionEffects[symbol]; ok {
		setCallEffects(identifier, effects)
	}
	return identifier
}

func getAncestorWhichIsChildOf(parent *ast.Node, node *ast.Node) *ast.Node {
	for node.Parent != nil && node.Parent != parent {
		node = node.Parent
	}
	if node.Parent != nil {
		return node
	}
	return nil
}

// for some reason, symbol.valueDeclaration doesn't point to imports?
func getDeclarationFromImport(symbol *ast.Symbol) *ast.Node {
	for _, declaration := range symbol.Declarations {
		importDec := getAncestor(declaration, ast.IsAnyImportSyntax)
		if importDec != nil {
			return declaration
		}
	}
	return nil
}

func checkIdentifierHoist(state *TransformState, node *ast.Node, symbol *ast.Symbol) {
	if _, ok := state.IsHoisted[symbol]; ok {
		return
	}

	declaration := symbol.ValueDeclaration
	if declaration == nil {
		declaration = getDeclarationFromImport(symbol)
	}

	// parameters cannot be hoisted
	if declaration == nil || getAncestor(declaration, ast.IsParameterDeclaration) != nil || ast.IsShorthandPropertyAssignment(declaration) {
		return
	}

	// class expressions can self refer
	if ast.IsClassLike(declaration) && isAncestorOf(declaration, node) {
		return
	}

	declarationStatement := getAncestor(declaration, ast.IsStatement)
	if declarationStatement == nil ||
		ast.IsForStatement(declarationStatement) ||
		ast.IsForOfStatement(declarationStatement) ||
		ast.IsTryStatement(declarationStatement) {
		return
	}

	parent := declarationStatement.Parent
	if parent == nil || !isBlockLike(parent) {
		return
	}

	sibling := getAncestorWhichIsChildOf(parent, node)
	if sibling == nil || !ast.IsStatement(sibling) {
		return
	}

	declarationIdx := slices.Index(parent.Statements(), declarationStatement)
	siblingIdx := slices.Index(parent.Statements(), sibling)

	if siblingIdx > declarationIdx {
		return
	}

	if siblingIdx == declarationIdx {
		// non-async function declarations, class declarations, and variable statements can self refer
		if (ast.IsFunctionDeclaration(declarationStatement) &&
			!ast.HasSyntacticModifier(declarationStatement, ast.ModifierFlagsAsync)) ||
			ast.IsClassDeclaration(declarationStatement) ||
			(ast.IsVariableStatement(declarationStatement) &&
				getAncestor(node, func(node *ast.Node) bool {
					return ast.IsStatement(node) || ast.IsFunctionLikeDeclaration(node)
				}) == declarationStatement) {
			return
		}
	}

	state.HoistsByStatement[sibling] = append(state.HoistsByStatement[sibling], node)
	state.IsHoisted[symbol] = true
}

// transformIdentifier is TSTransformer/nodes/expressions/transformIdentifier.ts transformIdentifier.
func transformIdentifier(state *TransformState, node *ast.Node) luau.Expression {
	// synthetic nodes don't have parents or symbols, so skip all the symbol-related logic
	// JSX EntityName functions like `getJsxFactoryEntity()` will return synthetic nodes
	// and transformEntityName will eventually end up here
	if node.Parent == nil || ast.PositionIsSynthesized(node.Pos()) {
		return luau.Create(&luau.Identifier{Name: node.Text()})
	}

	var symbol *ast.Symbol
	if ast.IsShorthandPropertyAssignment(node.Parent) {
		symbol = state.TypeChecker.GetShorthandAssignmentValueSymbol(node.Parent)
	} else {
		symbol = state.TypeChecker.GetSymbolAtLocation(node)
	}
	assert(symbol != nil)

	macroManager := state.Services.MacroManager
	if state.TypeChecker.IsUndefinedSymbol(symbol) {
		return luau.Nil()
	} else if state.TypeChecker.IsArgumentsSymbol(symbol) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoArguments.Create(node))
	} else if symbol == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.GlobalThis) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoGlobalThis.Create(node))
	}

	if macro := macroManager.GetIdentifierMacro(symbol); macro != nil {
		return macro(state, node)
	}

	constructSymbol := getFirstConstructSymbol(state, node)
	if constructSymbol != nil {
		if constructorMacro := macroManager.GetConstructorMacro(constructSymbol); constructorMacro != nil {
			isClassExtendsNode := false
			if node.Parent.Parent != nil && node.Parent.Parent.Parent != nil && ast.IsClassLike(node.Parent.Parent.Parent) {
				extendsNode := getExtendsNode(node.Parent.Parent.Parent)
				isClassExtendsNode = extendsNode != nil && extendsNode.Expression() == node
			}
			if isClassExtendsNode {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMacroExtends.Create(node))
			} else {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoConstructorMacroWithoutNew.Create(node))
			}
		}
	}

	parent := skipUpwards(node).Parent
	if (!ast.IsCallExpression(parent) || skipDownwards(parent.Expression()) != node) &&
		macroManager.GetCallMacro(symbol) != nil {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoIndexWithoutCall.Create(node))
		return luau.NewNone()
	}

	// exit here for export let so we don't check hoist later
	if symbol.ValueDeclaration != nil &&
		ast.GetSourceFileOfNode(symbol.ValueDeclaration) == ast.GetSourceFileOfNode(node) &&
		getAncestor(symbol.ValueDeclaration, func(node *ast.Node) bool {
			return ast.IsModuleDeclaration(node) && !isNamespace(node)
		}) == nil {
		exportAccess := state.GetModuleIdPropertyAccess(symbol)
		if exportAccess != nil && isSymbolMutable(state, symbol) {
			return exportAccess
		}
	}

	checkIdentifierHoist(state, node, symbol)

	return transformIdentifierDefined(state, node)
}
