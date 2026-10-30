package transform

// Port of TSTransformer/nodes/transformStatementList.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/ls/lsutil"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ExportInfo is the exportInfo parameter of transformStatementList.
type ExportInfo struct {
	Id      luau.AnyIdentifier
	Mapping map[*ast.Node][]string
}

// isNodeDescendantOf is ts.isNodeDescendantOf.
func isNodeDescendantOf(node *ast.Node, ancestor *ast.Node) bool {
	for node != nil {
		if node == ancestor {
			return true
		}
		node = node.Parent
	}
	return false
}

func getLastToken(state *TransformState, parent *ast.Node, statements []*ast.Node) *ast.Node {
	if len(statements) > 0 {
		lastStatement := statements[len(statements)-1]
		lastToken := lsutil.GetLastToken(lastStatement.Parent, state.SourceFile)
		if lastToken != nil && !isNodeDescendantOf(lastToken, lastStatement) {
			return lastToken
		}
	} else if parent != nil {
		// if statements is empty, there still might be an `EOF` or `}` token to look for
		return lsutil.GetLastToken(parent, state.SourceFile)
	}
	return nil
}

// transformStatementList converts a ts.Statement array into a luau.list<...> tree.
func transformStatementList(state *TransformState, parent *ast.Node, statements []*ast.Node, exportInfo *ExportInfo) *luau.List {
	// make a new Luau tree
	result := luau.MakeList()

	// iterate through each statement in the `statements` array
	for _, statement := range statements {
		transformedStatements := transformStatement(state, statement)

		// iterate through each of the leading comments of the statement
		if !state.CompilerOptions.RemoveComments.IsTrue() {
			result.PushList(state.GetLeadingComments(statement))
		}

		// check statement for hoisting
		// hoisting is the use of a variable before it was declared
		hoistDeclaration := createHoistDeclaration(state, statement)
		if hoistDeclaration != nil {
			result.Push(hoistDeclaration)
		}

		result.PushList(transformedStatements)

		if transformedStatements.Tail != nil && luau.IsFinalStatement(transformedStatements.Tail.Value) {
			break
		}

		// namespace export handling
		if exportInfo != nil {
			containerId := exportInfo.Id
			if exportMapping, ok := exportInfo.Mapping[statement]; ok {
				for _, exportName := range exportMapping {
					result.Push(luau.Create(&luau.Assignment{
						Left:     luau.Property(containerId, exportName),
						Operator: "=",
						Right:    luau.Id(exportName),
					}))
				}
			}
		}
	}

	if !state.CompilerOptions.RemoveComments.IsTrue() {
		lastToken := getLastToken(state, parent, statements)
		// see ProjectData.TransformersRan: TypeScript's printer drops the comments before the end of the file
		if lastToken != nil && !(state.transformersRan() && lastToken.Kind == ast.KindEndOfFile) {
			result.PushList(state.GetLeadingComments(lastToken))
		}
	}

	return result
}
