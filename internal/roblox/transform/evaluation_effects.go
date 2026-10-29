package transform

// Port of TSTransformer/util/evaluation/effects.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// EvaluationEffects are the effects of executing emitted code, not of merely constructing its AST.
type EvaluationEffects struct {
	// bindings whose current values may be observed
	Reads map[Binding]bool
	// bindings that may be reassigned, not mutations to their referenced objects
	Writes map[Binding]bool
	// may read mutable shared state, such as table entries or Instance properties
	ReadsHeap bool
	// may mutate shared state without reassigning a binding
	WritesHeap bool
	// may error or fail to terminate, preventing later evaluations
	Throws bool
	// may create a fresh identity, such as a table or closure, that cannot be duplicated
	Allocates bool
}

// getFunctionEffects is TSTransformer/util/evaluation/effects.ts getFunctionEffects.
func getFunctionEffects(parameters *luau.List, statements *luau.List) *EvaluationEffects {
	panic(todo("util/evaluation/effects.ts", "getFunctionEffects"))
}

// joinEffects is TSTransformer/util/evaluation/effects.ts joinEffects.
func joinEffects(effects ...*EvaluationEffects) *EvaluationEffects {
	panic(todo("util/evaluation/effects.ts", "joinEffects"))
}

// effectsCommute is TSTransformer/util/evaluation/effects.ts effectsCommute.
func effectsCommute(a *EvaluationEffects, b *EvaluationEffects) bool {
	panic(todo("util/evaluation/effects.ts", "effectsCommute"))
}

// canDiscard is TSTransformer/util/evaluation/effects.ts canDiscard.
func canDiscard(effects *EvaluationEffects) bool {
	panic(todo("util/evaluation/effects.ts", "canDiscard"))
}

// canRepeat is TSTransformer/util/evaluation/effects.ts canRepeat.
func canRepeat(effects *EvaluationEffects) bool {
	panic(todo("util/evaluation/effects.ts", "canRepeat"))
}

// getIntrinsicEffects is TSTransformer/util/evaluation/effects.ts getIntrinsicEffects.
func getIntrinsicEffects(node luau.Node) *EvaluationEffects {
	panic(todo("util/evaluation/effects.ts", "getIntrinsicEffects"))
}

// forEachChild is TSTransformer/util/evaluation/effects.ts forEachChild.
func forEachChild(node luau.Node, callback func(luau.Node)) {
	panic(todo("util/evaluation/effects.ts", "forEachChild"))
}

// getEffects is TSTransformer/util/evaluation/effects.ts getEffects.
func getEffects(node luau.Node) *EvaluationEffects {
	panic(todo("util/evaluation/effects.ts", "getEffects"))
}

// isLateRead is TSTransformer/util/evaluation/effects.ts isLateRead.
func isLateRead(expression luau.Expression) bool {
	panic(todo("util/evaluation/effects.ts", "isLateRead"))
}
