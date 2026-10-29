package transform

// Port of TSTransformer/nodes/statements/transformVariableStatement.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// transformVariable is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariable.
// right is nil when absent.
func transformVariable(state *TransformState, prereqs *Prereqs, identifier *ast.Node, right luau.Expression) luau.Expression {
	validateIdentifier(identifier)

	symbol := state.TypeChecker.GetSymbolAtLocation(identifier)
	assert(symbol != nil)

	// export let
	if isSymbolMutable(state, symbol) {
		exportAccess := state.GetModuleIdPropertyAccess(symbol)
		if exportAccess != nil {
			if right != nil {
				prereqs.Push(luau.Create(&luau.Assignment{
					Left:     exportAccess,
					Operator: "=",
					Right:    right,
				}))
			}
			return exportAccess
		}
	}

	left := transformIdentifierDefined(state, identifier)
	if right != nil && isConstantReference(left) {
		effects := getCallEffects(right)
		if effects != nil {
			state.MultiTransformState.FunctionEffects[symbol] = effects
		}
		copyValueFacts(right, left)
	}

	checkVariableHoist(state, identifier, symbol)
	if state.IsHoisted[symbol] {
		// no need to do `x = nil` if the variable is already created
		if right != nil {
			prereqs.Push(luau.Create(&luau.Assignment{Left: left, Operator: "=", Right: right}))
		}
	} else {
		var rightValue luau.ListOrNode
		if right != nil {
			rightValue = right
		}
		prereqs.Push(luau.Create(&luau.VariableDeclaration{Left: left, Right: rightValue}))
	}

	return left
}

func transformOptimizedArrayBindingPattern(state *TransformState, bindingPattern *ast.Node, rhs luau.ListOrNode) *luau.List {
	ids := luau.MakeList()
	bindingStatements := luau.MakeList()
	for _, element := range bindingPattern.Elements() {
		if ast.IsOmittedExpression(element) {
			ids.Push(luau.TempId())
		} else {
			if ast.IsIdentifier(element.Name()) {
				validateIdentifier(element.Name())
				id := transformIdentifierDefined(state, element.Name())
				ids.Push(id)
				if element.Initializer() != nil {
					bindingStatements.Push(transformInitializer(state, id, element.Initializer()))
				}
			} else {
				id := luau.TempId("binding")
				ids.Push(id)
				if element.Initializer() != nil {
					bindingStatements.Push(transformInitializer(state, id, element.Initializer()))
				}

				bindingPrereqs := NewPrereqs()
				if ast.IsArrayBindingPattern(element.Name()) {
					transformArrayBindingPattern(state, bindingPrereqs, element.Name(), id)
				} else {
					transformObjectBindingPattern(state, bindingPrereqs, element.Name(), id)
				}
				bindingStatements.PushList(bindingPrereqs.Statements)
			}
		}
	}

	assert(!ids.IsEmpty())
	statements := luau.MakeList(
		luau.Create(&luau.VariableDeclaration{Left: ids, Right: rhs}),
	)
	statements.PushList(bindingStatements)
	return statements
}

func transformOptimizedObjectBindingPattern(state *TransformState, bindingPattern *ast.Node, rhs luau.Expression) *luau.List {
	validateNotAnyType(state, bindingPattern)
	element := bindingPattern.Elements()[0]
	assert(ast.IsIdentifier(element.Name()))

	// a single named property reads the receiver once without computed key prerequisites
	prereqs := NewPrereqs()
	propertyName := element.PropertyName()
	if propertyName == nil {
		propertyName = element.Name()
	}
	_, access := objectAccessor(
		state,
		prereqs,
		convertToIndexableExpression(rhs),
		state.GetType(bindingPattern),
		propertyName,
	)
	id := transformVariable(state, prereqs, element.Name(), access)
	if element.Initializer() != nil {
		prereqs.Push(transformInitializer(state, id, element.Initializer()))
	}

	return prereqs.Statements
}

// transformVariableDeclaration is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableDeclaration.
func transformVariableDeclaration(state *TransformState, node *ast.Node) *luau.List {
	statements := luau.MakeList()
	var value luau.Expression
	if node.Initializer() != nil {
		// must transform right _before_ checking isHoisted, that way references inside of value can be hoisted
		initializerPrereqs := NewPrereqs()
		value = transformExpression(state, initializerPrereqs, node.Initializer())
		statements.PushList(initializerPrereqs.Statements)
	}

	name := node.Name()
	if ast.IsIdentifier(name) {
		variablePrereqs := NewPrereqs()
		transformVariable(state, variablePrereqs, name, value)
		statements.PushList(variablePrereqs.Statements)
	} else {
		// in destructuring, rhs must be executed first
		assert(node.Initializer() != nil && value != nil)

		// optimize empty destructure
		if len(name.Elements()) == 0 {
			if array, ok := value.(*luau.Array); !ok || !array.Members.IsEmpty() {
				statements.PushList(wrapExpressionStatement(value))
			}
			return statements
		}

		if ast.IsArrayBindingPattern(name) {
			if luau.IsCall(value) &&
				isLuaTupleType(state)(state.GetType(node.Initializer())) &&
				!arrayBindingPatternContainsHoists(state, name) &&
				!arrayLikeExpressionContainsSpread(name) {
				statements.PushList(transformOptimizedArrayBindingPattern(state, name, value))
			} else if array, ok := value.(*luau.Array); ok &&
				!array.Members.IsEmpty() &&
				// we can't localize multiple variables at the same time if any of them are hoisted
				!arrayBindingPatternContainsHoists(state, name) &&
				!arrayLikeExpressionContainsSpread(name) {
				statements.PushList(transformOptimizedArrayBindingPattern(state, name, array.Members))
			} else {
				bindingPrereqs := NewPrereqs()
				transformBindingPattern(state, bindingPrereqs, name, value)
				statements.PushList(bindingPrereqs.Statements)
			}
		} else {
			element := name.Elements()[0]
			if len(name.Elements()) == 1 &&
				ast.IsIdentifier(element.Name()) &&
				element.AsBindingElement().DotDotDotToken == nil &&
				(element.PropertyName() == nil || ast.IsIdentifier(element.PropertyName())) {
				statements.PushList(transformOptimizedObjectBindingPattern(state, name, value))
			} else {
				bindingPrereqs := NewPrereqs()
				transformBindingPattern(state, bindingPrereqs, name, value)
				statements.PushList(bindingPrereqs.Statements)
			}
		}
	}

	return statements
}

// isVarDeclaration is TSTransformer/nodes/statements/transformVariableStatement.ts isVarDeclaration.
func isVarDeclaration(node *ast.Node) bool {
	return node.Flags&ast.NodeFlagsConst == 0 && node.Flags&ast.NodeFlagsLet == 0
}

// transformVariableDeclarationList is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableDeclarationList.
func transformVariableDeclarationList(state *TransformState, node *ast.Node) *luau.List {
	if isVarDeclaration(node) {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoVar.Create(node))
	}

	statements := luau.MakeList()
	for _, declaration := range node.AsVariableDeclarationList().Declarations.Nodes {
		statements.PushList(transformVariableDeclaration(state, declaration))
	}

	return statements
}

// transformVariableStatement is TSTransformer/nodes/statements/transformVariableStatement.ts transformVariableStatement.
func transformVariableStatement(state *TransformState, node *ast.Node) *luau.List {
	return transformVariableDeclarationList(state, node.AsVariableStatement().DeclarationList)
}
