package transform

// Port of TSTransformer/nodes/transformStatementList.ts.

import (
	"strings"

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
		switch lastStatement.Parent.Kind {
		case ast.KindSourceFile:
			return lastStatement.Parent.AsSourceFile().EndOfFileToken
		case ast.KindCaseClause, ast.KindDefaultClause:
			// a clause ends with its last statement
			return nil
		}
		lastToken := lsutil.GetLastToken(lastStatement.Parent, state.SourceFile)
		if lastToken != nil && !isNodeDescendantOf(lastToken, lastStatement) {
			return lastToken
		}
	} else if parent != nil {
		switch parent.Kind {
		case ast.KindSourceFile:
			return parent.AsSourceFile().EndOfFileToken
		case ast.KindBlock, ast.KindModuleBlock:
			// nothing to find in an empty block without comments
			if !strings.Contains(state.sourceFileText[parent.Pos():parent.End()], "/") {
				return nil
			}
		}
		// if statements is empty, there still might be an `EOF` or `}` token to look for
		return lsutil.GetLastToken(parent, state.SourceFile)
	}
	return nil
}

// getLastTokenPos is the Pos() of getLastToken, when that doesn't take scanning the parent again. The
// closing brace of a block starts where its last statement ends. That may not be the last of statements:
// a constructor renders the statements around its super call separately.
func getLastTokenPos(statements []*ast.Node) (int, bool) {
	if len(statements) > 0 {
		if parent := statements[0].Parent; parent.Kind == ast.KindBlock || parent.Kind == ast.KindModuleBlock {
			allStatements := parent.Statements()
			return allStatements[len(allStatements)-1].End(), true
		}
	}
	return 0, false
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
		if pos, ok := getLastTokenPos(statements); ok {
			result.PushList(state.getLeadingCommentsAt(pos))
			return result
		}
		lastToken := getLastToken(state, parent, statements)
		// see ProjectData.TransformersRan: TypeScript's printer drops the comments before the end of the file
		if lastToken != nil && !(state.transformersRan() && lastToken.Kind == ast.KindEndOfFile) {
			result.PushList(state.GetLeadingComments(lastToken))
		}
	}

	return result
}
