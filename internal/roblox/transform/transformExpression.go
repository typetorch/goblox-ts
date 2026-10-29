package transform

// Port of TSTransformer/nodes/expressions/transformExpression.ts.

import (
	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// ExpressionTransformer is the dispatch signature of TRANSFORMER_BY_KIND.
type ExpressionTransformer = func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression

func expressionNoEmit(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	return luau.NewNone()
}

func expressionDiagnostic(factory *diagnostics.DiagnosticFactory) ExpressionTransformer {
	return func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
		DiagnosticService.AddDiagnostic(factory.Create(node))
		return luau.NewNone()
	}
}

var expressionTransformerByKind map[ast.Kind]ExpressionTransformer

func init() {
	expressionTransformerByKind = map[ast.Kind]ExpressionTransformer{
		// banned expressions
		ast.KindBigIntLiteral:            expressionDiagnostic(diagnostics.Errors.NoBigInt),
		ast.KindNullKeyword:              expressionDiagnostic(diagnostics.Errors.NoNullLiteral),
		ast.KindPrivateIdentifier:        expressionDiagnostic(diagnostics.Errors.NoPrivateIdentifier),
		ast.KindRegularExpressionLiteral: expressionDiagnostic(diagnostics.Errors.NoRegex),
		ast.KindTypeOfExpression:         expressionDiagnostic(diagnostics.Errors.NoTypeOfExpression),

		// skip transforms
		ast.KindImportKeyword: expressionNoEmit,

		// regular transforms
		ast.KindArrayLiteralExpression: transformArrayLiteralExpression,
		ast.KindArrowFunction:          withoutPrereqs(transformFunctionExpression),
		ast.KindAsExpression:           transformTypeExpression,
		ast.KindAwaitExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformAwaitExpression(state, prereqs, node)
		},
		ast.KindBinaryExpression:            transformBinaryExpression,
		ast.KindCallExpression:              transformCallExpression,
		ast.KindClassExpression:             transformClassExpression,
		ast.KindConditionalExpression:       transformConditionalExpression,
		ast.KindDeleteExpression:            transformDeleteExpression,
		ast.KindElementAccessExpression:     transformElementAccessExpression,
		ast.KindExpressionWithTypeArguments: transformTypeExpression,
		ast.KindFalseKeyword: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformFalseKeyword()
		},
		ast.KindFunctionExpression: withoutPrereqs(transformFunctionExpression),
		ast.KindIdentifier:         withoutPrereqs(transformIdentifier),
		ast.KindJsxElement:         transformJsxElement,
		ast.KindJsxExpression:      transformJsxExpression,
		ast.KindJsxFragment: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformJsxFragment(state, prereqs, node)
		},
		ast.KindJsxSelfClosingElement: transformJsxSelfClosingElement,
		ast.KindNewExpression:         transformNewExpression,
		ast.KindNonNullExpression:     transformTypeExpression,
		ast.KindNoSubstitutionTemplateLiteral: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformNoSubstitutionTemplateLiteral(state, node)
		},
		ast.KindNumericLiteral: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformNumericLiteral(state, node)
		},
		ast.KindObjectLiteralExpression: transformObjectLiteralExpression,
		ast.KindOmittedExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformOmittedExpression()
		},
		ast.KindParenthesizedExpression: transformParenthesizedExpression,
		ast.KindPostfixUnaryExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformPostfixUnaryExpression(state, prereqs, node)
		},
		ast.KindPrefixUnaryExpression:    transformPrefixUnaryExpression,
		ast.KindPropertyAccessExpression: transformPropertyAccessExpression,
		ast.KindSatisfiesExpression:      transformTypeExpression,
		ast.KindSpreadElement: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformSpreadElement(state, prereqs, node)
		},
		ast.KindStringLiteral: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformStringLiteral(state, node)
		},
		ast.KindSuperKeyword: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformSuperKeyword()
		},
		ast.KindTaggedTemplateExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformTaggedTemplateExpression(state, prereqs, node)
		},
		ast.KindTemplateExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformTemplateExpression(state, prereqs, node)
		},
		ast.KindThisKeyword: withoutPrereqs(transformThisExpression),
		ast.KindTrueKeyword: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformTrueKeyword()
		},
		ast.KindTypeAssertionExpression: transformTypeExpression,
		ast.KindVoidExpression: func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
			return transformVoidExpression(state, prereqs, node)
		},
		ast.KindYieldExpression: transformYieldExpression,
	}
}

// withoutPrereqs is TSTransformer/util/withoutPrereqs.ts: it adapts transforms that cannot emit caller
// prerequisites to the dispatch signature.
func withoutPrereqs[U luau.Expression](transform func(state *TransformState, node *ast.Node) U) ExpressionTransformer {
	return func(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
		return transform(state, node)
	}
}

func transformExpression(state *TransformState, prereqs *Prereqs, node *ast.Node) luau.Expression {
	transformer := expressionTransformerByKind[node.Kind]
	if transformer != nil {
		expression := transformer(state, prereqs, node)
		t := state.GetType(node)
		validateMethodExpression(state, node, t)
		if getOrSetDefault(state.MultiTransformState.IsPrimitiveTypeCache, t, func() bool {
			return isDefinitelyType(t, isBooleanType, isNumberType, isStringType, isUndefinedType)
		}) {
			markPrimitiveValue(expression)
		}
		return expression
	}
	assert(false, "Unknown expression: "+getKindName(node.Kind))
	return nil
}
