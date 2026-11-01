package transform

// Port of TSTransformer/nodes/transformSourceFile.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
)

func getExportDeclarations(exportSymbol *ast.Symbol) []*ast.Node {
	declarations := exportSymbol.Declarations
	assert(len(declarations) > 0)
	return declarations
}

type exportPair struct {
	name luau.Expression
	id   luau.AnyIdentifier
}

func getExportPair(state *TransformState, exportSymbol *ast.Symbol) exportPair {
	declaration := getExportDeclarations(exportSymbol)[0]
	if ast.IsExportSpecifier(declaration) {
		exportSpecifier := declaration.AsExportSpecifier()
		exportName := exportSpecifier.PropertyName
		if exportName == nil {
			exportName = exportSpecifier.Name()
		}
		// exportName is only a StringLiteral for re-exports, which are filtered out in handleExports
		assert(ast.IsIdentifier(exportName))
		namePrereqs := NewPrereqs()
		name := transformPropertyName(state, namePrereqs, exportSpecifier.Name())
		assert(namePrereqs.Statements.IsEmpty())
		return exportPair{name, transformIdentifierDefined(state, exportName)}
	}

	name := exportSymbol.Name
	if exportSymbol.Name == "default" &&
		(ast.IsFunctionDeclaration(declaration) || ast.IsClassDeclaration(declaration)) &&
		declaration.Name() != nil {
		name = declaration.Name().Text()
	}

	return exportPair{luau.String(exportSymbol.Name), luau.Id(name)}
}

func isExportSymbolFromExportFrom(exportSymbol *ast.Symbol) bool {
	for _, exportSpecifier := range getExportDeclarations(exportSymbol) {
		if ast.IsExportSpecifier(exportSpecifier) {
			exportDec := exportSpecifier.Parent.Parent
			if ast.IsExportDeclaration(exportDec) && exportDec.AsExportDeclaration().ModuleSpecifier != nil {
				return true
			}
		}
	}
	return false
}

func getIgnoredExportSymbols(state *TransformState, sourceFile *ast.SourceFile) map[*ast.Symbol]bool {
	ignoredSymbols := map[*ast.Symbol]bool{}
	for _, statement := range sourceFile.Statements.Nodes {
		if ast.IsExportDeclaration(statement) && statement.AsExportDeclaration().ModuleSpecifier != nil {
			exportDec := statement.AsExportDeclaration()
			if exportDec.ExportClause == nil {
				// export * from "./module";
				moduleSymbol := getOriginalSymbolOfNode(state.TypeChecker, exportDec.ModuleSpecifier)
				if moduleSymbol != nil {
					for _, v := range state.GetModuleExports(moduleSymbol) {
						ignoredSymbols[v] = true
					}
				}
			} else if ast.IsNamespaceExport(exportDec.ExportClause) {
				// export * as id from "./module";
				idSymbol := state.TypeChecker.GetSymbolAtLocation(exportDec.ExportClause.Name())
				assert(idSymbol != nil)
				ignoredSymbols[idSymbol] = true
			}
		}
	}
	return ignoredSymbols
}

// isExportSymbolOnlyFromDeclare is used to ignore exports in the form of `export declare const x: T;`
// however, this should still allow exports which are declare + export separately, i.e.
//
//	declare const x: number;
//	export { x };
//
// this mimics TypeScript behavior
func isExportSymbolOnlyFromDeclare(exportSymbol *ast.Symbol) bool {
	for _, declaration := range getExportDeclarations(exportSymbol) {
		statement := getAncestor(declaration, ast.IsStatement)
		assert(statement != nil && ast.CanHaveModifiers(statement))
		hasDeclare := false
		if modifiers := statement.Modifiers(); modifiers != nil {
			for _, v := range modifiers.Nodes {
				if v.Kind == ast.KindDeclareKeyword {
					hasDeclare = true
					break
				}
			}
		}
		if !hasDeclare {
			return false
		}
	}
	return true
}

// handleExports adds export information to the end of the tree.
func handleExports(state *TransformState, sourceFile *ast.SourceFile, symbol *ast.Symbol, statements *luau.List) {
	ignoredExportSymbols := getIgnoredExportSymbols(state, sourceFile)

	mustPushExports := state.HasExportFrom
	var exportPairs []exportPair
	// even an erased export = replaces the module's named exports
	hasExportEquals := false
	for _, statement := range sourceFile.Statements.Nodes {
		if ast.IsExportAssignment(statement) && statement.AsExportAssignment().IsExportEquals {
			hasExportEquals = true
			break
		}
	}
	if !hasExportEquals {
		for _, exportSymbol := range state.GetModuleExports(symbol) {
			if ignoredExportSymbols[exportSymbol] {
				continue
			}

			// export { default as x } from "./module";
			if isExportSymbolFromExportFrom(exportSymbol) {
				continue
			}

			originalSymbol := SkipAliases(state.TypeChecker, exportSymbol)

			// only export values
			if !isSymbolOfValue(originalSymbol) {
				continue
			}

			// handle this in transformIdentifier
			if isSymbolMutable(state, originalSymbol) {
				mustPushExports = true
				continue
			}

			// ignore exports in the form of `export declare const x: T;`
			if isExportSymbolOnlyFromDeclare(exportSymbol) {
				continue
			}

			exportPairs = append(exportPairs, getExportPair(state, exportSymbol))
		}
	}

	if state.HasExportEquals {
		// local exports variable is created in transformExportAssignment
		finalStatement := sourceFile.Statements.Nodes[len(sourceFile.Statements.Nodes)-1]
		if !(ast.IsExportAssignment(finalStatement) && finalStatement.AsExportAssignment().IsExportEquals) {
			statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Globals.Exports}))
		}
	} else if mustPushExports {
		// if there's an export let/from, we need to put `local exports = {}` at the top of the file
		statements.Unshift(luau.Create(&luau.VariableDeclaration{
			Left:  luau.Globals.Exports,
			Right: luau.NewMap(),
		}))
		for _, pair := range exportPairs {
			statements.Push(luau.Create(&luau.Assignment{
				Left: luau.Create(&luau.ComputedIndexExpression{
					Expression: luau.Globals.Exports,
					Index:      pair.name,
				}),
				Operator: "=",
				Right:    pair.id,
			}))
		}
		statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Globals.Exports}))
	} else if len(exportPairs) > 0 {
		// only regular exports, we can do this as just returning an object at the bottom of the file
		fields := luau.MakeList()
		for _, pair := range exportPairs {
			fields.Push(luau.Create(&luau.MapField{Index: pair.name, Value: pair.id}))
		}
		statements.Push(luau.Create(&luau.ReturnStatement{
			Expression: luau.Create(&luau.Map{Fields: fields}),
		}))
	}
}

func getLastNonCommentStatement(listNode *luau.ListNode) *luau.ListNode {
	for listNode != nil && luau.IsComment(listNode.Value) {
		listNode = listNode.Prev
	}
	return listNode
}

// transformSourceFile creates and returns a luau.list<luau.Statement> (Luau AST).
func transformSourceFile(state *TransformState, node *ast.SourceFile) *luau.List {
	symbol := state.TypeChecker.GetSymbolAtLocation(node.AsNode())
	assert(symbol != nil)
	state.SetModuleIdBySymbol(symbol, luau.Globals.Exports)

	// transform the `ts.Statements` of the source file into a `list.list<...>`
	statements := transformStatementList(state, node.AsNode(), node.Statements.Nodes, nil)

	handleExports(state, node, symbol, statements)

	// moduleScripts must `return nil` if they do not export any values
	lastStatement := getLastNonCommentStatement(statements.Tail)
	if lastStatement == nil || !luau.IsReturnStatement(lastStatement.Value) {
		outputPath := state.PathTranslator.GetOutputPath(node.FileName())
		if state.RojoResolver.GetRbxTypeFromFilePath(outputPath) == rojo.RbxTypeModuleScript {
			statements.Push(luau.Create(&luau.ReturnStatement{Expression: luau.Nil()}))
		}
	}

	headerStatements := luau.MakeList()

	// add build information to the tree
	if !state.Data.ProjectOptions.NoCompilerHeader {
		headerStatements.Push(luau.NewComment(" goblox, a roblox-ts fork. goblox v" + GOBLOX_VERSION + ", roblox-ts v" + COMPILER_VERSION))
	}

	// add the Runtime library to the tree if it is used
	if state.UsesRuntimeLib {
		headerStatements.Push(state.CreateRuntimeLibImport(node))
	}

	// extract Luau directive comments like --!strict so we can put them before headerStatements
	directiveComments := luau.MakeList()
	for statements.Head != nil && luau.IsComment(statements.Head.Value) &&
		len(statements.Head.Value.(*luau.Comment).Text) > 0 && statements.Head.Value.(*luau.Comment).Text[0] == '!' {
		directiveComments.Push(statements.Shift())
	}

	statements.UnshiftList(headerStatements)
	statements.UnshiftList(directiveComments)

	return statements
}
