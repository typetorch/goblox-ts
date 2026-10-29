package transform

// Port of TSTransformer/util/evaluation/facts.ts. TODO(port): stub, the signatures are fixed.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

type BindingKey struct {
	Captured         bool
	WrittenByClosure bool
}

// Binding is a *BindingKey, string, number or sentinel value.
type Binding = any

// markConstantReference is TSTransformer/util/evaluation/facts.ts markConstantReference.
func markConstantReference(expression luau.Expression) {
	panic(todo("util/evaluation/facts.ts", "markConstantReference"))
}

// markStableLookup is TSTransformer/util/evaluation/facts.ts markStableLookup.
func markStableLookup(expression luau.Expression) {
	panic(todo("util/evaluation/facts.ts", "markStableLookup"))
}

// markPrimitiveValue is TSTransformer/util/evaluation/facts.ts markPrimitiveValue.
func markPrimitiveValue(expression luau.Expression) {
	panic(todo("util/evaluation/facts.ts", "markPrimitiveValue"))
}

// copyValueFacts is TSTransformer/util/evaluation/facts.ts copyValueFacts.
func copyValueFacts(from luau.Expression, to luau.Expression) {
	panic(todo("util/evaluation/facts.ts", "copyValueFacts"))
}

// setCallEffects is TSTransformer/util/evaluation/facts.ts setCallEffects.
func setCallEffects(expression luau.Expression, effects *EvaluationEffects) {
	panic(todo("util/evaluation/facts.ts", "setCallEffects"))
}

// getCallEffects is TSTransformer/util/evaluation/facts.ts getCallEffects.
func getCallEffects(expression luau.Expression) *EvaluationEffects {
	panic(todo("util/evaluation/facts.ts", "getCallEffects"))
}

// isConstantReference is TSTransformer/util/evaluation/facts.ts isConstantReference.
func isConstantReference(node luau.Node) bool {
	panic(todo("util/evaluation/facts.ts", "isConstantReference"))
}

// isPrimitiveValue is TSTransformer/util/evaluation/facts.ts isPrimitiveValue.
func isPrimitiveValue(node luau.Expression) bool {
	panic(todo("util/evaluation/facts.ts", "isPrimitiveValue"))
}

// markBinding is TSTransformer/util/evaluation/facts.ts markBinding.
func markBinding(identifier luau.AnyIdentifier, symbol *ast.Symbol, key *BindingKey) {
	panic(todo("util/evaluation/facts.ts", "markBinding"))
}

// getBinding is TSTransformer/util/evaluation/facts.ts getBinding.
func getBinding(identifier luau.AnyIdentifier) Binding {
	panic(todo("util/evaluation/facts.ts", "getBinding"))
}

// markBuiltinReference is TSTransformer/util/evaluation/facts.ts markBuiltinReference.
func markBuiltinReference(expression luau.Expression, effects *EvaluationEffects) {
	panic(todo("util/evaluation/facts.ts", "markBuiltinReference"))
}

// getBuiltinCallEffects is TSTransformer/util/evaluation/facts.ts getBuiltinCallEffects.
func getBuiltinCallEffects(expression luau.Expression) *EvaluationEffects {
	panic(todo("util/evaluation/facts.ts", "getBuiltinCallEffects"))
}

// isStableLookup is TSTransformer/util/evaluation/facts.ts isStableLookup.
func isStableLookup(expression luau.Expression) bool {
	panic(todo("util/evaluation/facts.ts", "isStableLookup"))
}
