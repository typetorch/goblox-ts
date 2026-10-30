package transform

// Port of TSTransformer/classes/MacroManager.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
)

func macroManagerGetType(typeChecker *checker.Checker, node *ast.Node) *checker.Type {
	return typeChecker.GetTypeAtLocation(skipUpwards(node))
}

const TYPES_NOTICE = "\nYou may need to update your @rbxts/compiler-types!"

var SYMBOL_NAMES = struct {
	GlobalThis string

	ArrayConstructor       string
	SetConstructor         string
	MapConstructor         string
	WeakSetConstructor     string
	WeakMapConstructor     string
	ReadonlyMapConstructor string
	ReadonlySetConstructor string

	Array                string
	Generator            string
	IterableFunction     string
	LuaTuple             string
	Map                  string
	Object               string
	ReadonlyArray        string
	ReadonlyMap          string
	ReadonlySet          string
	ReadVoxelsArray      string
	Set                  string
	SharedTable          string
	String               string
	TemplateStringsArray string
	WeakMap              string
	WeakSet              string

	Iterable string

	Range string
	Tuple string
}{
	GlobalThis: "globalThis",

	ArrayConstructor:       "ArrayConstructor",
	SetConstructor:         "SetConstructor",
	MapConstructor:         "MapConstructor",
	WeakSetConstructor:     "WeakSetConstructor",
	WeakMapConstructor:     "WeakMapConstructor",
	ReadonlyMapConstructor: "ReadonlyMapConstructor",
	ReadonlySetConstructor: "ReadonlySetConstructor",

	Array:                "Array",
	Generator:            "Generator",
	IterableFunction:     "IterableFunction",
	LuaTuple:             "LuaTuple",
	Map:                  "Map",
	Object:               "Object",
	ReadonlyArray:        "ReadonlyArray",
	ReadonlyMap:          "ReadonlyMap",
	ReadonlySet:          "ReadonlySet",
	ReadVoxelsArray:      "ReadVoxelsArray",
	Set:                  "Set",
	SharedTable:          "SharedTable",
	String:               "String",
	TemplateStringsArray: "TemplateStringsArray",
	WeakMap:              "WeakMap",
	WeakSet:              "WeakSet",

	Iterable: "Iterable",

	Range: "$range",
	Tuple: "$tuple",
}

// symbolNamesValues is Object.values(SYMBOL_NAMES), in declaration order.
var symbolNamesValues = []string{
	SYMBOL_NAMES.GlobalThis,
	SYMBOL_NAMES.ArrayConstructor, SYMBOL_NAMES.SetConstructor, SYMBOL_NAMES.MapConstructor,
	SYMBOL_NAMES.WeakSetConstructor, SYMBOL_NAMES.WeakMapConstructor, SYMBOL_NAMES.ReadonlyMapConstructor,
	SYMBOL_NAMES.ReadonlySetConstructor,
	SYMBOL_NAMES.Array, SYMBOL_NAMES.Generator, SYMBOL_NAMES.IterableFunction, SYMBOL_NAMES.LuaTuple,
	SYMBOL_NAMES.Map, SYMBOL_NAMES.Object, SYMBOL_NAMES.ReadonlyArray, SYMBOL_NAMES.ReadonlyMap,
	SYMBOL_NAMES.ReadonlySet, SYMBOL_NAMES.ReadVoxelsArray, SYMBOL_NAMES.Set, SYMBOL_NAMES.SharedTable,
	SYMBOL_NAMES.String, SYMBOL_NAMES.TemplateStringsArray, SYMBOL_NAMES.WeakMap, SYMBOL_NAMES.WeakSet,
	SYMBOL_NAMES.Iterable,
	SYMBOL_NAMES.Range, SYMBOL_NAMES.Tuple,
}

const NOMINAL_LUA_TUPLE_NAME = "_nominal_LuaTuple"

var MACRO_ONLY_CLASSES = map[string]bool{
	SYMBOL_NAMES.ReadonlyArray: true,
	SYMBOL_NAMES.Array:         true,
	SYMBOL_NAMES.ReadonlyMap:   true,
	SYMBOL_NAMES.WeakMap:       true,
	SYMBOL_NAMES.Map:           true,
	SYMBOL_NAMES.ReadonlySet:   true,
	SYMBOL_NAMES.WeakSet:       true,
	SYMBOL_NAMES.Set:           true,
	SYMBOL_NAMES.String:        true,
}

func projectError(message string) {
	panic(&diagnostics.ProjectError{Message: message})
}

// Deliberate difference from roblox-ts: its MacroManager throws a ProjectError for any macro or symbol
// that the installed @rbxts/types or @rbxts/compiler-types do not declare. Here the lookups return nil
// and the macro is skipped: code can only reach a macro through a declaration, so a macro without one is
// unused. See docs/goblox/PORTING.md, "Type package versions".

func getConstructorInterface(typeChecker *checker.Checker, name string) *ast.Node {
	symbol := getGlobalSymbol(typeChecker, name, ast.SymbolFlagsInterface)
	if symbol == nil {
		return nil
	}
	for _, declaration := range symbol.Declarations {
		if ast.IsInterfaceDeclaration(declaration) {
			return declaration
		}
	}
	return nil
}

func getGlobalSymbol(typeChecker *checker.Checker, name string, meaning ast.SymbolFlags) *ast.Symbol {
	symbol := typeChecker.ResolveName(name, nil, meaning, false)
	if symbol != nil {
		return SkipAliases(typeChecker, symbol)
	}
	return nil
}

func getConstructorSymbol(node *ast.Node) *ast.Symbol {
	for _, member := range node.Members() {
		if ast.IsConstructSignatureDeclaration(member) {
			assert(member.Symbol() != nil)
			return member.Symbol()
		}
	}
	return nil
}

// MacroManager manages the macros of the ts. It is per checker: merged global symbols belong to one
// checker.
type MacroManager struct {
	typeChecker        *checker.Checker
	symbols            map[string]*ast.Symbol
	missingSymbols     map[string]bool
	macroOnlyClasses   map[*ast.Symbol]bool
	identifierMacros   map[*ast.Symbol]IdentifierMacro
	callMacros         map[*ast.Symbol]CallMacro
	constructorMacros  map[*ast.Symbol]ConstructorMacro
	propertyCallMacros map[*ast.Symbol]PropertyCallMacro
}

// NewMacroManager is `new MacroManager(typeChecker)`.
func NewMacroManager(typeChecker *checker.Checker) *MacroManager {
	m := &MacroManager{
		typeChecker:        typeChecker,
		symbols:            map[string]*ast.Symbol{},
		missingSymbols:     map[string]bool{},
		macroOnlyClasses:   map[*ast.Symbol]bool{},
		identifierMacros:   map[*ast.Symbol]IdentifierMacro{},
		callMacros:         map[*ast.Symbol]CallMacro{},
		constructorMacros:  map[*ast.Symbol]ConstructorMacro{},
		propertyCallMacros: map[*ast.Symbol]PropertyCallMacro{},
	}

	for name, macro := range IDENTIFIER_MACROS {
		if symbol := getGlobalSymbol(typeChecker, name, ast.SymbolFlagsVariable); symbol != nil {
			m.identifierMacros[symbol] = macro
		}
	}

	for name, macro := range CALL_MACROS {
		if symbol := getGlobalSymbol(typeChecker, name, ast.SymbolFlagsFunction); symbol != nil {
			m.callMacros[symbol] = macro
		}
	}

	for className, macro := range CONSTRUCTOR_MACROS {
		interfaceDec := getConstructorInterface(typeChecker, className)
		if interfaceDec == nil {
			continue
		}
		if constructSymbol := getConstructorSymbol(interfaceDec); constructSymbol != nil {
			m.constructorMacros[constructSymbol] = macro
		}
	}

	for className, methods := range PROPERTY_CALL_MACROS {
		symbol := getGlobalSymbol(typeChecker, className, ast.SymbolFlagsInterface)
		if symbol == nil {
			continue
		}

		methodMap := map[string]*ast.Symbol{}
		for _, declaration := range symbol.Declarations {
			if ast.IsInterfaceDeclaration(declaration) {
				for _, member := range declaration.Members() {
					if ast.IsMethodSignatureDeclaration(member) && ast.IsIdentifier(member.Name()) {
						symbol := macroManagerGetType(typeChecker, member).Symbol()
						assert(symbol != nil)
						methodMap[member.Name().Text()] = symbol
					}
				}
			}
		}

		for methodName, macro := range methods {
			if methodSymbol := methodMap[methodName]; methodSymbol != nil {
				m.propertyCallMacros[methodSymbol] = macro
			}
		}
	}

	for _, symbolName := range symbolNamesValues {
		symbol := getGlobalSymbol(typeChecker, symbolName, ast.SymbolFlagsAll)
		if symbol == nil {
			m.addMissingSymbol(symbolName)
			continue
		}
		m.symbols[symbolName] = symbol
		if MACRO_ONLY_CLASSES[symbolName] {
			m.macroOnlyClasses[symbol] = true
		}
	}

	var luaTupleTypeDec *ast.Node
	for _, v := range m.GetSymbolOrThrow(SYMBOL_NAMES.LuaTuple).Declarations {
		if ast.IsTypeAliasDeclaration(v) {
			luaTupleTypeDec = v
			break
		}
	}
	if luaTupleTypeDec != nil {
		nominalLuaTupleSymbol := typeGetProperty(typeChecker.GetTypeAtLocation(luaTupleTypeDec), NOMINAL_LUA_TUPLE_NAME)
		if nominalLuaTupleSymbol != nil {
			m.symbols[NOMINAL_LUA_TUPLE_NAME] = nominalLuaTupleSymbol
		}
	}
	if m.symbols[NOMINAL_LUA_TUPLE_NAME] == nil {
		m.addMissingSymbol(NOMINAL_LUA_TUPLE_NAME)
	}

	return m
}

// addMissingSymbol records that the type packages do not declare name. Its symbol is a placeholder that
// no type or declaration has, so the identity checks against it (isArrayType and the like) are false.
func (m *MacroManager) addMissingSymbol(name string) {
	m.symbols[name] = &ast.Symbol{Name: name}
	m.missingSymbols[name] = true
}

// GetSymbolOrThrow returns the placeholder of addMissingSymbol for a symbol the type packages do not
// declare, which is only good for identity checks. Code that reads the symbol uses RequireSymbol.
func (m *MacroManager) GetSymbolOrThrow(name string) *ast.Symbol {
	symbol := m.symbols[name]
	assert(symbol != nil)
	return symbol
}

// RequireSymbol is GetSymbolOrThrow for code that reads the symbol. It throws roblox-ts's ProjectError
// if the type packages do not declare it.
func (m *MacroManager) RequireSymbol(name string) *ast.Symbol {
	if m.missingSymbols[name] {
		projectError("MacroManager could not find symbol for " + name + TYPES_NOTICE)
	}
	return m.GetSymbolOrThrow(name)
}

// HasSymbol reports whether the type packages declare name.
func (m *MacroManager) HasSymbol(name string) bool {
	return !m.missingSymbols[name]
}

func (m *MacroManager) IsMacroOnlyClass(symbol *ast.Symbol) bool {
	return m.macroOnlyClasses[symbol]
}

func (m *MacroManager) GetIdentifierMacro(symbol *ast.Symbol) IdentifierMacro {
	return m.identifierMacros[SkipAliases(m.typeChecker, symbol)]
}

func (m *MacroManager) GetCallMacro(symbol *ast.Symbol) CallMacro {
	return m.callMacros[SkipAliases(m.typeChecker, symbol)]
}

func (m *MacroManager) GetConstructorMacro(symbol *ast.Symbol) ConstructorMacro {
	return m.constructorMacros[symbol]
}

func (m *MacroManager) GetPropertyCallMacro(symbol *ast.Symbol) PropertyCallMacro {
	macro := m.propertyCallMacros[symbol]
	if macro == nil && symbol.Parent != nil {
		// augmented interface members retain their original, unmerged parent symbol
		parent := m.typeChecker.GetMergedSymbol(symbol.Parent)
		if m.IsMacroOnlyClass(parent) {
			assert(false, "Macro "+parent.Name+"."+symbol.Name+"() is not implemented!")
		}
	}
	return macro
}
