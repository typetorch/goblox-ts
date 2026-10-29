package luau

// Port of LuauAST/impl/globals.ts. The nodes are shared (see sharedParent) and are cloned by Create
// whenever they are used as a child.

func sharedId(name string) *Identifier { return shared(&Identifier{Name: name}) }

func sharedProperty(expression *Identifier, name string) *PropertyAccessExpression {
	return shared(Create(&PropertyAccessExpression{Expression: expression, Name: name}))
}

var (
	coroutineId = sharedId("coroutine")
	mathId      = sharedId("math")
	stringId    = sharedId("string")
	tableId     = sharedId("table")
	utf8Id      = sharedId("utf8")
)

// Globals is `luau.globals`.
var Globals = struct {
	G            *Identifier // _G
	TS           *Identifier
	Assert       *Identifier
	Bit32        *Identifier
	Coroutine    struct{ Yield *PropertyAccessExpression }
	Error        *Identifier
	Exports      *Identifier
	Getmetatable *Identifier
	Ipairs       *Identifier
	Next         *Identifier
	Pairs        *Identifier
	Pcall        *Identifier
	Require      *Identifier
	Script       *Identifier
	Select       *Identifier
	Self         *Identifier
	Setmetatable *Identifier
	String       struct {
		Byte, Find, Format, Gmatch, Gsub, Lower, Match, Rep, Reverse, Split, Sub, Upper *PropertyAccessExpression
	}
	Super *Identifier
	Table struct {
		Clear, Clone, Concat, Create, Find, Freeze, Insert, Isfrozen, Maxn, Move, Pack, Remove, Sort, Unpack *PropertyAccessExpression
	}
	Utf8     struct{ Charpattern, Codes *PropertyAccessExpression }
	Math     struct{ Min *PropertyAccessExpression }
	Tostring *Identifier
	Type     *Identifier
	Typeof   *Identifier
	Unpack   *Identifier

	// roblox
	Game *Identifier
}{
	G:            sharedId("_G"),
	TS:           sharedId("TS"),
	Assert:       sharedId("assert"),
	Bit32:        sharedId("bit32"),
	Coroutine:    struct{ Yield *PropertyAccessExpression }{sharedProperty(coroutineId, "yield")},
	Error:        sharedId("error"),
	Exports:      sharedId("exports"),
	Getmetatable: sharedId("getmetatable"),
	Ipairs:       sharedId("ipairs"),
	Next:         sharedId("next"),
	Pairs:        sharedId("pairs"),
	Pcall:        sharedId("pcall"),
	Require:      sharedId("require"),
	Script:       sharedId("script"),
	Select:       sharedId("select"),
	Self:         sharedId("self"),
	Setmetatable: sharedId("setmetatable"),
	String: struct {
		Byte, Find, Format, Gmatch, Gsub, Lower, Match, Rep, Reverse, Split, Sub, Upper *PropertyAccessExpression
	}{
		sharedProperty(stringId, "byte"),
		sharedProperty(stringId, "find"),
		sharedProperty(stringId, "format"),
		sharedProperty(stringId, "gmatch"),
		sharedProperty(stringId, "gsub"),
		sharedProperty(stringId, "lower"),
		sharedProperty(stringId, "match"),
		sharedProperty(stringId, "rep"),
		sharedProperty(stringId, "reverse"),
		sharedProperty(stringId, "split"),
		sharedProperty(stringId, "sub"),
		sharedProperty(stringId, "upper"),
	},
	Super: sharedId("super"),
	Table: struct {
		Clear, Clone, Concat, Create, Find, Freeze, Insert, Isfrozen, Maxn, Move, Pack, Remove, Sort, Unpack *PropertyAccessExpression
	}{
		sharedProperty(tableId, "clear"),
		sharedProperty(tableId, "clone"),
		sharedProperty(tableId, "concat"),
		sharedProperty(tableId, "create"),
		sharedProperty(tableId, "find"),
		sharedProperty(tableId, "freeze"),
		sharedProperty(tableId, "insert"),
		sharedProperty(tableId, "isfrozen"),
		sharedProperty(tableId, "maxn"),
		sharedProperty(tableId, "move"),
		sharedProperty(tableId, "pack"),
		sharedProperty(tableId, "remove"),
		sharedProperty(tableId, "sort"),
		sharedProperty(tableId, "unpack"),
	},
	Utf8: struct{ Charpattern, Codes *PropertyAccessExpression }{
		sharedProperty(utf8Id, "charpattern"),
		sharedProperty(utf8Id, "codes"),
	},
	Math:     struct{ Min *PropertyAccessExpression }{sharedProperty(mathId, "min")},
	Tostring: sharedId("tostring"),
	Type:     sharedId("type"),
	Typeof:   sharedId("typeof"),
	Unpack:   sharedId("unpack"),

	Game: sharedId("game"),
}

// globalsKeys are the keys of the TypeScript `globals` object, for IsReservedIdentifier.
var globalsKeys = map[string]bool{
	"_G": true, "TS": true, "assert": true, "bit32": true, "coroutine": true, "error": true,
	"exports": true, "getmetatable": true, "ipairs": true, "next": true, "pairs": true, "pcall": true,
	"require": true, "script": true, "select": true, "self": true, "setmetatable": true,
	"string": true, "super": true, "table": true, "utf8": true, "math": true, "tostring": true,
	"type": true, "typeof": true, "unpack": true, "game": true,
}
