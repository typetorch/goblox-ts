package diagnostics

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
)

func itoa(i int) string { return strconv.Itoa(i) }

// ErrorsCatalogue is the type of Errors.
type ErrorsCatalogue struct {
	// reserved identifiers
	NoInvalidIdentifier   *DiagnosticFactory
	NoReservedIdentifier  *DiagnosticFactory
	NoReservedClassFields *DiagnosticFactory
	NoClassMetamethods    *DiagnosticFactory
	NoLengthIndexInTuples *DiagnosticFactory

	// banned statements
	NoForInStatement    *DiagnosticFactory
	NoLabeledStatement  *DiagnosticFactory
	NoDebuggerStatement *DiagnosticFactory

	// banned expressions
	NoNullLiteral       *DiagnosticFactory
	NoPrivateIdentifier *DiagnosticFactory
	NoTypeOfExpression  *DiagnosticFactory
	NoRegex             *DiagnosticFactory
	NoBigInt            *DiagnosticFactory

	// banned features
	NoAny                                     *DiagnosticFactory
	NoVar                                     *DiagnosticFactory
	NoGetterSetter                            *DiagnosticFactory
	NoAutoAccessorModifiers                   *DiagnosticFactory
	NoEqualsEquals                            *DiagnosticFactory
	NoExclamationEquals                       *DiagnosticFactory
	NoEnumMerging                             *DiagnosticFactory
	NoNamespaceMerging                        *DiagnosticFactory
	NoFunctionExpressionName                  *DiagnosticFactory
	NoPrecedingSpreadElement                  *DiagnosticFactory
	NoLuaTupleDestructureAssignmentExpression *DiagnosticFactory
	NoExportAssignmentLet                     *DiagnosticFactory
	NoGlobalThis                              *DiagnosticFactory
	NoArguments                               *DiagnosticFactory
	NoPrototype                               *DiagnosticFactory
	NoRobloxSymbolInstanceof                  *DiagnosticFactory
	NoRestSpreadingOfRobloxTypes              *DiagnosticFactory
	NoNonNumberStringRelationOperator         *DiagnosticFactory
	NoInstanceMethodCollisions                *DiagnosticFactory
	NoStaticMethodCollisions                  *DiagnosticFactory
	NoUnaryPlus                               *DiagnosticFactory
	NoNonNumberUnaryMinus                     *DiagnosticFactory
	NoAwaitForOf                              *DiagnosticFactory
	NoAsyncGeneratorFunctions                 *DiagnosticFactory
	NoNonStringModuleSpecifier                *DiagnosticFactory
	NoIterableIteration                       *DiagnosticFactory
	NoUnsupportedIteration                    *DiagnosticFactory
	NoMixedTypeCall                           *DiagnosticFactory
	NoUnstableThisType                        *DiagnosticFactory
	NoIndexWithoutCall                        *DiagnosticFactory
	NoCommentDirectives                       *DiagnosticFactory

	// macro methods
	NoOptionalMacroCall          *DiagnosticFactory
	NoConstructorMacroWithoutNew *DiagnosticFactory
	NoMacroExtends               *DiagnosticFactory
	NoMacroUnion                 *DiagnosticFactory
	NoMacroObjectSpread          *DiagnosticFactory
	NoVarArgsMacroSpread         *DiagnosticFactory
	NoRangeMacroOutsideForOf     *DiagnosticFactory
	NoTupleMacroOutsideReturn    *DiagnosticFactory

	// import/export
	NoModuleSpecifierFile *DiagnosticFactory
	NoInvalidScope        *DiagnosticFactory
	NoUnscopedModule      *DiagnosticFactory
	// FailedSymlinkResolve context: nodeModulesPath, moduleFilename, virtualPath ("" = undefined), relativePath string
	FailedSymlinkResolve *DiagnosticFactory
	NoNonModuleImport    *DiagnosticFactory
	NoIsolatedImport     *DiagnosticFactory
	NoServerImport       *DiagnosticFactory

	// jsx
	NoPrecedingJsxSpreadElement *DiagnosticFactory

	// semantic
	ExpectedMethodGotFunction *DiagnosticFactory
	ExpectedFunctionGotMethod *DiagnosticFactory

	// files
	// NoRojoData context: path string, isPackage bool
	NoRojoData *DiagnosticFactory
	// NoPackageImportWithoutScope context: path string, rbxPath []string
	NoPackageImportWithoutScope *DiagnosticFactory
	IncorrectFileName           func(originalFileName, suggestedFileName, fullPath string) *ast.Diagnostic
	RojoPathInSrc               func(partitionPath, suggestedPath string) *ast.Diagnostic
	NoMixedStringIndex          *DiagnosticFactory
}

// Errors is `errors`. Ids are assigned in declaration order, as in roblox-ts.
var Errors = ErrorsCatalogue{
	// reserved identifiers
	NoInvalidIdentifier: errorFactory(
		"Invalid Luau identifier!",
		"Luau identifiers must start with a letter and only contain letters, numbers, and underscores.",
		"Reserved Luau keywords cannot be used as identifiers.",
	),
	NoReservedIdentifier:  errorFactory("Cannot use identifier reserved for compiler internal usage."),
	NoReservedClassFields: errorFactory("Cannot use class field reserved for compiler internal usage."),
	NoClassMetamethods:    errorFactory("Metamethods cannot be used in class definitions!"),
	NoLengthIndexInTuples: errorFactory("Cannot index member `length` in a tuple!", suggestion("Use .size() instead.")),

	// banned statements
	NoForInStatement:    errorFactory("for-in loop statements are not supported!"),
	NoLabeledStatement:  errorFactory("labels are not supported!"),
	NoDebuggerStatement: errorFactory("`debugger` is not supported!"),

	// banned expressions
	NoNullLiteral:       errorFactory("`null` is not supported!", suggestion("Use `undefined` instead.")),
	NoPrivateIdentifier: errorFactory("Private identifiers are not supported!"),
	NoTypeOfExpression: errorFactory(
		"`typeof` operator is not supported!",
		suggestion("Use `typeIs(value, type)` or `typeOf(value)` instead."),
	),
	NoRegex:  errorFactory("Regular expressions are not supported!"),
	NoBigInt: errorFactory("BigInt literals are not supported!"),

	// banned features
	NoAny:          errorFactory("Using values of type `any` is not supported!", suggestion("Use `unknown` instead.")),
	NoVar:          errorFactory("`var` keyword is not supported!", suggestion("Use `let` or `const` instead.")),
	NoGetterSetter: errorFactory("Getters and Setters are not supported!", issue(457)),
	NoAutoAccessorModifiers: errorFactory(
		"Getters and Setters are not supported!",
		"The `accessor` keyword requires generating get/set accessors",
		issue(457),
	),
	NoEqualsEquals:           errorFactory("operator `==` is not supported!", suggestion("Use `===` instead.")),
	NoExclamationEquals:      errorFactory("operator `!=` is not supported!", suggestion("Use `!==` instead.")),
	NoEnumMerging:            errorFactory("Enum merging is not supported!"),
	NoNamespaceMerging:       errorFactory("Namespace merging is not supported!"),
	NoFunctionExpressionName: errorFactory("Function expression names are not supported!"),
	NoPrecedingSpreadElement: errorFactory("Spread element must come last in a list of arguments!"),
	NoLuaTupleDestructureAssignmentExpression: errorFactory(
		"Cannot destructure LuaTuple<T> expression outside of an ExpressionStatement!",
	),
	NoExportAssignmentLet: errorFactory("Cannot use `export =` on a `let` variable!", suggestion("Use `const` instead.")),
	NoGlobalThis:          errorFactory("`globalThis` is not supported!"),
	NoArguments:           errorFactory("`arguments` is not supported!"),
	NoPrototype:           errorFactory("`prototype` is not supported!"),
	NoRobloxSymbolInstanceof: errorFactory(
		"The `instanceof` operator can only be used on roblox-ts classes!",
		suggestion(`Use `+"`"+`typeIs(myThing, "TypeToCheck") instead`),
	),
	NoRestSpreadingOfRobloxTypes:      errorFactory("Operator `...` is not allowed on Roblox types!"),
	NoNonNumberStringRelationOperator: errorFactory("Relation operators can only be used on number or string types!"),
	NoInstanceMethodCollisions:        errorFactory("Static methods cannot use the same name as instance methods!"),
	NoStaticMethodCollisions:          errorFactory("Instance methods cannot use the same name as static methods!"),
	NoUnaryPlus:                       errorFactory("Unary `+` is not supported!", suggestion("Use `tonumber(x)` instead.")),
	NoNonNumberUnaryMinus:             errorFactory("Unary `-` is only supported for number types!"),
	NoAwaitForOf:                      errorFactory("`await` is not supported in for-of loops!"),
	NoAsyncGeneratorFunctions:         errorFactory("Async generator functions are not supported!"),
	NoNonStringModuleSpecifier:        errorFactory("Module specifiers must be a string literal."),
	NoIterableIteration:               errorFactory("Iterating on Iterable<T> is not supported! You must use a more specific type."),
	NoUnsupportedIteration: errorFactory(
		"Cannot iterate over this type!",
		"roblox-ts needs a supported, specific iterable type to generate the correct iteration code.",
		suggestion("Use a supported type such as Array<T>, Set<T>, or Map<K, V>."),
	),
	NoMixedTypeCall: errorFactory(
		"Attempted to call a function with mixed types! All definitions must either be a method or a callback.",
	),
	NoUnstableThisType: errorFactory(
		"The generic this type can become void, changing whether a receiver is passed.",
		suggestion("Use this: void for callbacks, or constrain the receiver type to exclude void."),
	),
	NoIndexWithoutCall: errorFactory(
		"Cannot index a method without calling it!",
		suggestion("Use the form `() => a.b()` instead of `a.b`."),
	),
	NoCommentDirectives: errorFactory(
		"Usage of `@ts-ignore`, `@ts-expect-error`, and `@ts-nocheck` are not supported!",
		"roblox-ts needs type and symbol info to compile correctly.",
		suggestion("Consider using type assertions or `declare` statements."),
	),

	// macro methods
	NoOptionalMacroCall: errorFactory(
		"Macro methods can not be optionally called!",
		suggestion("Macros always exist. Use a normal call."),
	),
	NoConstructorMacroWithoutNew: errorFactory("Cannot index a constructor macro without using the `new` operator!"),
	NoMacroExtends: errorFactory(
		"Cannot extend from a macro class!",
		suggestion("Store an instance of the macro class in a property."),
	),
	NoMacroUnion: errorFactory("Macro cannot be applied to a union type!"),
	NoMacroObjectSpread: errorFactory(
		"Macro classes cannot be used in an object spread!",
		suggestion("Did you mean to use an array spread? `[ ...exp ]`"),
	),
	NoVarArgsMacroSpread:      errorFactory("Macros which use variadic arguments do not support spread expressions!", issue(1149)),
	NoRangeMacroOutsideForOf:  errorFactory("$range() macro is only valid as an expression of a for-of loop!"),
	NoTupleMacroOutsideReturn: errorFactory("$tuple() macro is only valid as an expression of a return statement!"),

	// import/export
	NoModuleSpecifierFile: errorFactory("Could not find file for import. Did you forget to `npm install`?"),
	NoInvalidScope:        errorFactory("You can only use npm scopes that are listed in your typeRoots."),
	NoUnscopedModule: errorFactory(
		"Cannot import a module without a scope!",
		"roblox-ts does not support importing regular npm packages.",
	),
	FailedSymlinkResolve: errorWithContext(func(context ...any) []string {
		// path.join and path.relative results use the platform's separators
		nodeModulesPath, moduleFilename, virtualPath, relativePath := filepath.FromSlash(context[0].(string)), context[1].(string), context[2].(string), filepath.FromSlash(context[3].(string))
		virtual := ""
		if virtualPath != "" {
			virtual = "roblox-ts thinks it should be imported through this symlink: " + virtualPath
		}
		return []string{
			"Cannot resolve package entry point inside the project node_modules!",
			"Check the package main field and any package symlinks.",
			"Packages are expected to be installed under " + nodeModulesPath,
			"Package declarations resolved to " + moduleFilename,
			virtual,
			"The path from node_modules to the package would be " + relativePath,
		}
	}),
	NoNonModuleImport: errorFactory("Cannot import a non-ModuleScript!"),
	NoIsolatedImport:  errorFactory("Attempted to import a file inside of an isolated container from outside!"),
	NoServerImport: errorFactory(
		"Cannot import a server file from a shared or client location!",
		suggestion("Move the file you want to import to a shared location."),
	),

	// jsx
	NoPrecedingJsxSpreadElement: errorFactory("JSX spread expression must come last in children!"),

	// semantic
	ExpectedMethodGotFunction: errorFactory("Attempted to assign non-method where method was expected."),
	ExpectedFunctionGotMethod: errorFactory("Attempted to assign method where non-method was expected."),

	// files
	NoRojoData: errorWithContext(func(context ...any) []string {
		// path.relative results use the platform's separators
		path, isPackage := filepath.FromSlash(context[0].(string)), context[1].(bool)
		packageSuggestion := ""
		if isPackage {
			packageSuggestion = suggestion("Did you forget to add a custom npm scope to your default.project.json?")
		}
		return []string{
			"Could not find Rojo data. There is no $path in your Rojo config that covers " + path,
			packageSuggestion,
		}
	}),
	NoPackageImportWithoutScope: errorWithContext(func(context ...any) []string {
		// path.relative results use the platform's separators
		path, rbxPath := filepath.FromSlash(context[0].(string)), context[1].([]string)
		return []string{
			"Imported package Roblox path is missing an npm scope!",
			"Package path: " + path,
			"Roblox path: " + strings.Join(rbxPath, "."),
			suggestion(`You might need to update your "node_modules" in default.project.json to match:
"node_modules": {
	"$className": "Folder",
	"@rbxts": {
		"$path": "node_modules/@rbxts"
	}
}`),
		}
	}),
	IncorrectFileName: func(originalFileName, suggestedFileName, fullPath string) *ast.Diagnostic {
		return errorText(
			"Incorrect file name: `"+originalFileName+"`!",
			"Full path: "+fullPath,
			suggestion("Change `"+originalFileName+"` to `"+suggestedFileName+"`."),
		)
	},
	RojoPathInSrc: func(partitionPath, suggestedPath string) *ast.Diagnostic {
		// path.relative results use the platform's separators
		return errorText(
			"Invalid Rojo configuration. $path fields should be relative to out directory.",
			suggestion(`Change the value of $path from "`+filepath.FromSlash(partitionPath)+`" to "`+filepath.FromSlash(suggestedPath)+`".`),
		)
	},
	NoMixedStringIndex: errorFactory(
		"Cannot index a type containing both strings and other types!",
		suggestion("Narrow the value to a string or a non-string type before indexing."),
	),
}

// WarningsCatalogue is the type of Warnings.
type WarningsCatalogue struct {
	// TruthyChange creates a new factory (and id) on every call, like roblox-ts.
	TruthyChange                    func(checksStr string) *DiagnosticFactory
	TransformerNotFound             func(name string, err any) *ast.Diagnostic
	RuntimeLibUsedInReplicatedFirst *DiagnosticFactory
}

// Warnings is `warnings`.
var Warnings = WarningsCatalogue{
	TruthyChange: func(checksStr string) *DiagnosticFactory {
		return warning("Value will be checked against " + checksStr)
	},
	TransformerNotFound: func(name string, err any) *ast.Diagnostic {
		return warningText(
			"Transformer `"+name+"` was not found!",
			"More info: "+jsString(err),
			suggestion("Did you forget to install the package?"),
		)
	},
	RuntimeLibUsedInReplicatedFirst: warning(
		"This statement would generate a call to the runtime library. The runtime library should not be used from ReplicatedFirst.",
	),
}

func jsString(v any) string {
	switch v := v.(type) {
	case error:
		return "Error: " + v.Error()
	case string:
		return v
	case nil:
		return "undefined"
	}
	return "[object Object]"
}
