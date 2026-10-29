package transform

import (
	"github.com/microsoft/typescript-go/internal/roblox/luau"
)

// getStatementsEffects is `getEffects(statements)` for a luau.List: roblox-ts's getEffects takes
// `luau.Node | luau.List<luau.Statement>`, but the Go signature only takes a luau.Node. The effects
// of a list are the join of the effects of its statements (NO_EFFECTS when it is empty). See
// docs/goblox/AGENT3_NOTES.md.
func getStatementsEffects(statements *luau.List) *EvaluationEffects {
	return joinEffects(luau.ListMapToArray(statements, getEffects)...)
}
