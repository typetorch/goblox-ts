package luau

// Port of LuauAST/impl/strings.ts.

func sharedString(value string) *StringLiteral { return shared(&StringLiteral{Value: value}) }

// Strings is `luau.strings`.
var Strings = struct {
	// metamethods
	Index, Tostring, Mode, K *StringLiteral
	// types
	Number, Table *StringLiteral
	// opcall
	Success, Value, Error *StringLiteral
	// other
	CommaSpace *StringLiteral // ", ", used for ReadonlyArray.join()
}{
	Index:      sharedString("__index"),
	Tostring:   sharedString("__tostring"),
	Mode:       sharedString("__mode"),
	K:          sharedString("k"),
	Number:     sharedString("number"),
	Table:      sharedString("table"),
	Success:    sharedString("success"),
	Value:      sharedString("value"),
	Error:      sharedString("error"),
	CommaSpace: sharedString(", "),
}
