package transform

// Port of TSTransformer/util/isMethod.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
)

func containsInstantiableType(type_ *checker.Type) bool {
	if isUnionOrIntersection(type_) {
		for _, t := range type_.Types() {
			if containsInstantiableType(t) {
				return true
			}
		}
		return false
	}
	return type_.Flags()&checker.TypeFlagsInstantiable != 0
}

func canChangeReceiverConvention(state *TransformState, type_ *checker.Type) bool {
	if !containsInstantiableType(type_) {
		return false
	}

	// a fixed non-void union member keeps the instantiated union from becoming void
	if type_.IsUnion() {
		for _, t := range type_.Types() {
			if t.Flags()&(checker.TypeFlagsVoid|checker.TypeFlagsNever) == 0 && !containsInstantiableType(t) {
				return false
			}
		}
	}

	if type_.Flags()&checker.TypeFlagsConditional != 0 {
		conditionalType := type_.AsConditionalType()
		root := conditionalType.Root()
		checkType := root.CheckType()
		extendsType := conditionalType.ExtendsType()
		// singleton tuples suppress distribution without changing the void test
		if checker.IsTupleType(checkType) && checker.IsTupleType(extendsType) {
			checkElements := getTypeArguments(state, checkType)
			extendsElements := getTypeArguments(state, extendsType)
			if len(checkElements) == 1 && len(extendsElements) == 1 {
				checkType = checkElements[0]
				extendsType = extendsElements[0]
			}
		}

		// a conditional that removes void can still have an unconstrained fallback type
		if state.TypeChecker.GetTypeFromTypeNode(root.Node().TrueType).Flags()&checker.TypeFlagsNever != 0 &&
			state.TypeChecker.GetTypeFromTypeNode(root.Node().FalseType) == checkType &&
			state.TypeChecker.IsTypeAssignableTo(state.TypeChecker.GetVoidType(), extendsType) {
			return false
		}
	}

	constraint := state.TypeChecker.GetBaseConstraintOfType(type_)
	return constraint == nil || state.TypeChecker.IsTypeAssignableTo(state.TypeChecker.GetVoidType(), constraint)
}

func isMethodDeclaration(node *ast.Node) bool {
	if ast.IsMethodDeclaration(node) || ast.IsMethodSignatureDeclaration(node) {
		return true
	}

	// object literal function expressions have an implicit receiver
	return ast.IsFunctionExpression(node) && ast.IsPropertyAssignment(skipUpwards(node).Parent)
}

func isMethodInner(state *TransformState, node *ast.Node, type_ *checker.Type) bool {
	hasMethodDefinition := false
	hasCallbackDefinition := false

	for _, callSignature := range state.TypeChecker.GetCallSignatures(type_) {
		thisParameter := callSignature.ThisParameter()
		if thisParameter != nil {
			thisType := state.TypeChecker.GetTypeOfSymbolAtLocation(thisParameter, node)
			if canChangeReceiverConvention(state, thisType) {
				AddDiagnosticWithCache(
					node,
					diagnostics.Errors.NoUnstableThisType.Create(node),
					state.MultiTransformState.IsReportedByNoUnstableThisType,
				)
			}
			if thisType.Flags()&checker.TypeFlagsVoid == 0 {
				hasMethodDefinition = true
			} else {
				hasCallbackDefinition = true
			}
		} else if callSignature.Declaration() != nil {
			if isMethodDeclaration(callSignature.Declaration()) {
				hasMethodDefinition = true
			} else {
				hasCallbackDefinition = true
			}
		}
	}

	if hasMethodDefinition && hasCallbackDefinition {
		DiagnosticService.AddDiagnostic(diagnostics.Errors.NoMixedTypeCall.Create(node))
	}

	return hasMethodDefinition
}

// isMethodFromType is TSTransformer/util/isMethod.ts isMethodFromType.
func isMethodFromType(state *TransformState, node *ast.Node, type_ *checker.Type) bool {
	result := false

	walkTypes(type_, func(t *checker.Type) {
		if t.Symbol() != nil {
			result = result || getOrSetDefault(state.MultiTransformState.IsMethodCache, t, func() bool { return isMethodInner(state, node, t) })
		}
	})

	return result
}

// isMethod is TSTransformer/util/isMethod.ts isMethod.
func isMethod(state *TransformState, node *ast.Node) bool {
	return isMethodFromType(state, node, state.GetType(node))
}
