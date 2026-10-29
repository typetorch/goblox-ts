package luau

// Assert is LuauAST/util/assert.ts: it panics with roblox-ts's "compiler bug" message.
func Assert(value bool, message ...string) {
	if !value {
		msg := ""
		if len(message) > 0 {
			msg = message[0]
		}
		panic("Assertion Failed! " + msg +
			"\nThis is a compiler bug! Please submit a bug report here:" +
			"\nhttps://github.com/roblox-ts/roblox-ts/issues")
	}
}
