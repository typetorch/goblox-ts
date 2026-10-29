package transform

// Port of TSTransformer/util/validateMethodAssignment.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
)

func hasCallSignatures(t *checker.Type) bool {
	hasCallSignatures := false
	walkTypes(t, func(t *checker.Type) {
		hasCallSignatures = hasCallSignatures || len(typeGetCallSignatures(t)) > 0
	})
	return hasCallSignatures
}

func isZeroParameterFunction(state *TransformState, node *ast.Node) bool {
	if ast.IsPropertyAssignment(node) {
		node = skipDownwards(node.Initializer())
	}

	if ast.IsIdentifier(node) || ast.IsShorthandPropertyAssignment(node) {
		var symbol *ast.Symbol
		if ast.IsShorthandPropertyAssignment(node) {
			symbol = state.TypeChecker.GetShorthandAssignmentValueSymbol(node)
		} else {
			symbol = state.TypeChecker.GetSymbolAtLocation(node)
		}
		assert(symbol != nil)
		declaration := symbol.ValueDeclaration

		if declaration != nil && ast.IsVariableDeclaration(declaration) {
			if declaration.Parent.Flags&ast.NodeFlagsConst == 0 || declaration.Initializer() == nil {
				return false
			}

			node = skipDownwards(declaration.Initializer())
		} else {
			// overload signatures can hide optional or rest parameters in the implementation
			var implementation *ast.Node
			for _, declaration := range symbol.Declarations {
				if ast.IsFunctionDeclaration(declaration) && declaration.Body() != nil {
					implementation = declaration
					break
				}
			}
			if implementation == nil {
				return false
			}

			node = implementation
		}
	}

	if !(ast.IsArrowFunction(node) || ast.IsFunctionExpression(node) || ast.IsFunctionDeclaration(node)) {
		return false
	}
	for _, parameter := range node.Parameters() {
		if !ast.IsThisIdentifier(parameter.Name()) {
			return false
		}
	}
	return !isMethod(state, node)
}

func validateTypes(state *TransformState, node *ast.Node, baseType *checker.Type, assignmentType *checker.Type) {
	if hasCallSignatures(baseType) && hasCallSignatures(assignmentType) {
		assignmentIsMethod := isMethodFromType(state, node, assignmentType)
		if isMethodFromType(state, node, baseType) != assignmentIsMethod {
			if assignmentIsMethod {
				// zero-parameter functions without a receiver ignore the extra receiver argument
				if isZeroParameterFunction(state, node) {
					return
				}

				DiagnosticService.AddDiagnostic(diagnostics.Errors.ExpectedMethodGotFunction.Create(node))
			} else {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.ExpectedFunctionGotMethod.Create(node))
			}
		}
	}
}

// validateMethodExpression is TSTransformer/util/validateMethodAssignment.ts validateMethodExpression.
func validateMethodExpression(state *TransformState, node *ast.Node, type_ *checker.Type) {
	parent := skipUpwards(node).Parent
	// object literal members are checked together with their contextual property types
	if ast.IsPropertyAssignment(parent) || ast.IsShorthandPropertyAssignment(parent) || !hasCallSignatures(type_) {
		return
	}

	contextualType := state.TypeChecker.GetContextualType(node, checker.ContextFlagsNone)
	if contextualType != nil && contextualType != type_ {
		expression := skipDownwards(node)
		// method extraction already reports why this value cannot be used as a callback
		if (ast.IsPropertyAccessExpression(expression) || ast.IsElementAccessExpression(expression)) &&
			!isValidMethodIndexWithoutCall(state, skipUpwards(node)) &&
			isMethodFromType(state, node, type_) {
			return
		}

		validateTypes(state, expression, type_, contextualType)
	}
}

func validateObjectLiteralElement(state *TransformState, node *ast.Node) {
	t := state.GetType(node)
	contextualType := state.TypeChecker.GetContextualTypeForObjectLiteralElement(node, checker.ContextFlagsNone)
	if contextualType != nil && contextualType != t {
		validateTypes(state, node, t, contextualType)
	}
}

func validateHeritageClause(state *TransformState, node *ast.Node, typeNode *ast.Node) {
	name := ast.GetPropertyNameForPropertyNameNode(node.Name())
	if name == "" || name == ast.InternalSymbolNameMissing {
		return
	}

	t := state.GetType(node)
	propertyType := state.TypeChecker.GetTypeOfPropertyOfType(state.GetType(typeNode), name)
	if propertyType == nil {
		return
	}

	validateTypes(state, node, t, propertyType)
}

func validateSpread(state *TransformState, node *ast.Node) {
	expression := node.Expression()
	t := state.GetType(expression)
	contextualType := state.TypeChecker.GetContextualType(expression, checker.ContextFlagsNone)
	if contextualType == nil {
		return
	}

	for _, property := range typeGetProperties(t) {
		basePropertyType := state.TypeChecker.GetTypeOfPropertyOfType(t, property.Name)
		assignmentPropertyType := state.TypeChecker.GetTypeOfPropertyOfType(contextualType, property.Name)
		if basePropertyType == nil {
			continue
		}
		if assignmentPropertyType == nil {
			continue
		}

		validateTypes(state, node, basePropertyType, assignmentPropertyType)
	}
}

// validateMethodAssignment is TSTransformer/util/validateMethodAssignment.ts validateMethodAssignment.
func validateMethodAssignment(state *TransformState, node *ast.Node) {
	if ast.IsClassElement(node) && ast.IsClassLike(node.Parent) && node.Name() != nil {
		for _, typeNode := range getAllSuperTypeNodes(node.Parent) {
			validateHeritageClause(state, node, typeNode)
		}
	} else if ast.IsObjectLiteralElement(node) {
		if ast.IsSpreadAssignment(node) {
			if !ast.IsObjectLiteralExpression(node.Expression()) {
				validateSpread(state, node)
			}
		} else {
			validateObjectLiteralElement(state, node)
		}
	}
}
