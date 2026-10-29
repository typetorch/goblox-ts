package transform

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
)

// Port of TSTransformer/util/types.ts.

// TypeCheck is a predicate over types.
type TypeCheck = func(t *checker.Type) bool

func (s *TransformState) getConstraint(t *checker.Type) *checker.Type {
	// TypeScript's Type.getConstraint(): the base constraint of a type parameter / indexed access
	if t.Flags()&checker.TypeFlagsInstantiable != 0 {
		return s.TypeChecker.GetBaseConstraintOfType(t)
	}
	return nil
}

func (s *TransformState) getBaseTypes(t *checker.Type) []*checker.Type {
	if t.Flags()&checker.TypeFlagsObject != 0 && t.ObjectFlags()&checker.ObjectFlagsClassOrInterface != 0 {
		return s.TypeChecker.GetBaseTypes(t)
	}
	return nil
}

func (s *TransformState) isDefinitelyTypeInner(t *checker.Type, callbacks []TypeCheck) bool {
	if t.IsUnion() {
		for _, u := range t.Types() {
			if !s.isDefinitelyTypeInner(u, callbacks) {
				return false
			}
		}
		return true
	} else if t.IsIntersection() {
		for _, u := range t.Types() {
			if s.isDefinitelyTypeInner(u, callbacks) {
				return true
			}
		}
		return false
	}
	// getBaseTypes() checks isClassOrInterface() internally
	for _, base := range s.getBaseTypes(t) {
		if s.isDefinitelyTypeInner(base, callbacks) {
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

func (s *TransformState) isDefinitelyType(t *checker.Type, callbacks ...TypeCheck) bool {
	if c := s.getConstraint(t); c != nil {
		t = c
	}
	return s.isDefinitelyTypeInner(t, callbacks)
}

func (s *TransformState) isPossiblyTypeInner(t *checker.Type, callbacks []TypeCheck, onlyUndefined bool) bool {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		for _, u := range t.Types() {
			if s.isPossiblyTypeInner(u, callbacks, onlyUndefined) {
				return true
			}
		}
		return false
	}
	// getBaseTypes() checks isClassOrInterface() internally
	for _, base := range s.getBaseTypes(t) {
		if s.isPossiblyTypeInner(base, callbacks, onlyUndefined) {
			return true
		}
	}
	// type variable without constraint, any, or unknown
	if t.Flags()&(checker.TypeFlagsTypeVariable|checker.TypeFlagsAnyOrUnknown) != 0 {
		return true
	}
	// defined type
	if s.isDefinedType(t) {
		// if only matching undefined, then defined means not possible
		return !onlyUndefined
	}
	for _, cb := range callbacks {
		if cb(t) {
			return true
		}
	}
	return false
}

// isPossiblyType is isPossiblyType. Pass onlyUndefined when the only callback is isUndefinedType
// (TypeScript compares the callback by identity, which Go cannot do).
func (s *TransformState) isPossiblyType(t *checker.Type, callbacks ...TypeCheck) bool {
	return s.isPossiblyTypeEx(t, false, callbacks...)
}

func (s *TransformState) isPossiblyUndefined(t *checker.Type) bool {
	return s.isPossiblyTypeEx(t, true, isUndefinedType)
}

func (s *TransformState) isPossiblyTypeEx(t *checker.Type, onlyUndefined bool, callbacks ...TypeCheck) bool {
	if c := s.getConstraint(t); c != nil {
		t = c
	}
	return s.isPossiblyTypeInner(t, callbacks, onlyUndefined)
}

func (s *TransformState) isDefinedType(t *checker.Type) bool {
	c := s.TypeChecker
	return t.Flags() == checker.TypeFlagsObject &&
		len(c.GetPropertiesOfType(t)) == 0 &&
		len(c.GetSignaturesOfType(t, checker.SignatureKindCall)) == 0 &&
		len(c.GetSignaturesOfType(t, checker.SignatureKindConstruct)) == 0 &&
		c.GetIndexTypeOfType(t, c.GetNumberType()) == nil &&
		c.GetIndexTypeOfType(t, c.GetStringType()) == nil
}

func (s *TransformState) isAnyType(t *checker.Type) bool { return t == s.TypeChecker.GetAnyType() }

func isBooleanType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsBoolean|checker.TypeFlagsBooleanLiteral) != 0
}

func (s *TransformState) isBooleanLiteralType(value bool) TypeCheck {
	return func(t *checker.Type) bool {
		if t.Flags()&checker.TypeFlagsBooleanLiteral != 0 {
			valueType := s.TypeChecker.GetFalseType()
			if value {
				valueType = s.TypeChecker.GetTrueType()
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
			return numberLiteralValue(t) == value
		}
		return isNumberType(t)
	}
}

func isNaNType(t *checker.Type) bool { return isNumberType(t) && !t.IsNumberLiteral() }

func isStringType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsString|checker.TypeFlagsStringLike|checker.TypeFlagsStringLiteral) != 0
}

func (s *TransformState) isMixedStringType(t *checker.Type) bool {
	if c := s.getConstraint(t); c != nil {
		t = c
	}
	if !t.IsUnion() || s.isDefinitelyType(t, isStringType) {
		return false
	}
	for _, u := range t.Types() {
		if s.isDefinitelyType(u, isStringType) {
			return true
		}
	}
	return false
}

func (s *TransformState) symbolIs(t *checker.Type, names ...string) bool {
	symbol := t.Symbol()
	if symbol == nil {
		return false
	}
	for _, name := range names {
		if symbol == s.Services.MacroManager.GetSymbolOrThrow(name) {
			return true
		}
	}
	return false
}

func (s *TransformState) isArrayType(t *checker.Type) bool {
	// typeChecker.isArrayLikeType() will return true for `any`, so rule it out here
	if t.Flags()&checker.TypeFlagsAny != 0 {
		return false
	}
	return checker.IsTupleType(t) || s.TypeChecker.IsArrayLikeType(t) ||
		s.symbolIs(t, SYMBOL_NAMES.ReadonlyArray, SYMBOL_NAMES.Array, SYMBOL_NAMES.ReadVoxelsArray, SYMBOL_NAMES.TemplateStringsArray)
}

func (s *TransformState) isSetType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.Set, SYMBOL_NAMES.ReadonlySet, SYMBOL_NAMES.WeakSet)
}

func (s *TransformState) isMapType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.Map, SYMBOL_NAMES.ReadonlyMap, SYMBOL_NAMES.WeakMap)
}

func (s *TransformState) isSharedTableType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.SharedTable)
}

func (s *TransformState) isGeneratorType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.Generator)
}

func (s *TransformState) isIterableFunctionType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.IterableFunction)
}

func (s *TransformState) isLuaTupleType(t *checker.Type) bool {
	nominal := s.Services.MacroManager.symbols[NOMINAL_LUA_TUPLE_NAME]
	return nominal != nil && s.TypeChecker.GetPropertyOfType(t, NOMINAL_LUA_TUPLE_NAME) == nominal
}

func (s *TransformState) isIterableFunctionLuaTupleType(t *checker.Type) bool {
	if s.isIterableFunctionType(t) {
		args := s.getTypeArguments(t)
		return len(args) > 0 && s.isLuaTupleType(args[0])
	}
	return false
}

func (s *TransformState) isIterableType(t *checker.Type) bool {
	return s.symbolIs(t, SYMBOL_NAMES.Iterable)
}

func isObjectType(t *checker.Type) bool { return t.Flags()&checker.TypeFlagsObject != 0 }

func isUndefinedType(t *checker.Type) bool {
	return t.Flags()&(checker.TypeFlagsUndefined|checker.TypeFlagsVoid) != 0
}

func isEmptyStringType(t *checker.Type) bool {
	if t.IsStringLiteral() {
		return t.AsLiteralType().Value() == ""
	}
	if t.Flags()&checker.TypeFlagsTemplateLiteral != 0 {
		texts := t.AsTemplateLiteralType().Texts()
		for _, v := range texts {
			if v != "" {
				return false
			}
		}
		return true
	}
	return isStringType(t)
}

func (s *TransformState) isRobloxType(t *checker.Type) bool {
	typesPath := nodepath.Join(s.Data.NodeModulesPath, RBXTS_SCOPE, "types")
	symbol := t.Symbol()
	if symbol == nil {
		return false
	}
	for _, d := range symbol.Declarations {
		filePath := ast.GetSourceFileOfNode(d).FileName()
		if isPathDescendantOf(filePath, typesPath) {
			return true
		}
	}
	return false
}

// isPathDescendantOf is Shared/util/isPathDescendantOf.ts.
func isPathDescendantOf(filePath string, dirPath string) bool {
	return dirPath == filePath || !hasDotDotPrefix(nodepath.Relative(dirPath, filePath))
}

func hasDotDotPrefix(p string) bool { return len(p) >= 2 && p[:2] == ".." }

// type utilities

func (s *TransformState) walkTypes(t *checker.Type, callback func(*checker.Type)) {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		for _, u := range t.Types() {
			s.walkTypes(u, callback)
		}
		return
	}
	// in template literal types, constraint === type and this causes infinite recursion
	constraint := s.getConstraint(t)
	if constraint != nil && constraint != t {
		s.walkTypes(constraint, callback)
	} else {
		callback(t)
	}
}

func (s *TransformState) getFirstConstructSymbol(expression *ast.Node) *ast.Symbol {
	t := s.GetType(expression)
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

func (s *TransformState) getFirstDefinedSymbol(t *checker.Type) *ast.Symbol {
	if t.Flags()&checker.TypeFlagsUnionOrIntersection != 0 {
		for _, u := range t.Types() {
			if u.Symbol() != nil && !s.TypeChecker.IsUndefinedSymbol(u.Symbol()) {
				return u.Symbol()
			}
		}
		return nil
	}
	return t.Symbol()
}

func (s *TransformState) getTypeArguments(t *checker.Type) []*checker.Type {
	if t.ObjectFlags()&checker.ObjectFlagsReference == 0 {
		return nil
	}
	return s.TypeChecker.GetTypeArguments(t)
}

func numberLiteralValue(t *checker.Type) float64 {
	value, _ := numberValue(t.AsLiteralType().Value())
	return value
}
