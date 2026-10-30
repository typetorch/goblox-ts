package transform

// Port of TSTransformer/util/types.ts.

import (
	"reflect"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/jsnum"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
)

// TypeCheck is a predicate over types.
type TypeCheck = func(t *checker.Type) bool

// The methods of TypeScript's TypeObject that the transformer uses. A type knows its checker.

// typeGetConstraint is TypeScript's Type.getConstraint().
func typeGetConstraint(t *checker.Type) *checker.Type {
	return t.Checker().GetBaseConstraintOfType(t)
}

// typeGetConstraintOrSelf is `type.getConstraint() ?? type`.
func typeGetConstraintOrSelf(t *checker.Type) *checker.Type {
	if constraint := typeGetConstraint(t); constraint != nil {
		return constraint
	}
	return t
}

// typeGetBaseTypes is TypeScript's Type.getBaseTypes().
func typeGetBaseTypes(t *checker.Type) []*checker.Type {
	if t.Flags()&checker.TypeFlagsObject != 0 && t.ObjectFlags()&checker.ObjectFlagsClassOrInterface != 0 {
		return t.Checker().GetBaseTypes(t)
	}
	return nil
}

// typeGetProperty is TypeScript's Type.getProperty(name).
func typeGetProperty(t *checker.Type, name string) *ast.Symbol {
	return t.Checker().GetPropertyOfType(t, name)
}

// typeGetProperties is TypeScript's Type.getProperties().
func typeGetProperties(t *checker.Type) []*ast.Symbol {
	return t.Checker().GetPropertiesOfType(t)
}

// typeGetCallSignatures is TypeScript's Type.getCallSignatures().
func typeGetCallSignatures(t *checker.Type) []*checker.Signature {
	return t.Checker().GetSignaturesOfType(t, checker.SignatureKindCall)
}

// typeGetConstructSignatures is TypeScript's Type.getConstructSignatures().
func typeGetConstructSignatures(t *checker.Type) []*checker.Signature {
	return t.Checker().GetSignaturesOfType(t, checker.SignatureKindConstruct)
}

// typeGetNumberIndexType is TypeScript's Type.getNumberIndexType().
func typeGetNumberIndexType(t *checker.Type) *checker.Type {
	c := t.Checker()
	return c.GetIndexTypeOfType(t, c.GetNumberType())
}

// typeGetStringIndexType is TypeScript's Type.getStringIndexType().
func typeGetStringIndexType(t *checker.Type) *checker.Type {
	c := t.Checker()
	return c.GetIndexTypeOfType(t, c.GetStringType())
}

// typeTypes is TypeScript's UnionOrIntersectionType.types.
func typeTypes(t *checker.Type) []*checker.Type {
	return t.Types()
}

// typeNumberValue is a NumberLiteralType's value.
func typeNumberValue(t *checker.Type) float64 {
	return float64(t.AsLiteralType().Value().(jsnum.Number))
}

// typeStringValue is a StringLiteralType's value.
func typeStringValue(t *checker.Type) string {
	return t.AsLiteralType().Value().(string)
}

func isUnionOrIntersection(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsUnionOrIntersection != 0
}

func isDefinitelyTypeInner(t *checker.Type, callbacks []TypeCheck) bool {
	if t.IsUnion() {
		for _, u := range t.Types() {
			if !isDefinitelyTypeInner(u, callbacks) {
				return false
			}
		}
		return true
	} else if t.IsIntersection() {
		for _, u := range t.Types() {
			if isDefinitelyTypeInner(u, callbacks) {
				return true
			}
		}
		return false
	} else {
		// getBaseTypes() checks isClassOrInterface() internally
		for _, base := range typeGetBaseTypes(t) {
			if isDefinitelyTypeInner(base, callbacks) {
				return true
			}
		}
		for _, cb := range callbacks {
			if cb(t) {
				return true
			}
		}
		return false
	}
}

func isDefinitelyType(t *checker.Type, callbacks ...TypeCheck) bool {
	return isDefinitelyTypeInner(typeGetConstraintOrSelf(t), callbacks)
}

func isPossiblyTypeInner(t *checker.Type, callbacks []TypeCheck) bool {
	if isUnionOrIntersection(t) {
		for _, u := range t.Types() {
			if isPossiblyTypeInner(u, callbacks) {
				return true
			}
		}
		return false
	} else {
		// getBaseTypes() checks isClassOrInterface() internally
		for _, base := range typeGetBaseTypes(t) {
			if isPossiblyTypeInner(base, callbacks) {
				return true
			}
		}

		// type variable without constraint, any, or unknown
		if t.Flags()&(checker.TypeFlagsTypeVariable|checker.TypeFlagsAnyOrUnknown) != 0 {
			return true
		}

		// defined type
		if isDefinedType(t) {
			if len(callbacks) == 1 && sameFunc(callbacks[0], isUndefinedType) {
				// if only matching undefined, then defined means not possible
				return false
			}
			return true
		}

		for _, cb := range callbacks {
			if cb(t) {
				return true
			}
		}
		return false
	}
}

// sameFunc is `===` on two top-level functions.
func sameFunc(a, b TypeCheck) bool {
	return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
}

func isPossiblyType(t *checker.Type, callbacks ...TypeCheck) bool {
	return isPossiblyTypeInner(typeGetConstraintOrSelf(t), callbacks)
}

func isDefinedType(t *checker.Type) bool {
	return t.Flags() == checker.TypeFlagsObject &&
		len(typeGetProperties(t)) == 0 &&
		len(typeGetCallSignatures(t)) == 0 &&
		len(typeGetConstructSignatures(t)) == 0 &&
		typeGetNumberIndexType(t) == nil &&
		typeGetStringIndexType(t) == nil
}

func isAnyType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool { return t == state.TypeChecker.GetAnyType() }
}

func isBooleanType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsBoolean|checker.TypeFlagsBooleanLiteral) != 0
}

func isBooleanLiteralType(state *TransformState, value bool) TypeCheck {
	return func(t *checker.Type) bool {
		if t.Flags()&checker.TypeFlagsBooleanLiteral != 0 {
			valueType := state.TypeChecker.GetFalseType()
			if value {
				valueType = state.TypeChecker.GetTrueType()
			}
			return t == valueType
		}
		return isBooleanType(t)
	}
}

func isNumberType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsNumber|checker.TypeFlagsNumberLike|checker.TypeFlagsNumberLiteral) != 0
}

func isNumberLiteralType(value float64) TypeCheck {
	return func(t *checker.Type) bool {
		if t.IsNumberLiteral() {
			return typeNumberValue(t) == value
		}
		return isNumberType(t)
	}
}

func isNaNType(t *checker.Type) bool {
	return isNumberType(t) && !t.IsNumberLiteral()
}

func isStringType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsString|checker.TypeFlagsStringLike|checker.TypeFlagsStringLiteral) != 0
}

func isMixedStringType(t *checker.Type) bool {
	t = typeGetConstraintOrSelf(t)
	if !t.IsUnion() || isDefinitelyType(t, isStringType) {
		return false
	}
	for _, u := range t.Types() {
		if isDefinitelyType(u, isStringType) {
			return true
		}
	}
	return false
}

func isArrayType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		// typeChecker.isArrayLikeType() will return true for `any`, so rule it out here
		if t.Flags()&checker.TypeFlagsAny != 0 {
			return false
		}
		macroManager := state.Services.MacroManager
		return checker.IsTupleType(t) ||
			state.TypeChecker.IsArrayLikeType(t) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.ReadonlyArray) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.Array) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.ReadVoxelsArray) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.TemplateStringsArray)
	}
}

func isSetType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		macroManager := state.Services.MacroManager
		return t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.Set) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.ReadonlySet) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.WeakSet)
	}
}

func isMapType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		macroManager := state.Services.MacroManager
		return t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.Map) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.ReadonlyMap) ||
			t.Symbol() == macroManager.GetSymbolOrThrow(SYMBOL_NAMES.WeakMap)
	}
}

func isSharedTableType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.SharedTable)
	}
}

func isGeneratorType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.Generator)
	}
}

func isIterableFunctionType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.IterableFunction)
	}
}

func isLuaTupleType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		return typeGetProperty(t, NOMINAL_LUA_TUPLE_NAME) ==
			state.Services.MacroManager.GetSymbolOrThrow(NOMINAL_LUA_TUPLE_NAME)
	}
}

func isIterableFunctionLuaTupleType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		if isIterableFunctionType(state)(t) {
			typeArguments := getTypeArguments(state, t)
			return len(typeArguments) > 0 && isLuaTupleType(state)(typeArguments[0])
		}
		return false
	}
}

func isIterableType(state *TransformState) TypeCheck {
	return func(t *checker.Type) bool {
		return t.Symbol() == state.Services.MacroManager.GetSymbolOrThrow(SYMBOL_NAMES.Iterable)
	}
}

func isObjectType(t *checker.Type) bool {
	return t.Flags()&checker.TypeFlagsObject != 0
}

func isUndefinedType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0
}

func isEmptyStringType(t *checker.Type) bool {
	if t.IsStringLiteral() {
		return typeStringValue(t) == ""
	}
	if isTemplateLiteralType(t) {
		texts := t.AsTemplateLiteralType().Texts()
		for _, v := range texts {
			if len(v) != 0 {
				return false
			}
		}
		return true
	}
	return isStringType(t)
}

func isRobloxType(state *TransformState) TypeCheck {
	multiTransformState := state.MultiTransformState
	if multiTransformState.isRobloxTypeCheck == nil {
		typesPath := nodepath.Join(state.Data.NodeModulesPath, RBXTS_SCOPE, "types")
		multiTransformState.isRobloxTypeCheck = func(t *checker.Type) bool {
			symbol := t.Symbol()
			if symbol == nil {
				return false
			}
			return getOrSetDefault(multiTransformState.IsRobloxSymbolCache, symbol, func() bool {
				for _, d := range symbol.Declarations {
					filePath := ast.GetSourceFileOfNode(d).FileName()
					if isPathDescendantOf(filePath, typesPath) {
						return true
					}
				}
				return false
			})
		}
	}
	return multiTransformState.isRobloxTypeCheck
}

// isPathDescendantOf is Shared/util/isPathDescendantOf.ts.
func isPathDescendantOf(filePath string, dirPath string) bool {
	return nodepath.IsPathDescendantOf(filePath, dirPath)
}

// type utilities

func walkTypes(t *checker.Type, callback func(t *checker.Type)) {
	if isUnionOrIntersection(t) {
		for _, u := range t.Types() {
			walkTypes(u, callback)
		}
	} else {
		// in template literal types, constraint === type and this causes infinite recursion
		constraint := typeGetConstraint(t)
		if constraint != nil && constraint != t {
			walkTypes(constraint, callback)
		} else {
			callback(t)
		}
	}
}

func getFirstConstructSymbol(state *TransformState, expression *ast.Node) *ast.Symbol {
	t := state.GetType(expression)
	if symbol := t.Symbol(); symbol != nil {
		for _, declaration := range symbol.Declarations {
			if ast.IsInterfaceDeclaration(declaration) {
				for _, member := range declaration.Members() {
					if ast.IsConstructSignatureDeclaration(member) {
						return member.Symbol()
					}
				}
			}
		}
	}
	return nil
}

func getFirstDefinedSymbol(state *TransformState, t *checker.Type) *ast.Symbol {
	if isUnionOrIntersection(t) {
		for _, u := range t.Types() {
			if u.Symbol() != nil && !state.TypeChecker.IsUndefinedSymbol(u.Symbol()) {
				return u.Symbol()
			}
		}
		return nil
	}
	return t.Symbol()
}

func getTypeArguments(state *TransformState, t *checker.Type) []*checker.Type {
	// TypeScript's getTypeArguments reads the reference's resolvedTypeArguments; non-references have none
	if t.ObjectFlags()&checker.ObjectFlagsReference == 0 {
		return nil
	}
	return state.TypeChecker.GetTypeArguments(t)
}
