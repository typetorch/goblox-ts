package luau

import "regexp"

// Ports of LuauAST/util/*.ts.

var luauMetamethods = map[string]bool{
	"__index": true, "__newindex": true, "__call": true, "__concat": true, "__unm": true, "__add": true,
	"__sub": true, "__mul": true, "__div": true, "__mod": true, "__pow": true, "__tostring": true,
	"__metatable": true, "__eq": true, "__lt": true, "__le": true, "__mode": true, "__gc": true,
	"__len": true,
}

// IsMetamethod returns true if the given string is a valid Luau metamethod.
func IsMetamethod(id string) bool { return luauMetamethods[id] }

var luauReservedClassFields = map[string]bool{"__index": true, "new": true}

// IsReservedClassField is LuauAST/util/isReservedClassField.ts.
func IsReservedClassField(id string) bool { return luauReservedClassFields[id] }

// IsReservedIdentifier is LuauAST/util/isReservedIdentifier.ts.
func IsReservedIdentifier(id string) bool { return globalsKeys[id] }

// LuauReservedKeywords is LUAU_RESERVED_KEYWORDS.
var LuauReservedKeywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true, "false": true,
	"for": true, "function": true, "if": true, "in": true, "local": true, "nil": true, "not": true,
	"or": true, "repeat": true, "return": true, "then": true, "true": true, "until": true, "while": true,
}

// LuauIdentifierRegex is LUAU_IDENTIFIER_REGEX.
var LuauIdentifierRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isIdentifierText(id string) bool {
	if len(id) == 0 {
		return false
	}
	for i := range len(id) {
		c := id[i]
		if c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (i > 0 && c >= '0' && c <= '9') {
			continue
		}
		return false
	}
	return true
}

// IsValidIdentifier returns true if the given string is a valid Luau identifier, and does not conflict
// with a temporary identifier.
func IsValidIdentifier(id string) bool {
	return !LuauReservedKeywords[id] && isIdentifierText(id)
}

var (
	decimalLiteralRegex     = regexp.MustCompile(`^(?:\d[\d_]*(?:\.[\d_]*)?|\.\d[\d_]*)(?:[eE][+-]?_*\d[\d_]*)?$`)
	binaryLiteralRegex      = regexp.MustCompile(`^0_*[bB]_*[01][01_]*$`)
	hexadecimalLiteralRegex = regexp.MustCompile(`^0_*[xX]_*[\da-fA-F][\da-fA-F_]*$`)
)

// IsValidNumberLiteral is LuauAST/util/isValidNumberLiteral.ts.
func IsValidNumberLiteral(text string) bool {
	return decimalLiteralRegex.MatchString(text) || binaryLiteralRegex.MatchString(text) || hexadecimalLiteralRegex.MatchString(text)
}
