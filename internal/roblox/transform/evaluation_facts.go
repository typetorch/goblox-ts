package transform

// Port of TSTransformer/util/evaluation/facts.ts.

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

// annotated is the set of symbol-keyed facts of a luau node. It is stored as a *annotated through
// luau.SetFacts and updated in place; luau gives every clone its own copy (CloneFacts), so a shallow clone
// keeps the facts its original had when it was cloned (luau.create copies symbol keys).
type annotated struct {
	binding      Binding // nil when absent
	constant     bool
	builtin      *EvaluationEffects
	primitive    bool
	callEffects  *EvaluationEffects
	stableLookup bool
}

// CloneFacts implements luau.FactsCloner.
func (facts *annotated) CloneFacts() any {
	clone := *facts
	return &clone
}

func getAnnotated(node luau.Node) annotated {
	if facts, ok := luau.GetFacts(node).(*annotated); ok {
		return *facts
	}
	return annotated{}
}

func updateAnnotated(node luau.Node, update func(facts *annotated)) {
	facts, ok := luau.GetFacts(node).(*annotated)
	if !ok {
		facts = &annotated{}
		luau.SetFacts(node, facts)
	}
	update(facts)
}

// markConstantReference is TSTransformer/util/evaluation/facts.ts markConstantReference.
func markConstantReference(expression luau.Expression) {
	updateAnnotated(expression, func(facts *annotated) { facts.constant = true })
}

// markStableLookup is TSTransformer/util/evaluation/facts.ts markStableLookup: the lookup is inert;
// evaluating the receiver and invoking the member are not.
func markStableLookup(expression luau.Expression) {
	updateAnnotated(expression, func(facts *annotated) { facts.stableLookup = true })
}

// markPrimitiveValue is TSTransformer/util/evaluation/facts.ts markPrimitiveValue.
func markPrimitiveValue(expression luau.Expression) {
	updateAnnotated(expression, func(facts *annotated) { facts.primitive = true })
}

// copyValueFacts is TSTransformer/util/evaluation/facts.ts copyValueFacts.
func copyValueFacts(from luau.Expression, to luau.Expression) {
	if getAnnotated(from).primitive {
		markPrimitiveValue(to)
	}
	effects := getAnnotated(from).callEffects
	if effects != nil {
		updateAnnotated(to, func(facts *annotated) { facts.callEffects = effects })
	}
}

// setCallEffects is TSTransformer/util/evaluation/facts.ts setCallEffects.
func setCallEffects(expression luau.Expression, effects *EvaluationEffects) {
	updateAnnotated(expression, func(facts *annotated) { facts.callEffects = effects })
}

// getCallEffects is TSTransformer/util/evaluation/facts.ts getCallEffects.
func getCallEffects(expression luau.Expression) *EvaluationEffects {
	return getAnnotated(expression).callEffects
}

// isConstantReference is TSTransformer/util/evaluation/facts.ts isConstantReference.
func isConstantReference(node luau.Node) bool {
	return getAnnotated(node).constant
}

// isPrimitiveValue is TSTransformer/util/evaluation/facts.ts isPrimitiveValue.
func isPrimitiveValue(node luau.Expression) bool {
	if getAnnotated(node).primitive || luau.IsSimplePrimitive(node) {
		return true
	}
	if parenthesized, ok := node.(*luau.ParenthesizedExpression); ok {
		return isPrimitiveValue(parenthesized.Expression)
	}
	return false
}

// markBinding is TSTransformer/util/evaluation/facts.ts markBinding.
func markBinding(identifier luau.AnyIdentifier, symbol *ast.Symbol, key *BindingKey) {
	updateAnnotated(identifier, func(facts *annotated) { facts.binding = key })
	declaration := symbol.ValueDeclaration
	for _, d := range symbol.Declarations {
		if ast.IsImportSpecifier(d) ||
			ast.IsImportClause(d) ||
			ast.IsNamespaceImport(d) ||
			ast.IsImportEqualsDeclaration(d) {
			markConstantReference(identifier)
			break
		}
	}
	if declaration != nil && ast.IsFunctionDeclaration(declaration) {
		updateAnnotated(identifier, func(facts *annotated) { facts.constant = true })
	}
	parent := declaration
	for parent != nil &&
		(ast.IsBindingElement(parent) || ast.IsObjectBindingPattern(parent) || ast.IsArrayBindingPattern(parent)) {
		parent = parent.Parent
	}
	if parent != nil && ast.IsVariableDeclaration(parent) && ast.IsVariableDeclarationList(parent.Parent) {
		isConst := parent.Parent.Flags&ast.NodeFlagsConst != 0
		updateAnnotated(identifier, func(facts *annotated) { facts.constant = isConst })
	}
}

// getBinding is TSTransformer/util/evaluation/facts.ts getBinding.
func getBinding(identifier luau.AnyIdentifier) Binding {
	if binding := getAnnotated(identifier).binding; binding != nil {
		return binding
	}
	if temporary, ok := identifier.(*luau.TemporaryIdentifier); ok {
		return temporary.Id
	}
	return identifier.(*luau.Identifier).Name
}

// markBuiltinReference is TSTransformer/util/evaluation/facts.ts markBuiltinReference.
func markBuiltinReference(expression luau.Expression, effects *EvaluationEffects) {
	updateAnnotated(expression, func(facts *annotated) {
		facts.constant = true
		facts.builtin = effects
	})
}

// getBuiltinCallEffects is TSTransformer/util/evaluation/facts.ts getBuiltinCallEffects.
func getBuiltinCallEffects(expression luau.Expression) *EvaluationEffects {
	return getAnnotated(expression).builtin
}

// isStableLookup is TSTransformer/util/evaluation/facts.ts isStableLookup.
func isStableLookup(expression luau.Expression) bool {
	return getAnnotated(expression).stableLookup
}
