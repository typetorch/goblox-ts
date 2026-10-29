package transform

// Port of TSTransformer/util/evaluation/builtins.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
)

var LIBRARIES = map[string]bool{
	"math": true, "string": true, "table": true, "bit32": true, "utf8": true,
	"coroutine": true, "task": true, "os": true, "debug": true, "buffer": true,
}

var PURE_MATH = map[string]bool{
	"abs": true, "acos": true, "asin": true, "atan": true, "atan2": true, "ceil": true, "cos": true, "cosh": true,
	"deg": true, "exp": true, "floor": true, "fmod": true, "frexp": true, "ldexp": true, "log": true, "log10": true,
	"max": true, "min": true, "modf": true, "noise": true, "pow": true, "rad": true, "round": true, "sign": true,
	"sin": true, "sinh": true, "sqrt": true, "tan": true, "tanh": true,
}

func isBuiltinDeclaration(state *TransformState, declaration *ast.Node) bool {
	return isPathDescendantOf(
		ast.GetSourceFileOfNode(declaration).FileName(),
		nodepath.Join(state.Data.NodeModulesPath, RBXTS_SCOPE, "types"),
	)
}

// isBuiltinLibrary is TSTransformer/util/evaluation/builtins.ts isBuiltinLibrary: check declaration
// identity, not spelling: local lookalikes are ordinary tables.
func isBuiltinLibrary(state *TransformState, node *ast.Node) bool {
	if !ast.IsIdentifier(node) || !LIBRARIES[node.Text()] {
		return false
	}
	symbol := state.TypeChecker.GetSymbolAtLocation(node)
	assert(symbol != nil)
	declarations := symbol.Declarations
	assert(declarations != nil)
	if len(declarations) == 0 {
		return false
	}
	for _, d := range declarations {
		if !(ast.IsModuleDeclaration(d) && isBuiltinDeclaration(state, d)) {
			return false
		}
	}
	return true
}

// tryMarkBuiltinLibrary is TSTransformer/util/evaluation/builtins.ts tryMarkBuiltinLibrary.
func tryMarkBuiltinLibrary(state *TransformState, node *ast.Node, expression luau.Expression) {
	if isBuiltinLibrary(state, node) {
		markConstantReference(expression)
	}
}

// isStableBuiltinMember is TSTransformer/util/evaluation/builtins.ts isStableBuiltinMember: native
// methods are immutable even when calling them can mutate, throw or yield.
func isStableBuiltinMember(state *TransformState, node *ast.Node) bool {
	if isBuiltinLibrary(state, node.Expression()) {
		return true
	}
	// optional-chain accesses run inside a nil guard; getNonOptionalType alone
	// does not remove a real T | undefined union from the original expression
	lookupType := func(expression *ast.Node) *checker.Type {
		if ast.IsOptionalChain(node) {
			return state.TypeChecker.GetNonNullableType(state.GetType(expression))
		}
		return state.TypeChecker.GetNonOptionalType(state.GetType(expression))
	}
	t := lookupType(node.Expression())
	if !getOrSetDefault(state.MultiTransformState.IsRobloxTypeCache, t, func() bool {
		return isDefinitelyType(t, isRobloxType(state))
	}) {
		return false
	}
	symbol := lookupType(node).Symbol()
	if symbol == nil || len(symbol.Declarations) == 0 {
		return false
	}
	for _, d := range symbol.Declarations {
		if !(ast.IsMethodSignatureDeclaration(d) && isBuiltinDeclaration(state, d)) {
			return false
		}
	}
	return true
}

// tryMarkBuiltinMember is TSTransformer/util/evaluation/builtins.ts tryMarkBuiltinMember.
func tryMarkBuiltinMember(state *TransformState, node *ast.Node, expression luau.Expression) {
	if !isStableBuiltinMember(state, node) {
		return
	}
	markStableLookup(expression)
	// these numeric functions are inert for their statically checked arguments.
	// clamp can throw even for numbers; random/randomseed mutate RNG state.
	if isBuiltinLibrary(state, node.Expression()) && node.Expression().Text() == "math" {
		name, hasName := "", false
		if ast.IsPropertyAccessExpression(node) {
			name, hasName = node.Name().Text(), true
		} else if argumentExpression := node.AsElementAccessExpression().ArgumentExpression; ast.IsStringLiteral(argumentExpression) {
			name, hasName = argumentExpression.Text(), true
		}
		call := node.Parent
		nonemptyArguments := ast.IsCallExpression(call) &&
			call.Expression() == node &&
			len(call.Arguments()) > 0 &&
			!ast.IsSpreadElement(call.Arguments()[0])
		if (name == "min" || name == "max") && !nonemptyArguments {
			// the types permit empty varargs, which throw. A function value must
			// retain that possibility until an actual call supplies an argument.
			setCallEffects(expression, withEffects(NO_EFFECTS, func(e *EvaluationEffects) { e.Throws = true }))
		} else if hasName && PURE_MATH[name] {
			setCallEffects(expression, NO_EFFECTS)
		} else if name == "clamp" {
			setCallEffects(expression, withEffects(NO_EFFECTS, func(e *EvaluationEffects) { e.Throws = true }))
		}
	}
}
