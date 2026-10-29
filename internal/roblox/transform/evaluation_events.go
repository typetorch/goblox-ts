package transform

// Port of TSTransformer/util/evaluation/events.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type EvaluationReference struct {
	OperandIndex int
	Expression   luau.Expression
	Effects      *EvaluationEffects
}

type EvaluationEvent struct {
	Effects      *EvaluationEffects
	OperandIndex int // -1 when absent
	Conditional  bool
}

// getEvaluationEvents is TSTransformer/util/evaluation/events.ts getEvaluationEvents.
func getEvaluationEvents(statements *luau.List, result luau.Expression, references map[int]*EvaluationReference) []*EvaluationEvent {
	panic(todo("util/evaluation/events.ts", "getEvaluationEvents"))
}
