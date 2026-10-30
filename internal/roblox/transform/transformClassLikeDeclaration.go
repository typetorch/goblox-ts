package transform

// Port of TSTransformer/nodes/class/transformClassLikeDeclaration.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

const MAGIC_TO_STRING_METHOD = "toString"

func createNameFunction(name string) *luau.FunctionExpression {
	return luau.Create(&luau.FunctionExpression{
		Statements: luau.MakeList(
			luau.Create(&luau.ReturnStatement{
				Expression: luau.String(name),
			}),
		),
		Parameters:   luau.MakeList(),
		HasDotDotDot: false,
	})
}

func createBoilerplate(
	state *TransformState,
	node *ast.Node,
	className luau.AnyIdentifier,
	isClassExpression bool,
) *luau.List {
	isAbstract := ast.HasAbstractModifier(node)
	statements := luau.MakeList()

	/* boilerplate:
	className = setmetatable({}, {
		__tostring = function() return "className" end;
	});
	className.__index = className;
	function className.new(...)
		local self = setmetatable({}, className);
		self:constructor(...);
		return self;
	end;
	function className:constructor()
	end;
	*/

	// 	className = setmetatable({}, {
	// 		__tostring = function() return "className" end;
	// 		__index = super,
	//	});

	// if a class is abstract and it does not extend any class, it can just be a plain table
	// otherwise we can use the default boilerplate
	extendsNode := getExtendsNode(node)
	if isAbstract && extendsNode == nil {
		statements.Push(
			luau.Create(&luau.Assignment{
				Left:     className,
				Operator: "=",
				Right:    luau.NewMap(),
			}),
		)
	} else {
		metatableFields := luau.MakeList()
		toStringName := "Anonymous"
		if id, ok := className.(*luau.Identifier); ok {
			toStringName = id.Name
		}
		metatableFields.Push(
			luau.Create(&luau.MapField{
				Index: luau.Strings.Tostring,
				Value: createNameFunction(toStringName),
			}),
		)

		if extendsNode != nil {
			extendsExpPrereqs := NewPrereqs()
			extendsExp := transformExpression(state, extendsExpPrereqs, extendsNode.Expression())
			superId := luau.Id("super")
			statements.PushList(extendsExpPrereqs.Statements)
			statements.Push(
				luau.Create(&luau.VariableDeclaration{
					Left:  superId,
					Right: extendsExp,
				}),
			)
			metatableFields.Push(
				luau.Create(&luau.MapField{
					Index: luau.Strings.Index,
					Value: superId,
				}),
			)
		}

		metatable := luau.Call(luau.Globals.Setmetatable,
			luau.NewMap(),
			luau.Create(&luau.Map{Fields: metatableFields}),
		)

		if isClassExpression && node.Name() != nil {
			statements.Push(
				luau.Create(&luau.VariableDeclaration{
					Left:  transformIdentifierDefined(state, node.Name()),
					Right: metatable,
				}),
			)
		} else {
			statements.Push(
				luau.Create(&luau.Assignment{
					Left:     className,
					Operator: "=",
					Right:    metatable,
				}),
			)
		}

		//	className.__index = className;
		statements.Push(
			luau.Create(&luau.Assignment{
				Left:     luau.Property(className, "__index"),
				Operator: "=",
				Right:    className,
			}),
		)
	}

	// statements for className.new
	if !isAbstract {
		statementsInner := luau.MakeList()

		//	local self = setmetatable({}, className);
		statementsInner.Push(
			luau.Create(&luau.VariableDeclaration{
				Left:  luau.Globals.Self,
				Right: luau.Call(luau.Globals.Setmetatable, luau.NewMap(), className),
			}),
		)

		//	return self:constructor(...) or self;
		statementsInner.Push(
			luau.Create(&luau.ReturnStatement{
				Expression: luau.Binary(
					luau.Create(&luau.MethodCallExpression{
						Expression: luau.Globals.Self,
						Name:       "constructor",
						Args:       luau.MakeList(luau.Create(&luau.VarArgsLiteral{})),
					}),
					"or",
					luau.Globals.Self,
				),
			}),
		)

		//	function className.new(...)
		//	end;
		statements.Push(
			luau.Create(&luau.FunctionDeclaration{
				Name:         luau.Property(className, "new"),
				Parameters:   luau.MakeList(),
				HasDotDotDot: true,
				Statements:   statementsInner,
				Localize:     false,
			}),
		)
	}

	return statements
}

func isClassHoisted(state *TransformState, node *ast.Node) bool {
	if node.Name() != nil {
		symbol := state.TypeChecker.GetSymbolAtLocation(node.Name())
		assert(symbol != nil)
		return state.IsHoisted[symbol]
	}
	return false
}

// transformClassLikeDeclaration is TSTransformer/nodes/class/transformClassLikeDeclaration.ts transformClassLikeDeclaration.
func transformClassLikeDeclaration(state *TransformState, node *ast.Node) (statements *luau.List, name luau.AnyIdentifier) {
	isClassExpression := ast.IsClassExpression(node)
	statements = luau.MakeList()

	isExportDefault := ast.HasSyntacticModifier(node, ast.ModifierFlagsExportDefault)

	if node.Name() != nil {
		validateIdentifier(node.Name())
	}

	/*
		local className;
		do
			OOP boilerplate
			class functions
		end
	*/

	shouldUseInternalName := isClassExpression && node.Name() != nil

	var returnVar luau.AnyIdentifier
	if shouldUseInternalName {
		returnVar = luau.TempId("class")
	} else if node.Name() != nil {
		returnVar = transformIdentifierDefined(state, node.Name())
	} else if isExportDefault {
		returnVar = luau.Id("default")
	} else {
		returnVar = luau.TempId("class")
	}

	var internalName luau.AnyIdentifier
	if shouldUseInternalName {
		internalName = transformIdentifierDefined(state, node.Name())
	} else {
		internalName = returnVar
	}
	state.ClassIdentifierMap[node] = internalName
	if !isClassHoisted(state, node) {
		statements.Push(
			luau.Create(&luau.VariableDeclaration{
				Left:  returnVar,
				Right: nil,
			}),
		)
	}

	// OOP boilerplate + class functions
	statementsInner := luau.MakeList()
	statementsInner.PushList(createBoilerplate(state, node, internalName, isClassExpression))

	if constructor := findConstructor(node); constructor != nil {
		statementsInner.PushList(transformClassConstructor(state, constructor, internalName))
	} else {
		statementsInner.PushList(transformImplicitClassConstructor(state, node, internalName))
	}

	for _, member := range node.Members() {
		if (ast.IsPropertyDeclaration(member) || ast.IsMethodDeclaration(member)) &&
			(ast.IsIdentifier(member.Name()) || ast.IsStringLiteral(member.Name())) &&
			luau.IsReservedClassField(member.Name().Text()) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoReservedClassFields.Create(member.Name()))
		}
		if ast.IsAutoAccessorPropertyDeclaration(member) {
			// member must have AccessorKeyword to be AutoAccessorPropertyDeclaration
			var keyword *ast.Node
			for _, m := range member.Modifiers().Nodes {
				if m.Kind == ast.KindAccessorKeyword {
					keyword = m
					break
				}
			}
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoAutoAccessorModifiers.Create(keyword))
		}
	}

	var methods []*ast.Node
	var staticDeclarations []*ast.Node

	for _, member := range node.Members() {
		validateMethodAssignment(state, member)
		if ast.IsConstructorDeclaration(member) ||
			ast.IsIndexSignatureDeclaration(member) ||
			ast.IsSemicolonClassElement(member) {
			continue
		} else if ast.IsMethodDeclaration(member) {
			methods = append(methods, member)
		} else if ast.IsPropertyDeclaration(member) {
			// non-static properties are done in transformClassConstructor
			if !ast.HasStaticModifier(member) {
				continue
			}
			staticDeclarations = append(staticDeclarations, member)
		} else if ast.IsAccessor(member) {
			DiagnosticService.AddDiagnostic(diagnostics.Errors.NoGetterSetter.Create(member))
		} else {
			assert(
				ast.IsClassStaticBlockDeclaration(member),
				"ClassMember kind not implemented: "+getKindName(member.Kind),
			)
			staticDeclarations = append(staticDeclarations, member)
		}
	}

	classType := state.TypeChecker.GetTypeOfSymbolAtLocation(node.Symbol(), node)
	instanceType := state.TypeChecker.GetDeclaredTypeOfSymbol(node.Symbol())

	for _, method := range methods {
		if ast.IsIdentifier(method.Name()) || ast.IsStringLiteral(method.Name()) {
			if luau.IsMetamethod(method.Name().Text()) {
				DiagnosticService.AddDiagnostic(diagnostics.Errors.NoClassMetamethods.Create(method.Name()))
			}

			if ast.HasStaticModifier(method) {
				if typeGetProperty(instanceType, method.Name().Text()) != nil {
					DiagnosticService.AddDiagnostic(diagnostics.Errors.NoInstanceMethodCollisions.Create(method))
				}
			} else {
				if typeGetProperty(classType, method.Name().Text()) != nil {
					DiagnosticService.AddDiagnostic(diagnostics.Errors.NoStaticMethodCollisions.Create(method))
				}
			}
		}

		methodPrereqs := NewPrereqs()
		methodStatements := transformMethodDeclaration(state, methodPrereqs, method, &Pointer{
			Name:  "name",
			Value: internalName,
		})
		statementsInner.PushList(methodPrereqs.Statements)
		statementsInner.PushList(methodStatements)
	}

	toStringProperty := typeGetProperty(instanceType, MAGIC_TO_STRING_METHOD)
	if toStringProperty != nil && toStringProperty.Flags&ast.SymbolFlagsMethod != 0 {
		statementsInner.Push(
			luau.Create(&luau.MethodDeclaration{
				Expression:   internalName,
				Name:         "__tostring",
				HasDotDotDot: false,
				Parameters:   luau.MakeList(),
				Statements: luau.MakeList(
					luau.Create(&luau.ReturnStatement{
						Expression: luau.Create(&luau.MethodCallExpression{
							Expression: luau.Globals.Self,
							Name:       MAGIC_TO_STRING_METHOD,
							Args:       luau.MakeList(),
						}),
					}),
				),
			}),
		)
	}

	for _, declaration := range staticDeclarations {
		if ast.IsClassStaticBlockDeclaration(declaration) {
			statementsInner.PushList(transformBlock(state, declaration.AsClassStaticBlockDeclaration().Body))
		} else {
			propertyPrereqs := NewPrereqs()
			propertyStatements := transformPropertyDeclaration(state, propertyPrereqs, declaration, internalName)
			statementsInner.PushList(propertyPrereqs.Statements)
			statementsInner.PushList(propertyStatements)
		}
	}

	// if using internal name, assign to return var
	if shouldUseInternalName {
		statementsInner.Push(
			luau.Create(&luau.Assignment{
				Left:     returnVar,
				Operator: "=",
				Right:    internalName,
			}),
		)
	}

	statementsInner.PushList(transformDecorators(state, node, returnVar))

	statements.Push(
		luau.Create(&luau.DoStatement{
			Statements: statementsInner,
		}),
	)

	return statements, returnVar
}
