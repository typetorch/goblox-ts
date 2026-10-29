package transform

// Port of TSTransformer/util/evaluation/plan.ts.

import (
	"strconv"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type EvaluationOperand struct {
	Expression luau.Expression
	Prereqs    *luau.List
}

// planEvaluation is TSTransformer/util/evaluation/plan.ts planEvaluation.
//
// opaque references prevent macros from decomposing or duplicating an operand's
// computation before we decide where its value must be captured
func planEvaluation(prereqs *Prereqs, operands []*EvaluationOperand, expand func(*Prereqs, []luau.Expression) luau.Expression) luau.Expression {
	effects := make([]*EvaluationEffects, len(operands))
	for i, operand := range operands {
		effects[i] = getEffects(operand.Expression)
	}
	slots := map[int]*EvaluationReference{}
	references := make([]luau.Expression, len(operands))
	for index, operand := range operands {
		expression := operand.Expression
		// literal inspection (typeIs, default arguments, index folding) is harmless.
		if luau.IsSimplePrimitive(expression) || luau.IsNone(expression) {
			references[index] = expression
			continue
		}
		reference := luau.TempId("operand")
		copyValueFacts(expression, reference)
		slots[reference.Id] = &EvaluationReference{OperandIndex: index, Expression: expression, Effects: effects[index]}
		references[index] = reference
	}
	expansionPrereqs := NewPrereqs()
	result := expand(expansionPrereqs, references)
	events := getEvaluationEvents(expansionPrereqs.Statements, result, slots)

	uses := make([][]int, len(operands))
	for index, event := range events {
		if event.OperandIndex >= 0 {
			uses[event.OperandIndex] = append(uses[event.OperandIndex], index)
		}
	}
	captures := make([]bool, len(operands))
	hoistedEffects := NO_EFFECTS
	for i := len(operands) - 1; i >= 0; i-- {
		positions := uses[i]
		interveningEffects := hoistedEffects
		// earlier operands are checked on subsequent iterations; captured later
		// operands already contribute their effects at the hoisted position
		last := -1
		if len(positions) > 0 {
			last = positions[len(positions)-1]
		}
		uncertain := false
		for _, position := range positions {
			if events[position].Conditional {
				uncertain = true
				break
			}
		}
		for j, event := range events {
			if event.OperandIndex >= 0 && (event.OperandIndex <= i || captures[event.OperandIndex]) {
				continue
			}
			if j < last || uncertain {
				interveningEffects = joinEffects(interveningEffects, event.Effects)
			}
		}
		if len(positions) == 0 {
			captures[i] = !canDiscard(effects[i])
		} else {
			captures[i] = ((len(positions) > 1 || uncertain) && !canRepeat(effects[i])) ||
				!effectsCommute(effects[i], interveningEffects)
		}
		if captures[i] {
			hoistedEffects = joinEffects(effects[i], hoistedEffects)
		}
		hoistedEffects = joinEffects(getStatementsEffects(operands[i].Prereqs), hoistedEffects)
	}

	replacements := map[int]luau.Expression{}
	for i, operand := range operands {
		prereqs.PushList(operand.Prereqs)
		expression := operand.Expression
		if captures[i] {
			if len(uses[i]) == 0 {
				prereqs.PushList(wrapExpressionStatement(expression))
			} else {
				name := "exp"
				if i != 0 {
					name = "arg" + strconv.Itoa(i-1)
				}
				captured := prereqs.PushToVar(expression, name)
				// working temporaries may change value; only immutable captures inherit these facts
				copyValueFacts(expression, captured)
				expression = captured
			}
		}
		if reference, ok := references[i].(*luau.TemporaryIdentifier); ok && slots[reference.Id] != nil {
			replacements[reference.Id] = expression
		}
	}

	prereqs.PushList(
		luau.MakeList(luau.ListMapToArray(expansionPrereqs.Statements, func(statement luau.Node) luau.Node {
			return substitute(statement, replacements)
		})...),
	)
	return substitute(result, replacements)
}
