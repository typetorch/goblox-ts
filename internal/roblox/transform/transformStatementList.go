package transform

// Port of TSTransformer/nodes/transformStatementList.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ExportInfo is the exportInfo parameter of transformStatementList.
type ExportInfo struct {
	Id      luau.AnyIdentifier
	Mapping map[*ast.Node][]string
}

// transformStatementList is TSTransformer/nodes/transformStatementList.ts transformStatementList.
func transformStatementList(state *TransformState, parent *ast.Node, statements []*ast.Node, exportInfo *ExportInfo) *luau.List {
	panic(todo("nodes/transformStatementList.ts", "transformStatementList"))
}
