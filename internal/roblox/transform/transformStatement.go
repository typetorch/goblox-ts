package transform

// Port of TSTransformer/nodes/statements/transformStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// StatementTransformer is the dispatch signature of TRANSFORMER_BY_KIND.
type StatementTransformer = func(state *TransformState, node *ast.Node) *luau.List

func statementNoEmit(state *TransformState, node *ast.Node) *luau.List {
	return luau.MakeList()
}

func statementDiagnostic(factory *diagnostics.DiagnosticFactory) StatementTransformer {
	return func(state *TransformState, node *ast.Node) *luau.List {
		DiagnosticService.AddDiagnostic(factory.Create(node))
		return luau.MakeList()
	}
}

var statementTransformerByKind map[ast.Kind]StatementTransformer

func init() {
	statementTransformerByKind = map[ast.Kind]StatementTransformer{
		// no emit
		ast.KindInterfaceDeclaration: statementNoEmit,
		ast.KindTypeAliasDeclaration: statementNoEmit,
		ast.KindEmptyStatement:       statementNoEmit,

		// banned statements
		ast.KindForInStatement:    statementDiagnostic(diagnostics.Errors.NoForInStatement),
		ast.KindLabeledStatement:  statementDiagnostic(diagnostics.Errors.NoLabeledStatement),
		ast.KindDebuggerStatement: statementDiagnostic(diagnostics.Errors.NoDebuggerStatement),

		// regular transforms
		ast.KindBlock:                   transformBlock,
		ast.KindBreakStatement:          transformBreakStatement,
		ast.KindClassDeclaration:        transformClassDeclaration,
		ast.KindContinueStatement:       transformContinueStatement,
		ast.KindDoStatement:             transformDoStatement,
		ast.KindEnumDeclaration:         transformEnumDeclaration,
		ast.KindExportAssignment:        transformExportAssignment,
		ast.KindExportDeclaration:       transformExportDeclaration,
		ast.KindExpressionStatement:     transformExpressionStatement,
		ast.KindForOfStatement:          transformForOfStatement,
		ast.KindForStatement:            transformForStatement,
		ast.KindFunctionDeclaration:     transformFunctionDeclaration,
		ast.KindIfStatement:             transformIfStatement,
		ast.KindImportDeclaration:       transformImportDeclaration,
		ast.KindImportEqualsDeclaration: transformImportEqualsDeclaration,
		ast.KindModuleDeclaration:       transformModuleDeclaration,
		ast.KindReturnStatement:         transformReturnStatement,
		ast.KindSwitchStatement:         transformSwitchStatement,
		ast.KindThrowStatement:          transformThrowStatement,
		ast.KindTryStatement:            transformTryStatement,
		ast.KindVariableStatement:       transformVariableStatement,
		ast.KindWhileStatement:          transformWhileStatement,
	}
}

// transformStatement transforms a singular `ts.Statement` in a `luau.list<...>`.
func transformStatement(state *TransformState, node *ast.Node) *luau.List {
	// if any modifiers of the node include the `declare` keyword we do not transform
	// `declare` tells us that the identifier of the node is defined somewhere else and we should trust it
	if hasDeclareModifier(node) {
		return luau.MakeList()
	}

	transformer := statementTransformerByKind[node.Kind]
	if transformer != nil {
		return transformer(state, node)
	}
	assert(false, "Unknown statement: "+getKindName(node.Kind))
	return nil
}
