package transform

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// Expected Luau from roblox-ts 68648b0 (tests/src/tests/try.spec.ts and truthiness.spec.ts). The
// statement transforms themselves go through transformExpression and transformStatementList and are
// covered by the fixture harness instead.

func TestCollapseFlowControlCases(t *testing.T) {
	t.Parallel()
	tsConstant := func(name string) luau.Expression { return luau.Property(luau.Id("TS"), name) }

	t.Run("single case", func(t *testing.T) {
		t.Parallel()
		exitTypeId := luau.TempId("exitType")
		got := luau.RenderAST(collapseFlowControlCases(exitTypeId, []flowControlCase{
			{
				condition:  luau.Binary(exitTypeId, "==", tsConstant("TRY_BREAK")),
				statements: luau.MakeList(luau.Create(&luau.BreakStatement{})),
			},
		}))
		want := "if _exitType then\n\tbreak\nend\n"
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})

	t.Run("return then continue", func(t *testing.T) {
		t.Parallel()
		exitTypeId := luau.TempId("exitType")
		returnsId := luau.TempId("returns")
		got := luau.RenderAST(collapseFlowControlCases(exitTypeId, []flowControlCase{
			{
				condition: luau.Binary(exitTypeId, "==", tsConstant("TRY_RETURN")),
				statements: luau.MakeList(luau.Create(&luau.ReturnStatement{
					Expression: luau.Call(luau.Globals.Unpack, returnsId),
				})),
			},
			{
				condition:  luau.Binary(exitTypeId, "==", tsConstant("TRY_CONTINUE")),
				statements: luau.MakeList(luau.Create(&luau.ContinueStatement{})),
			},
		}))
		want := testTabs(`if _exitType == TS.TRY_RETURN then
    return unpack(_returns)
elseif _exitType then
    continue
end
`)
		if got != want {
			t.Errorf("got:\n%s\nwant:\n%s", got, want)
		}
	})
}

func TestBinaryExpressionChain(t *testing.T) {
	t.Parallel()
	prereqs := NewPrereqs()
	result := binaryExpressionChain([]luau.Expression{testId("a"), testId("b"), testId("c")}, "and")
	if got, want := renderMacroResult(prereqs, result), "local x = a and b and c\n"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
