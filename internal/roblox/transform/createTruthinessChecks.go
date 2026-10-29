package transform

// Port of TSTransformer/util/createTruthinessChecks.ts.

import (
	"strings"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/checker"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// willCreateTruthinessChecks is TSTransformer/util/createTruthinessChecks.ts willCreateTruthinessChecks.
func willCreateTruthinessChecks(type_ *checker.Type) bool {
	return isPossiblyType(type_, isNumberLiteralType(0)) ||
		isPossiblyType(type_, isNaNType) ||
		isPossiblyType(type_, isEmptyStringType)
}

// createTruthinessChecks is TSTransformer/util/createTruthinessChecks.ts createTruthinessChecks.
func createTruthinessChecks(state *TransformState, prereqs *Prereqs, exp luau.Expression, node *ast.Node) luau.Expression {
	type_ := state.GetType(node)
	isAssignableToZero := isPossiblyType(type_, isNumberLiteralType(0))
	isAssignableToNaN := isPossiblyType(type_, isNaNType)
	isAssignableToEmptyString := isPossiblyType(type_, isEmptyStringType)

	if isAssignableToZero || isAssignableToNaN || isAssignableToEmptyString {
		exp = prereqs.PushToVarIfComplex(exp, "value")
	}

	var checks []luau.Expression

	if isAssignableToZero {
		checks = append(checks, luau.Binary(exp, "~=", luau.Number(0)))
	}

	// workaround for https://github.com/microsoft/TypeScript/issues/32778
	if isAssignableToZero || isAssignableToNaN {
		checks = append(checks, luau.Binary(exp, "==", exp))
	}

	if isAssignableToEmptyString {
		checks = append(checks, luau.Binary(exp, "~=", luau.String("")))
	}

	checks = append(checks, exp)

	if state.Data.ProjectOptions.LogTruthyChanges &&
		(isAssignableToZero || isAssignableToNaN || isAssignableToEmptyString) {
		var checkStrs []string
		if isAssignableToZero {
			checkStrs = append(checkStrs, "0")
		}
		if isAssignableToZero || isAssignableToNaN {
			checkStrs = append(checkStrs, "NaN")
		}
		if isAssignableToEmptyString {
			checkStrs = append(checkStrs, `""`)
		}
		DiagnosticService.AddDiagnostic(diagnostics.Warnings.TruthyChange(strings.Join(checkStrs, ", ")).Create(node))
	}

	return binaryExpressionChain(checks, "and")
}
