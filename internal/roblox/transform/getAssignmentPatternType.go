package transform

// Port of TSTransformer/util/binding/getAssignmentPatternType.ts.

import (
	"slices"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

func getIterableElementType(state *TransformState, type_ *checker.Type, node *ast.Node) *checker.Type {
	c := state.TypeChecker
	if isDefinitelyType(type_, isArrayType(state)) {
		element := typeGetNumberIndexType(type_)
		assert(element != nil)
		return element
	}

	iterable := c.GetDeclaredTypeOfSymbol(
		state.Services.MacroManager.RequireSymbol(SYMBOL_NAMES.Iterable),
	)
	iteratorName := typeGetProperties(iterable)[0].Name
	var iterator *ast.Symbol
	for _, property := range typeGetProperties(type_) {
		if property.Name == iteratorName {
			iterator = property
			break
		}
	}
	assert(iterator != nil)
	iteratorType := c.GetReturnTypeOfSignature(typeGetCallSignatures(c.GetTypeOfSymbolAtLocation(iterator, node))[0])
	next := c.GetTypeOfPropertyOfType(iteratorType, "next")
	assert(next != nil)
	result := c.GetReturnTypeOfSignature(typeGetCallSignatures(next)[0])
	var values []*checker.Type
	for _, part := range unionTypesOrSelf(result) {
		if c.GetTypeOfPropertyOfType(part, "done") != c.GetTrueType() {
			value := c.GetTypeOfPropertyOfType(part, "value")
			assert(value != nil)
			values = append(values, value)
		}
	}
	return c.GetUnionType(values)
}

// unionTypesOrSelf is `type.isUnion() ? type.types : [type]`.
func unionTypesOrSelf(t *checker.Type) []*checker.Type {
	if t.IsUnion() {
		return t.Types()
	}
	return []*checker.Type{t}
}

func getElementType(state *TransformState, pattern *ast.Node, index float64) *checker.Type {
	if ast.IsSpreadElement(pattern.Parent) {
		spread := pattern.Parent
		assert(ast.IsArrayLiteralExpression(spread.Parent))
		return getElementType(state, spread.Parent, float64(slices.Index(spread.Parent.Elements(), spread))+index)
	}
	type_ := getAssignmentPatternType(state, pattern)
	c := state.TypeChecker
	// contextual property lookup includes tuple positions beyond the fixed prefix
	if t := c.GetTypeOfPropertyOfContextualType(c.GetApparentType(type_), jsnum.Number(index).String()); t != nil {
		return t
	}
	return getIterableElementType(state, type_, pattern)
}

func getRestElementType(state *TransformState, pattern *ast.Node, index int) *checker.Type {
	if ast.IsSpreadElement(pattern.Parent) {
		spread := pattern.Parent
		assert(ast.IsArrayLiteralExpression(spread.Parent))
		return getRestElementType(state, spread.Parent, slices.Index(spread.Parent.Elements(), spread)+index)
	}

	c := state.TypeChecker
	type_ := c.GetApparentType(getAssignmentPatternType(state, pattern))
	var values []*checker.Type
	for _, part := range unionTypesOrSelf(type_) {
		if checker.IsTupleType(part) {
			tuple := part.TargetTupleType()
			elementFlags := tuple.ElementFlags()
			elements := c.GetTypeArguments(part)
			// a variable tail can still contribute elements after its first position was consumed
			for i := min(index, tuple.FixedLength()); i < len(elements); i++ {
				element := elements[i]
				// elementFlags[i] is undefined (so not Variadic) past the tuple's elements
				if i < len(elementFlags) && elementFlags[i]&checker.ElementFlagsVariadic != 0 {
					values = append(values, getIterableElementType(state, element, pattern))
				} else {
					values = append(values, element)
				}
			}
		} else {
			values = append(values, getIterableElementType(state, part, pattern))
		}
	}
	return c.GetUnionType(values)
}

func getPropertyType(state *TransformState, pattern *ast.Node, name *ast.Node) *checker.Type {
	c := state.TypeChecker
	type_ := getAssignmentPatternType(state, pattern)
	var keyType *checker.Type
	if ast.IsComputedPropertyName(name) {
		keyType = state.GetType(name.Expression())
	} else if ast.IsNumericLiteral(name) {
		keyType = c.GetNumberLiteralType(jsnum.FromString(name.Text()))
	} else {
		keyType = c.GetStringLiteralType(name.Text())
	}
	var values []*checker.Type
	for _, key := range unionTypesOrSelf(keyType) {
		var value *checker.Type
		if key.IsStringLiteral() || key.IsNumberLiteral() {
			if ast.IsSpreadElement(pattern.Parent) {
				spread := pattern.Parent
				assert(ast.IsArrayLiteralExpression(spread.Parent))
				value = getElementType(
					state,
					spread.Parent,
					float64(slices.Index(spread.Parent.Elements(), spread))+literalTypeNumberValue(key),
				)
			} else {
				value = c.GetTypeOfPropertyOfContextualType(
					c.GetApparentType(type_),
					literalTypeStringValue(key),
				)
			}
		}
		// valid array keys include numeric-string types, which use the number index signature
		if value == nil {
			if isDefinitelyType(type_, isArrayType(state)) || key.Flags()&checker.TypeFlagsNumberLike != 0 {
				value = c.GetIndexTypeOfType(type_, c.GetNumberType())
			} else {
				value = c.GetIndexTypeOfType(type_, c.GetStringType())
			}
		}
		assert(value != nil)
		values = append(values, value)
	}
	return c.GetUnionType(values)
}

// literalTypeStringValue is `String(key.value)` for a string or number literal type.
func literalTypeStringValue(t *checker.Type) string {
	if t.IsNumberLiteral() {
		return jsnum.Number(typeNumberValue(t)).String()
	}
	return typeStringValue(t)
}

// literalTypeNumberValue is `Number(key.value)` for a string or number literal type.
func literalTypeNumberValue(t *checker.Type) float64 {
	if t.IsNumberLiteral() {
		return typeNumberValue(t)
	}
	return float64(jsnum.FromString(typeStringValue(t)))
}

func hasRestAssignmentAncestor(pattern *ast.Node) bool {
	node := pattern
	for ast.IsAssignmentTarget(node) {
		parent := node.Parent
		if ast.IsSpreadElement(parent) {
			return true
		}
		// property wrappers belong to the pattern, but independent assignments end the search
		if ast.IsPropertyAssignment(parent) {
			node = parent.Parent
		} else {
			node = parent
		}
	}
	return false
}

// TypeScript's assignment-pattern query cannot traverse a rest element's parent
// resolve that part of the source path here, retaining tuple positions through nested rest

// getAssignmentPatternType is TSTransformer/util/binding/getAssignmentPatternType.ts getAssignmentPatternType.
func getAssignmentPatternType(state *TransformState, pattern *ast.Node) *checker.Type {
	if !hasRestAssignmentAncestor(pattern) {
		return state.TypeChecker.GetTypeOfAssignmentPattern(pattern)
	}
	parent := pattern.Parent
	if ast.IsSpreadElement(parent) {
		assert(ast.IsArrayLiteralExpression(parent.Parent))
		return state.TypeChecker.CreateArrayType(
			getRestElementType(state, parent.Parent, slices.Index(parent.Parent.Elements(), parent)),
		)
	}

	element := pattern
	if ast.IsBinaryExpression(parent) {
		element = parent
	}
	container := element.Parent
	var source *checker.Type
	if ast.IsArrayLiteralExpression(container) {
		source = getElementType(state, container, float64(slices.Index(container.Elements(), element)))
	} else {
		assert(ast.IsPropertyAssignment(container))
		source = getPropertyType(state, container.Parent, container.Name())
	}
	if ast.IsBinaryExpression(element) {
		return state.TypeChecker.GetUnionType([]*checker.Type{
			state.TypeChecker.GetNonNullableType(source),
			state.GetType(element.AsBinaryExpression().Right),
		})
	}
	return source
}
