package transform

// Port of TSTransformer/nodes/class/transformDecorators.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

func countDecorators(node *ast.Node) int {
	return len(node.Decorators())
}

func shouldInline(state *TransformState, isLastDecorator bool, decorator *ast.Node, expression luau.Expression) bool {
	// immutable expressions can be inlined
	if !expressionMightMutate(state, expression, decorator.Expression()) {
		return true
	}

	// if it's not the last decorator, we can't inline
	// this is because we need to initialize all decorators before running them
	if !isLastDecorator {
		return false
	}

	node := decorator.Parent

	// if the node is a method declaration and has a decorator on a parameter, we can't inline
	// this is because we need to run parameter decorators between initializing and running method decorators
	if ast.IsMethodDeclaration(node) {
		for _, parameter := range node.Parameters() {
			if countDecorators(parameter) > 0 {
				return false
			}
		}
	}

	// if the node is a class declaration and has a decorator on a constructor parameter, we can't inline
	// this is because we need to run constructor parameter decorators between initializing and running class decorators
	if ast.IsClassLike(node) {
		if constructor := findConstructor(node); constructor != nil {
			for _, parameter := range constructor.Parameters() {
				if countDecorators(parameter) > 0 {
					return false
				}
			}
		}
	}

	// if the node is a parameter and there are any parameters with decorators after it, we can't inline
	// this ensures all of the parameters are initialized before running any, including from sibling parameters
	if ast.IsParameterDeclaration(node) {
		parameters := node.Parent.Parameters()
		paramIdx := -1
		for i, param := range parameters {
			if param == node {
				paramIdx = i
				break
			}
		}
		for i := paramIdx + 1; i < len(parameters); i++ {
			if countDecorators(parameters[i]) > 0 {
				return false
			}
		}
	}

	return true
}

func transformMemberDecorators(
	state *TransformState,
	node *ast.Node,
	callback func(expression luau.IndexableExpression) *luau.List,
) (initializers *luau.List, finalizers *luau.List) {
	initializers = luau.MakeList()
	finalizers = luau.MakeList()

	decorators := node.Decorators()

	for i, decorator := range decorators {
		expressionPrereqs := NewPrereqs()
		expression := transformExpression(state, expressionPrereqs, decorator.Expression())

		initializers.PushList(expressionPrereqs.Statements)

		isLastDecorator := i == len(decorators)-1

		if !shouldInline(state, isLastDecorator, decorator, expression) {
			tempId := luau.TempId("decorator")
			initializers.Push(
				luau.Create(&luau.VariableDeclaration{
					Left:  tempId,
					Right: expression,
				}),
			)
			expression = tempId
		}

		finalizers.UnshiftList(callback(convertToIndexableExpression(expression)))
	}

	return initializers, finalizers
}

func transformMethodDecorators(state *TransformState, member *ast.Node, classId luau.AnyIdentifier) *luau.List {
	initializers, finalizers := transformMemberDecorators(state, member, func(expression luau.IndexableExpression) *luau.List {
		result := luau.MakeList()

		// local _descriptor = decorator(Class, "name", { value = Class.name })
		// if _descriptor then
		// 	Class.name = _descriptor.value
		// end

		descriptorId := luau.TempId("descriptor")
		key := state.GetClassElementObjectKey(member)
		assert(key != nil, "Did not find method key for method decorator")

		result.Push(
			luau.Create(&luau.VariableDeclaration{
				Left: descriptorId,
				Right: luau.Call(expression,
					classId,
					key,
					luau.NewMap(luau.MapEntry{
						Index: luau.String("value"),
						Value: luau.Create(&luau.ComputedIndexExpression{
							Expression: classId,
							Index:      key,
						}),
					}),
				),
			}),
		)

		result.Push(
			luau.Create(&luau.IfStatement{
				Condition: descriptorId,
				Statements: luau.MakeList(
					luau.Create(&luau.Assignment{
						Left: luau.Create(&luau.ComputedIndexExpression{
							Expression: classId,
							Index:      key,
						}),
						Operator: "=",
						Right:    luau.Property(descriptorId, "value"),
					}),
				),
				ElseBody: luau.MakeList(),
			}),
		)

		return result
	})

	result := luau.MakeList()
	result.PushList(initializers)
	result.PushList(transformParameterDecorators(state, member, classId))
	result.PushList(finalizers)
	return result
}

func transformPropertyDecorators(state *TransformState, member *ast.Node, classId luau.AnyIdentifier) *luau.List {
	initializers, finalizers := transformMemberDecorators(state, member, func(expression luau.IndexableExpression) *luau.List {
		// typescript enforces that property keys are static, so they shouldn't have prereqs
		keyPrereqs := NewPrereqs()
		key := transformPropertyName(state, keyPrereqs, member.Name())
		assert(keyPrereqs.Statements.IsEmpty())

		// decorator(Class, "name")
		return luau.MakeList(
			luau.Create(&luau.CallStatement{
				Expression: luau.Call(expression, classId, key),
			}),
		)
	})

	result := luau.MakeList()
	result.PushList(initializers)
	result.PushList(finalizers)
	return result
}

func transformParameterDecorators(state *TransformState, member *ast.Node, classId luau.AnyIdentifier) *luau.List {
	initializers := luau.MakeList()
	finalizers := luau.MakeList()

	for i, parameter := range member.Parameters() {
		paramInitializers, paramFinalizers := transformMemberDecorators(state, parameter, func(expression luau.IndexableExpression) *luau.List {
			// No member.name means it's the constructor, so the name argument should be nil
			var key luau.Expression
			if member.Name() != nil {
				key = state.GetClassElementObjectKey(member)
			} else {
				key = luau.Nil()
			}
			assert(key != nil, "Did not find method key for parameter decorator")

			// decorator(Class, "name", 0)
			return luau.MakeList(
				luau.Create(&luau.CallStatement{
					Expression: luau.Call(expression, classId, key, luau.Number(float64(i))),
				}),
			)
		})
		initializers.PushList(paramInitializers)
		finalizers.UnshiftList(paramFinalizers)
	}

	result := luau.MakeList()
	result.PushList(initializers)
	result.PushList(finalizers)
	return result
}

func transformClassDecorators(state *TransformState, node *ast.Node, classId luau.AnyIdentifier) *luau.List {
	initializers, finalizers := transformMemberDecorators(state, node, func(expression luau.IndexableExpression) *luau.List {
		// Class = decorator(Class) or Class
		return luau.MakeList(
			luau.Create(&luau.Assignment{
				Left:     classId,
				Operator: "=",
				Right:    luau.Binary(luau.Call(expression, classId), "or", classId),
			}),
		)
	})

	result := luau.MakeList()
	result.PushList(initializers)

	if constructor := findConstructor(node); constructor != nil {
		result.PushList(transformParameterDecorators(state, constructor, classId))
	}

	result.PushList(finalizers)
	return result
}

// transformDecorators is TSTransformer/nodes/class/transformDecorators.ts transformDecorators.
func transformDecorators(state *TransformState, node *ast.Node, classId luau.AnyIdentifier) *luau.List {
	result := luau.MakeList()

	// https://www.typescriptlang.org/docs/handbook/decorators.html#decorator-evaluation

	// Instance Decorators
	for _, member := range node.Members() {
		if !ast.HasStaticModifier(member) {
			// check `member.body` to skip overload signatures because decorators are not valid on them
			if ast.IsMethodDeclaration(member) && member.Body() != nil {
				result.PushList(transformMethodDecorators(state, member, classId))
			} else if ast.IsPropertyDeclaration(member) {
				result.PushList(transformPropertyDecorators(state, member, classId))
			}
		}
	}

	// Static Decorators
	for _, member := range node.Members() {
		if ast.HasStaticModifier(member) {
			// check `member.body` to skip overload signatures because decorators are not valid on them
			if ast.IsMethodDeclaration(member) && member.Body() != nil {
				result.PushList(transformMethodDecorators(state, member, classId))
			} else if ast.IsPropertyDeclaration(member) {
				result.PushList(transformPropertyDecorators(state, member, classId))
			}
		}
	}

	// Class Decorators
	result.PushList(transformClassDecorators(state, node, classId))

	return result
}
