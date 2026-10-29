package transform

// Port of TSTransformer/util/getConstantInteger.ts.

import (
	"math"
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

func getBindingInitializer(declaration *ast.Node) *ast.Node {
	if ast.IsVariableDeclaration(declaration) {
		return declaration.Initializer()
	}

	pattern := declaration.Parent
	parent := pattern.Parent
	if ast.IsArrayBindingPattern(pattern) &&
		ast.IsVariableDeclaration(parent) &&
		parent.Initializer() != nil &&
		declaration.AsBindingElement().DotDotDotToken == nil {
		initializer := skipDownwards(parent.Initializer())
		if ast.IsArrayLiteralExpression(initializer) && !slices.ContainsFunc(initializer.Elements(), ast.IsSpreadElement) {
			index := slices.Index(pattern.Elements(), declaration)
			elements := initializer.Elements()
			if index >= 0 && index < len(elements) {
				element := elements[index]
				if !ast.IsOmittedExpression(element) {
					return element
				}
			}
		}
	}
	return nil
}

func getConstantNumber(
	state *TransformState,
	expression *ast.Node,
	requireInitialized bool,
	getInteger func(expression *ast.Node) (float64, bool),
) (float64, bool) {
	if ast.IsNumericLiteral(expression) {
		return float64(jsnum.FromString(expression.Text())), true
	}

	if ast.IsPrefixUnaryExpression(expression) && expression.AsPrefixUnaryExpression().Operator == ast.KindMinusToken {
		if value, ok := getInteger(expression.AsPrefixUnaryExpression().Operand); ok {
			return -value, true
		}
	}

	if ast.IsBinaryExpression(expression) {
		binary := expression.AsBinaryExpression()
		left, leftOk := getInteger(binary.Left)
		right, rightOk := getInteger(binary.Right)
		if leftOk && rightOk {
			switch binary.OperatorToken.Kind {
			case ast.KindPlusToken:
				return left + right, true
			case ast.KindMinusToken:
				return left - right, true
			case ast.KindAsteriskToken:
				return left * right, true
			case ast.KindSlashToken:
				return left / right, true
			case ast.KindAsteriskAsteriskToken:
				return float64(jsnum.Number(left).Exponentiate(jsnum.Number(right))), true
			}
		}
	}

	if ast.IsIdentifier(expression) {
		symbol := state.TypeChecker.GetSymbolAtLocation(expression)
		assert(symbol != nil)
		declaration := symbol.ValueDeclaration
		if declaration == nil || (!ast.IsVariableDeclaration(declaration) && !ast.IsBindingElement(declaration)) {
			return 0, false
		}

		variable := declaration
		for ast.IsBindingElement(variable) {
			variable = variable.Parent.Parent
		}
		if !ast.IsVariableDeclaration(variable) ||
			ast.GetSourceFileOfNode(variable) != ast.GetSourceFileOfNode(expression) ||
			variable.Parent.Flags&ast.NodeFlagsConst == 0 ||
			(ast.GetCombinedModifierFlags(variable)&ast.ModifierFlagsAmbient) != 0 ||
			variable.Initializer() == nil {
			return 0, false
		}

		// steps are read before the body in a numeric loop, even if the source would break first
		// captured bindings may still be uninitialized when their function is called
		if requireInitialized {
			declarationStatement := variable.Parent.Parent
			if variable.End() > expression.Pos() ||
				!isAncestorOf(declarationStatement.Parent, expression) ||
				getAncestor(variable, ast.IsFunctionLike) != getAncestor(expression, ast.IsFunctionLike) {
				return 0, false
			}
		}

		initializer := getBindingInitializer(declaration)
		if initializer != nil {
			return getInteger(initializer)
		}
	}
	return 0, false
}

func isSafeInteger(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && math.Trunc(value) == value && math.Abs(value) <= 9007199254740991
}

// getConstantInteger is TSTransformer/util/getConstantInteger.ts getConstantInteger. numeric loops
// hoist bound and step reads, so reject effects and mutable references. Infer values from
// initializers, since literal-typed properties can change through wider aliases.
func getConstantInteger(state *TransformState, expression *ast.Node, requireInitialized bool) (float64, bool) {
	// share initializer results within this analysis, including failures to prove an integer
	// separate calls retain their own initialization requirements and reference checks
	type cached struct {
		value float64
		ok    bool
	}
	cache := map[*ast.Node]cached{}
	var getInteger func(expression *ast.Node) (float64, bool)
	getInteger = func(expression *ast.Node) (float64, bool) {
		expression = skipDownwards(expression)
		if c, has := cache[expression]; has {
			return c.value, c.ok
		}

		value, ok := getConstantNumber(state, expression, requireInitialized, getInteger)
		integer := cached{}
		if ok && isSafeInteger(value) {
			integer = cached{value, true}
		}
		cache[expression] = integer
		return integer.value, integer.ok
	}

	return getInteger(expression)
}
