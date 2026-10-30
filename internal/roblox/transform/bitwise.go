package transform

// Port of TSTransformer/util/bitwise.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// bitwiseOperatorMap is OPERATOR_MAP of bitwise.ts.
var bitwiseOperatorMap = map[ast.Kind]string{
	// bitwise
	ast.KindLessThanLessThanToken:                  "lshift",
	ast.KindGreaterThanGreaterThanGreaterThanToken: "rshift",
	ast.KindGreaterThanGreaterThanToken:            "arshift",

	// bitwise compound assignment
	ast.KindAmpersandEqualsToken:                         "band",
	ast.KindBarEqualsToken:                               "bor",
	ast.KindCaretEqualsToken:                             "bxor",
	ast.KindLessThanLessThanEqualsToken:                  "lshift",
	ast.KindGreaterThanGreaterThanGreaterThanEqualsToken: "rshift",
	ast.KindGreaterThanGreaterThanEqualsToken:            "arshift",
}

// bitwiseLogicalOperatorMap is LOGICAL_OPERATOR_MAP of bitwise.ts.
var bitwiseLogicalOperatorMap = map[ast.Kind]string{
	ast.KindAmpersandToken: "band",
	ast.KindBarToken:       "bor",
	ast.KindCaretToken:     "bxor",
}

func flattenBitwiseSegmentInto(expressionList *[]*ast.Node, operatorKind ast.Kind, node *ast.Node) {
	if ast.IsBinaryExpression(node) {
		binary := node.AsBinaryExpression()
		if operatorKind == binary.OperatorToken.Kind {
			flattenBitwiseSegmentInto(expressionList, operatorKind, binary.Left)

			*expressionList = append(*expressionList, skipDownwards(binary.Right))
		} else {
			*expressionList = append(*expressionList, node)
		}

		return
	}

	*expressionList = append(*expressionList, skipDownwards(node))
}

func isBitwiseLogicalOperator(operatorKind ast.Kind) bool {
	_, ok := bitwiseLogicalOperatorMap[operatorKind]
	return ok
}

// isBitwiseOperator is TSTransformer/util/bitwise.ts isBitwiseOperator.
func isBitwiseOperator(operatorKind ast.Kind) bool {
	_, ok := bitwiseOperatorMap[operatorKind]
	return ok || isBitwiseLogicalOperator(operatorKind)
}

// createBitwiseCall is TSTransformer/util/bitwise.ts createBitwiseCall.
func createBitwiseCall(operatorKind ast.Kind, expressions []luau.Expression) luau.Expression {
	name, ok := bitwiseOperatorMap[operatorKind]
	if !ok {
		name, ok = bitwiseLogicalOperatorMap[operatorKind]
	}
	assert(ok, "createBitwiseFromOperator unknown operator: "+getKindName(operatorKind))

	return luau.Call(luau.Property(luau.Globals.Bit32, name), expressions...)
}

// createBitwiseFromOperator is TSTransformer/util/bitwise.ts createBitwiseFromOperator.
func createBitwiseFromOperator(state *TransformState, prereqs *Prereqs, operatorKind ast.Kind, node *ast.Node) luau.Expression {
	binary := node.AsBinaryExpression()
	var flattenedExpressions []*ast.Node
	if isBitwiseLogicalOperator(operatorKind) {
		flattenBitwiseSegmentInto(&flattenedExpressions, operatorKind, binary.Left)

		flattenedExpressions = append(flattenedExpressions, binary.Right)
	} else {
		flattenedExpressions = append(flattenedExpressions, binary.Left, binary.Right)
	}

	return createBitwiseCall(operatorKind, ensureTransformOrder(state, prereqs, flattenedExpressions, transformExpression))
}
