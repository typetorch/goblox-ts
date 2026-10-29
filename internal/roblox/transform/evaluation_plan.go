package transform

// Port of TSTransformer/util/evaluation/plan.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type EvaluationOperand struct {
	Expression luau.Expression
	Prereqs    *luau.List
}

// planEvaluation is TSTransformer/util/evaluation/plan.ts planEvaluation.
func planEvaluation(prereqs *Prereqs, operands []*EvaluationOperand, expand func(*Prereqs, []luau.Expression) luau.Expression) luau.Expression {
	panic(todo("util/evaluation/plan.ts", "planEvaluation"))
}
