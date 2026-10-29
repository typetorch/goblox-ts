package transform

// Port of TSTransformer/util/getExtendsNode.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
)

// getExtendsNode is TSTransformer/util/getExtendsNode.ts getExtendsNode.
func getExtendsNode(node *ast.Node) *ast.Node {
	heritageClauses := node.ClassLikeData().HeritageClauses
	if heritageClauses == nil {
		return nil
	}
	for _, clause := range heritageClauses.Nodes {
		if clause.AsHeritageClause().Token == ast.KindExtendsKeyword {
			return clause.AsHeritageClause().Types.Nodes[0]
		}
	}
	return nil
}
