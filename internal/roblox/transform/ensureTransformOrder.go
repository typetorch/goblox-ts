package transform

// Port of TSTransformer/util/ensureTransformOrder.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ensureTransformOrder is TSTransformer/util/ensureTransformOrder.ts ensureTransformOrder.
// A nil transformer is the TS default, transformExpression.
func ensureTransformOrder(state *TransformState, prereqs *Prereqs, nodes []*ast.Node, transformer func(*TransformState, *Prereqs, *ast.Node) luau.Expression) []luau.Expression {
	if transformer == nil {
		transformer = transformExpression
	}
	type expressionInfo struct {
		expression       luau.Expression
		prereqStatements *luau.List
	}
	expressionInfoList := make([]expressionInfo, len(nodes))
	for i, node := range nodes {
		expressionPrereqs := NewPrereqs()
		expression := transformer(state, expressionPrereqs, node)
		expressionInfoList[i] = expressionInfo{expression, expressionPrereqs.Statements}
	}
	captures := make([]bool, len(nodes))
	// each capture becomes a prerequisite that earlier expressions must also cross
	suffix := NO_EFFECTS
	for i := len(expressionInfoList) - 1; i >= 0; i-- {
		info := expressionInfoList[i]
		// everything commutes with no effects, and this is every call's last argument
		if suffix != NO_EFFECTS {
			effects := getEffects(info.expression)
			captures[i] = !effectsCommute(effects, suffix)
			if captures[i] {
				suffix = joinEffects(effects, suffix)
			}
		}
		suffix = joinEffects(getStatementsEffects(info.prereqStatements), suffix)
	}
	result := make([]luau.Expression, 0, len(nodes))
	for i, info := range expressionInfoList {
		prereqs.PushList(info.prereqStatements)

		if captures[i] {
			result = append(result, prereqs.PushToVar(info.expression, "exp"))
		} else {
			result = append(result, info.expression)
		}
	}
	return result
}
