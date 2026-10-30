package transform

import (
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
	"github.com/microsoft/typescript-go/internal/roblox/nodepath"
	"github.com/microsoft/typescript-go/internal/roblox/rojo"
	"github.com/microsoft/typescript-go/internal/scanner"
	tsdiagnostics "github.com/microsoft/typescript-go/internal/diagnostics"
)

// ProjectType is Shared/constants.ts ProjectType.
type ProjectType string

const (
	ProjectTypeGame    ProjectType = "game"
	ProjectTypeModel   ProjectType = "model"
	ProjectTypePackage ProjectType = "package"
)

// ProjectOptions is Shared/types.ts ProjectOptions. Type is "" when undefined.
type ProjectOptions struct {
	IncludePath            string
	Rojo                   string
	Type                   ProjectType
	LogTruthyChanges       bool
	NoInclude              bool
	NoCompilerHeader       bool
	UsePolling             bool
	Verbose                bool
	Watch                  bool
	WriteOnlyChanged       bool
	WriteTransformedFiles  bool
	OptimizedLoops         bool
	AllowCommentDirectives bool
	Luau                   bool
}

// DEFAULT_PROJECT_OPTIONS is Shared/constants.ts DEFAULT_PROJECT_OPTIONS.
var DEFAULT_PROJECT_OPTIONS = ProjectOptions{
	OptimizedLoops: true,
	Luau:           true,
}

// ProjectData is Shared/types.ts ProjectData. RojoConfigPath is "" when undefined.
type ProjectData struct {
	IsPackage       bool
	NodeModulesPath string
	ProjectOptions  ProjectOptions
	ProjectPath     string
	RojoConfigPath  string
	TsConfigPath    string
	// ProjectReferencePaths maps the projectPathKey of every source and declaration file of the
	// referenced projects to its import path in the referenced project's output (ProjectGraph.ts
	// mapReferencePaths).
	ProjectReferencePaths map[string]string
	// TransformersRan is whether transformer plugins ran for this build. rbxtsc then prints every file
	// again and compiles the reparsed text, whether the plugin changed the file or not. goblox doesn't
	// reprint a file no plugin changed, so where that print is not the identity the transform has to
	// do what it does: numeric separators and comments before the end of the file go away. The other
	// difference, reindented block comments, is left to a real reprint (project.reprintKeepsBlockComments).
	TransformersRan bool
}

func (state *TransformState) transformersRan() bool {
	return state.Data != nil && state.Data.TransformersRan
}

// TransformServices is TSTransformer/types.ts TransformServices.
type TransformServices struct {
	MacroManager *MacroManager
}

// TryUses is TSTransformer/types.ts TryUses.
type TryUses struct {
	UsesReturn   bool
	UsesBreak    bool
	UsesContinue bool
}

// MultiTransformState is state that lives only for a single compilation step. goblox creates one per
// checker: its caches are keyed by that checker's types and symbols, and the files of a checker are
// transformed sequentially by one goroutine, so it needs no locking.
type MultiTransformState struct {
	// type classifications stay valid for this compilation; emitted AST facts can still change
	IsPrimitiveTypeCache                 map[*checker.Type]bool
	IsRobloxTypeCache                    map[*checker.Type]bool
	FunctionEffects                      map[*ast.Symbol]*EvaluationEffects
	BindingKeys                          map[*ast.Symbol]*BindingKey
	BindingUsageBySourceFile             map[*ast.SourceFile]*BindingUsage
	IsMethodCache                        map[*checker.Type]bool
	IsReportedByNoUnstableThisType       map[*ast.Node]bool
	IsDefinedAsLetCache                  map[*ast.Symbol]bool
	IsAnyOrAnyArrayCache                 map[*checker.Type]bool
	IsReportedByNoAnyCache               map[*ast.Symbol]bool
	IsReportedByMultipleDefinitionsCache map[*ast.Symbol]bool
	GetModuleExportsCache                map[*ast.Symbol][]*ast.Symbol
	GetModuleExportsAliasMapCache        map[*ast.Symbol]map[*ast.Symbol]luau.Expression
}

// NewMultiTransformState is `new MultiTransformState()`.
func NewMultiTransformState() *MultiTransformState {
	return &MultiTransformState{
		IsPrimitiveTypeCache:                 map[*checker.Type]bool{},
		IsRobloxTypeCache:                    map[*checker.Type]bool{},
		FunctionEffects:                      map[*ast.Symbol]*EvaluationEffects{},
		BindingKeys:                          map[*ast.Symbol]*BindingKey{},
		BindingUsageBySourceFile:             map[*ast.SourceFile]*BindingUsage{},
		IsMethodCache:                        map[*checker.Type]bool{},
		IsReportedByNoUnstableThisType:       map[*ast.Node]bool{},
		IsDefinedAsLetCache:                  map[*ast.Symbol]bool{},
		IsAnyOrAnyArrayCache:                 map[*checker.Type]bool{},
		IsReportedByNoAnyCache:               map[*ast.Symbol]bool{},
		IsReportedByMultipleDefinitionsCache: map[*ast.Symbol]bool{},
		GetModuleExportsCache:                map[*ast.Symbol][]*ast.Symbol{},
		GetModuleExportsAliasMapCache:        map[*ast.Symbol]map[*ast.Symbol]luau.Expression{},
	}
}

// getOrSetDefault is Shared/util/getOrSetDefault.ts.
func getOrSetDefault[K comparable, V any](m map[K]V, key K, getDefaultValue func() V) V {
	if value, ok := m[key]; ok {
		return value
	}
	value := getDefaultValue()
	m[key] = value
	return value
}

// Prereqs collects the statements an expression needs to run before it. Each collector owns one
// destination for generated statements.
type Prereqs struct {
	Statements *luau.List
}

// NewPrereqs is `new Prereqs()`.
func NewPrereqs() *Prereqs { return &Prereqs{Statements: luau.MakeList()} }

func (p *Prereqs) Push(statement luau.Statement)    { p.Statements.Push(statement) }
func (p *Prereqs) PushList(statements *luau.List)    { p.Statements.PushList(statements) }
func (p *Prereqs) Unshift(statement luau.Statement) { p.Statements.Unshift(statement) }
func (p *Prereqs) UnshiftList(statements *luau.List) { p.Statements.UnshiftList(statements) }

// PushToVar declares a temporary for expression (which may be nil) and returns it.
func (p *Prereqs) PushToVar(expression luau.Expression, name string) *luau.TemporaryIdentifier {
	if name == "" && expression != nil {
		name = valueToIdStr(expression)
	}
	temp := luau.TempId(name)
	p.Push(luau.Create(&luau.VariableDeclaration{Left: temp, Right: listOrNode(expression)}))
	return temp
}

// PushToVarIfComplex returns expression if it is simple, otherwise a temporary holding it.
func (p *Prereqs) PushToVarIfComplex(expression luau.Expression, name string) luau.Expression {
	if luau.IsSimple(expression) {
		return expression
	}
	return p.PushToVar(expression, name)
}

// PushToVarIfNonId returns expression if it is an identifier, otherwise a temporary holding it.
func (p *Prereqs) PushToVarIfNonId(expression luau.Expression, name string) luau.AnyIdentifier {
	if luau.IsAnyIdentifier(expression) {
		return expression
	}
	return p.PushToVar(expression, name)
}

// listOrNode converts a possibly nil Node to a ListOrNode without creating a typed nil.
func listOrNode(n luau.Node) luau.ListOrNode {
	if n == nil {
		return nil
	}
	return n
}

// DiagnosticService collects the diagnostics of a compilation. Like roblox-ts's static class it is
// global; it is safe for concurrent use.
//
// goblox transforms files in parallel, one goroutine per checker, while roblox-ts transforms them in
// order and stops transforming once any file has an error. To report the same set of diagnostics, the
// service keeps each diagnostic that has a file in that file's bucket, and the driver replays the
// buckets in file order (TakeFileDiagnostics). Diagnostics without a file are kept globally.
var DiagnosticService = &diagnosticService{
	singleDiagnostics: map[int32]bool{},
	byFile:            map[*ast.SourceFile][]*ast.Diagnostic{},
}

type diagnosticService struct {
	mu                sync.Mutex
	diagnostics       []*ast.Diagnostic
	byFile            map[*ast.SourceFile][]*ast.Diagnostic
	singleDiagnostics map[int32]bool
}

// add must be called with s.mu held.
func (s *diagnosticService) add(d *ast.Diagnostic) {
	if file := d.File(); file != nil {
		s.byFile[file] = append(s.byFile[file], d)
	} else {
		s.diagnostics = append(s.diagnostics, d)
	}
}

func (s *diagnosticService) AddSingleDiagnostic(d *ast.Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.singleDiagnostics[d.Code()] {
		s.singleDiagnostics[d.Code()] = true
		s.add(d)
	}
}

func (s *diagnosticService) AddDiagnostic(d *ast.Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.add(d)
}

func (s *diagnosticService) AddDiagnostics(ds []*ast.Diagnostic) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range ds {
		s.add(d)
	}
}

// AddDiagnosticWithCache adds d unless cacheBy was already reported.
func AddDiagnosticWithCache[T comparable](cacheBy T, d *ast.Diagnostic, cache map[T]bool) {
	DiagnosticService.mu.Lock()
	defer DiagnosticService.mu.Unlock()
	if !cache[cacheBy] {
		cache[cacheBy] = true
		DiagnosticService.add(d)
	}
}

// TakeFileDiagnostics removes and returns the diagnostics of file, in the order they were added.
func (s *diagnosticService) TakeFileDiagnostics(file *ast.SourceFile) []*ast.Diagnostic {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.byFile[file]
	delete(s.byFile, file)
	return current
}

// Flush removes and returns every diagnostic: the ones without a file, then each file's, by file name.
func (s *diagnosticService) Flush() []*ast.Diagnostic {
	s.mu.Lock()
	defer s.mu.Unlock()
	current := s.diagnostics
	files := slices.SortedFunc(maps.Keys(s.byFile), func(a, b *ast.SourceFile) int {
		return strings.Compare(a.FileName(), b.FileName())
	})
	for _, file := range files {
		current = append(current, s.byFile[file]...)
	}
	s.diagnostics = nil
	clear(s.byFile)
	clear(s.singleDiagnostics)
	return current
}

func (s *diagnosticService) HasErrors() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if hasErrors(s.diagnostics) {
		return true
	}
	for _, ds := range s.byFile {
		if hasErrors(ds) {
			return true
		}
	}
	return false
}

// hasErrors is Shared/util/hasErrors.ts.
func hasErrors(ds []*ast.Diagnostic) bool {
	return slices.ContainsFunc(ds, func(d *ast.Diagnostic) bool { return d.Category() == tsdiagnostics.CategoryError })
}

// TransformState is the state of the transformation of one source file from TS to Luau AST.
type TransformState struct {
	sourceFileText  string
	HasExportEquals bool
	HasExportFrom   bool

	ClassIdentifierMap map[*ast.Node]luau.AnyIdentifier

	Program                *compiler.Program
	Data                   *ProjectData
	Services               *TransformServices
	PathTranslator         *rojo.PathTranslator
	MultiTransformState    *MultiTransformState
	CompilerOptions        *core.CompilerOptions
	RojoResolver           *rojo.RojoResolver
	PkgRojoResolvers       []*rojo.RojoResolver
	NodeModulesPathMapping map[string]string
	RuntimeLibRbxPath      rojo.RbxPath
	TypeChecker            *checker.Checker
	ProjectType            ProjectType
	SourceFile             *ast.SourceFile

	isInReplicatedFirst bool
	tryUsesStack        []*TryUses

	HoistsByStatement map[*ast.Node][]*ast.Node
	// IsHoisted is tri-state like the TypeScript Map: absent, false or true.
	IsHoisted map[*ast.Symbol]bool

	getTypeCache map[*ast.Node]*checker.Type

	UsesRuntimeLib bool

	moduleIdBySymbol           map[*ast.Symbol]luau.AnyIdentifier
	SymbolToIdMap              map[*ast.Symbol]*luau.TemporaryIdentifier
	classElementToObjectKeyMap map[*ast.Node]luau.Expression
}

// NewTransformState is `new TransformState(...)`.
func NewTransformState(
	program *compiler.Program,
	data *ProjectData,
	services *TransformServices,
	pathTranslator *rojo.PathTranslator,
	multiTransformState *MultiTransformState,
	compilerOptions *core.CompilerOptions,
	rojoResolver *rojo.RojoResolver,
	pkgRojoResolvers []*rojo.RojoResolver,
	nodeModulesPathMapping map[string]string,
	runtimeLibRbxPath rojo.RbxPath,
	typeChecker *checker.Checker,
	projectType ProjectType,
	sourceFile *ast.SourceFile,
) *TransformState {
	state := &TransformState{
		Program:                    program,
		Data:                       data,
		Services:                   services,
		PathTranslator:             pathTranslator,
		MultiTransformState:        multiTransformState,
		CompilerOptions:            compilerOptions,
		RojoResolver:               rojoResolver,
		PkgRojoResolvers:           pkgRojoResolvers,
		NodeModulesPathMapping:     nodeModulesPathMapping,
		RuntimeLibRbxPath:          runtimeLibRbxPath,
		TypeChecker:                typeChecker,
		ProjectType:                projectType,
		SourceFile:                 sourceFile,
		sourceFileText:             sourceFile.Text(),
		ClassIdentifierMap:         map[*ast.Node]luau.AnyIdentifier{},
		HoistsByStatement:          map[*ast.Node][]*ast.Node{},
		IsHoisted:                  map[*ast.Symbol]bool{},
		getTypeCache:               map[*ast.Node]*checker.Type{},
		moduleIdBySymbol:           map[*ast.Symbol]luau.AnyIdentifier{},
		SymbolToIdMap:              map[*ast.Symbol]*luau.TemporaryIdentifier{},
		classElementToObjectKeyMap: map[*ast.Node]luau.Expression{},
	}
	sourceOutPath := pathTranslator.GetOutputPath(sourceFile.FileName())
	rbxPath := rojoResolver.GetRbxPathFromFilePath(sourceOutPath)
	state.isInReplicatedFirst = len(rbxPath) > 0 && rbxPath[0] == "ReplicatedFirst"
	return state
}

// DebugRender renders a node, for debugging.
func (s *TransformState) DebugRender(node luau.Node) string {
	renderState := luau.NewRenderState()
	luau.SolveTempIds(renderState, node)
	return luau.Render(renderState, node)
}

// PushTryUsesStack pushes tryUses information onto the tryUses stack and returns it.
func (s *TransformState) PushTryUsesStack() *TryUses {
	tryUses := &TryUses{}
	s.tryUsesStack = append(s.tryUsesStack, tryUses)
	return tryUses
}

// MarkTryUses marks the current try statement as exiting with return, break, or continue statements.
func (s *TransformState) MarkTryUses(property string) {
	if len(s.tryUsesStack) != 0 {
		top := s.tryUsesStack[len(s.tryUsesStack)-1]
		switch property {
		case "usesReturn":
			top.UsesReturn = true
		case "usesBreak":
			top.UsesBreak = true
		case "usesContinue":
			top.UsesContinue = true
		}
	}
}

// PopTryUsesStack pops tryUses information from the tryUses stack.
func (s *TransformState) PopTryUsesStack() {
	s.tryUsesStack = s.tryUsesStack[:len(s.tryUsesStack)-1]
}

// GetLeadingComments returns the leading comments of a node as Luau comments.
func (s *TransformState) GetLeadingComments(node *ast.Node) *luau.List {
	result := luau.MakeList()
	factory := &ast.NodeFactory{}
	for commentRange := range scanner.GetLeadingCommentRanges(factory, s.sourceFileText, node.Pos()) {
		source := s.sourceFileText[commentRange.Pos():commentRange.End()]
		var text string
		if commentRange.Kind == ast.KindSingleLineCommentTrivia {
			text = source[2:]
		} else {
			text = getBlockCommentText(source)
		}
		result.Push(luau.NewComment(text))
	}
	return result
}

// GetType is the (cached) type of a node, looked up on its outermost parenthesized/asserted form.
func (s *TransformState) GetType(node *ast.Node) *checker.Type {
	return getOrSetDefault(s.getTypeCache, node, func() *checker.Type {
		return s.TypeChecker.GetTypeAtLocation(skipUpwards(node))
	})
}

// TS returns `TS.<name>` and marks the file as using the runtime library.
func (s *TransformState) TS(node *ast.Node, name string) luau.IndexableExpression {
	s.UsesRuntimeLib = true

	if s.ProjectType == ProjectTypeGame && s.isInReplicatedFirst {
		DiagnosticService.AddDiagnostic(diagnostics.Warnings.RuntimeLibUsedInReplicatedFirst.Create(node))
	}

	return luau.Property(luau.Globals.TS, name)
}

// CreateRuntimeLibImport returns the `local TS = ...` declaration for RuntimeLib.lua.
func (s *TransformState) CreateRuntimeLibImport(sourceFile *ast.SourceFile) luau.Statement {
	// if the transform state has the game path to the RuntimeLib.lua
	if s.RuntimeLibRbxPath != nil {
		if s.ProjectType == ProjectTypeGame {
			// create an expression to obtain the service where RuntimeLib is stored
			serviceName := s.RuntimeLibRbxPath[0]
			assert(serviceName != "")

			var expression luau.IndexableExpression = createGetService(serviceName)
			// iterate through the rest of the path
			// for each instance in the path, create a new WaitForChild call to be added on to the end of the final expression
			for i := 1; i < len(s.RuntimeLibRbxPath); i++ {
				expression = luau.Create(&luau.MethodCallExpression{
					Expression: expression,
					Name:       "WaitForChild",
					Args:       luau.MakeList(luau.String(s.RuntimeLibRbxPath[i])),
				})
			}

			// nest the chain of `WaitForChild`s inside a require call
			expression = luau.Call(luau.Globals.Require, expression)

			// create a variable declaration for this call
			return luau.Create(&luau.VariableDeclaration{Left: luau.Globals.TS, Right: expression})
		}
		sourceOutPath := s.PathTranslator.GetOutputPath(sourceFile.FileName())
		rbxPath := s.RojoResolver.GetRbxPathFromFilePath(sourceOutPath)
		if rbxPath == nil {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoRojoData.CreateWithRange(
				sourceFile, sourceFile.Loc, nodepath.Relative(s.Data.ProjectPath, sourceOutPath), false))
			return luau.Create(&luau.VariableDeclaration{Left: luau.Globals.TS, Right: luau.NewNone()})
		}

		var names []string
		for _, v := range rojo.Relative(rbxPath, s.RuntimeLibRbxPath) {
			if v.IsParent {
				names = append(names, PARENT_FIELD)
			} else {
				names = append(names, v.Name)
			}
		}
		return luau.Create(&luau.VariableDeclaration{
			Left:  luau.Globals.TS,
			Right: luau.Call(luau.Globals.Require, propertyAccessExpressionChain(luau.Globals.Script, names)),
		})
	}
	// we pass RuntimeLib access to packages via `_G[script] = TS`
	// access it here via `local TS = _G[script]`
	return luau.Create(&luau.VariableDeclaration{
		Left: luau.Globals.TS,
		Right: luau.Create(&luau.ComputedIndexExpression{
			Expression: luau.Globals.G,
			Index:      luau.Globals.Script,
		}),
	})
}

// GetModuleExports returns the value exports of a module.
func (s *TransformState) GetModuleExports(moduleSymbol *ast.Symbol) []*ast.Symbol {
	return getOrSetDefault(s.MultiTransformState.GetModuleExportsCache, moduleSymbol, func() []*ast.Symbol {
		var result []*ast.Symbol
		for _, exportSymbol := range s.orderModuleExports(moduleSymbol, s.TypeChecker.GetExportsOfModule(moduleSymbol)) {
			typeOnlyDeclaration := s.TypeChecker.GetTypeOnlyAliasDeclaration(exportSymbol)
			// erased aliases must not redirect mutable locals to properties on the exports table
			if typeOnlyDeclaration == nil ||
				(s.CompilerOptions.VerbatimModuleSyntax.IsTrue() && !isTypeOnlyExportDeclaration(typeOnlyDeclaration)) {
				result = append(result, exportSymbol)
			}
		}
		return result
	})
}

// GetModuleExportsAliasMap maps the original symbol of each export of a module to its export key.
func (s *TransformState) GetModuleExportsAliasMap(moduleSymbol *ast.Symbol) map[*ast.Symbol]luau.Expression {
	if cached, ok := s.MultiTransformState.GetModuleExportsAliasMapCache[moduleSymbol]; ok {
		return cached
	}
	aliasMap := map[*ast.Symbol]luau.Expression{}
	for _, exportSymbol := range s.GetModuleExports(moduleSymbol) {
		originalSymbol := s.TypeChecker.SkipAlias(exportSymbol)
		var declaration *ast.Node
		if len(exportSymbol.Declarations) > 0 {
			declaration = exportSymbol.Declarations[0]
		}
		if declaration != nil && ast.IsExportSpecifier(declaration) {
			namePrereqs := NewPrereqs()
			name := transformPropertyName(s, namePrereqs, declaration.Name())
			assert(namePrereqs.Statements.IsEmpty())
			aliasMap[originalSymbol] = name
		} else {
			aliasMap[originalSymbol] = luau.String(exportSymbol.Name)
		}
	}
	s.MultiTransformState.GetModuleExportsAliasMapCache[moduleSymbol] = aliasMap
	return aliasMap
}

func (s *TransformState) getModuleSymbolFromNode(node *ast.Node) *ast.Symbol {
	moduleAncestor := getModuleAncestor(node)
	location := moduleAncestor
	if !ast.IsSourceFile(moduleAncestor) {
		location = moduleAncestor.Name()
	}
	exportSymbol := s.TypeChecker.GetSymbolAtLocation(location)
	assert(exportSymbol != nil)
	return exportSymbol
}

func (s *TransformState) getModuleIdFromSymbol(moduleSymbol *ast.Symbol) luau.AnyIdentifier {
	moduleId, ok := s.moduleIdBySymbol[moduleSymbol]
	assert(ok)
	return moduleId
}

// SetModuleIdBySymbol records the Luau identifier of a module (exports or a namespace table).
func (s *TransformState) SetModuleIdBySymbol(moduleSymbol *ast.Symbol, moduleId luau.AnyIdentifier) {
	s.moduleIdBySymbol[moduleSymbol] = moduleId
}

// GetModuleIdFromNode returns the Luau identifier of the module containing node.
func (s *TransformState) GetModuleIdFromNode(node *ast.Node) luau.AnyIdentifier {
	return s.getModuleIdFromSymbol(s.getModuleSymbolFromNode(node))
}

// GetModuleIdPropertyAccess returns `exports.<name>` for an exported symbol, or nil.
func (s *TransformState) GetModuleIdPropertyAccess(idSymbol *ast.Symbol) luau.Expression {
	if idSymbol.ValueDeclaration != nil {
		moduleSymbol := s.getModuleSymbolFromNode(idSymbol.ValueDeclaration)
		if alias, ok := s.GetModuleExportsAliasMap(moduleSymbol)[idSymbol]; ok {
			return luau.Create(&luau.ComputedIndexExpression{
				Expression: s.getModuleIdFromSymbol(moduleSymbol),
				Index:      alias,
			})
		}
	}
	return nil
}

// SetClassElementObjectKey stores the key of `obj[key] = value` for a class element.
func (s *TransformState) SetClassElementObjectKey(classElement *ast.Node, identifier luau.Expression) {
	_, exists := s.classElementToObjectKeyMap[classElement]
	assert(!exists)
	s.classElementToObjectKeyMap[classElement] = identifier
}

// GetClassElementObjectKey is the key stored by SetClassElementObjectKey, or nil.
func (s *TransformState) GetClassElementObjectKey(classElement *ast.Node) luau.Expression {
	return s.classElementToObjectKeyMap[classElement]
}

// assert is Shared/util/assert.ts.
func assert(value bool, message ...string) {
	if !value {
		panic("Assertion Failed! " + strings.Join(message, "") +
			"\nThis is a compiler bug! Please submit a bug report here:" +
			"\nhttps://github.com/roblox-ts/roblox-ts/issues")
	}
}

// isTypeOnlyExportDeclaration is TypeScript's isTypeOnlyExportDeclaration (unexported in typescript-go).
func isTypeOnlyExportDeclaration(node *ast.Node) bool {
	switch node.Kind {
	case ast.KindExportSpecifier:
		return node.IsTypeOnly() || node.Parent.Parent.IsTypeOnly()
	case ast.KindExportDeclaration:
		d := node.AsExportDeclaration()
		return d.IsTypeOnly && d.ModuleSpecifier != nil && d.ExportClause == nil
	case ast.KindNamespaceExport:
		return node.Parent.IsTypeOnly()
	}
	return false
}
