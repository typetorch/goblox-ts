package transform

// Small private helpers of the agent-4 port (block comments, JSX) with no roblox-ts counterpart.

import (
	"strings"
	"unicode"
)

// isJsWhiteSpace is JavaScript's WhiteSpace or LineTerminator, the characters String.prototype.trim
// removes.
func isJsWhiteSpace(r rune) bool {
	switch r {
	case '\t', '\v', '\f', '\n', '\r', 0xFEFF, 0x2028, 0x2029:
		return true
	}
	return unicode.Is(unicode.Zs, r)
}

// jsTrimStart is JavaScript's String.prototype.trimStart.
func jsTrimStart(s string) string { return strings.TrimLeftFunc(s, isJsWhiteSpace) }

// jsTrimEnd is JavaScript's String.prototype.trimEnd.
func jsTrimEnd(s string) string { return strings.TrimRightFunc(s, isJsWhiteSpace) }
