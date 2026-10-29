// Hand-written Luau ASTs rendered by the reference @roblox-ts/luau-ast 2.1.2 renderer.
// Regenerate ../render.json with ./generate.sh (see README.md).
import luau from "LuauAST";

const L = (...values: Array<luau.Node>) => luau.list.make<any>(...values);
const c = luau.create;
const K = luau.SyntaxKind;

const call = (name: string, ...args: Array<luau.Expression>) =>
	c(K.CallStatement, { expression: luau.call(luau.id(name), args) });
const local = (name: string | luau.AnyIdentifier, right?: luau.Expression) =>
	c(K.VariableDeclaration, { left: typeof name === "string" ? luau.id(name) : name, right });
const ret = (e: luau.Expression | luau.List<luau.Expression>) => c(K.ReturnStatement, { expression: e });
const paren = (e: luau.Expression) => c(K.ParenthesizedExpression, { expression: e });
const fn = (params: Array<luau.AnyIdentifier>, statements: Array<luau.Statement>, hasDotDotDot = false) =>
	c(K.FunctionExpression, { parameters: L(...params), hasDotDotDot, statements: L(...statements) });

export const cases: Array<[string, () => luau.List<luau.Statement>]> = [
	["literals", () => L(
		local("a", luau.nil()),
		local("b", luau.bool(true)),
		local("c", luau.bool(false)),
		local("d", luau.number(1.5)),
		local("e", luau.number(-3)),
		local("f", luau.number(1e21)),
		local("g", luau.number(0.1 + 0.2)),
		local("h", c(K.NumberLiteral, { value: "0x_FF" })),
		local("i", c(K.NumberLiteral, { value: "1_000" })),
		local("j", c(K.NumberLiteral, { value: "1__0e_5" })),
		local("k", c(K.NumberLiteral, { value: "0b1010" })),
		local("l", c(K.NumberLiteral, { value: "08" })),
		local("m", c(K.NumberLiteral, { value: "1e+400" })),
		local("n", c(K.VarArgsLiteral, {})),
		local("o", luau.number(123456789012)),
		local("p", luau.number(5e-7)),
	)],
	["strings", () => L(
		local("a", luau.string("plain")),
		local("b", luau.string('has "double"')),
		local("c", luau.string("has 'single' and \"double\"")),
		local("d", luau.string("multi\nline")),
		local("e", luau.string("tricky ]] and ]=] end]")),
		local("f", luau.string("ends with ]")),
		local("g", luau.string("explicit", "'")),
		local("h", luau.string("")),
		local("i", c(K.ComputedIndexExpression, { expression: luau.id("t"), index: luau.string("a\nb") })),
		local("j", luau.set([luau.string("ok"), luau.string("not ok"), luau.string("x\ny"), luau.number(1)])),
		local("k", luau.map([[luau.string("a\nb"), luau.number(1)], [luau.string("key"), luau.string("v")], [luau.string("end"), luau.number(2)]])),
	)],
	["identifiers and indexing", () => L(
		local("a", luau.property(luau.id("game"), "Workspace")),
		local("b", luau.property(luau.id("t"), "not valid")),
		local("c", luau.property(luau.id("t"), "end")),
		local("d", c(K.ComputedIndexExpression, { expression: luau.id("t"), index: luau.string("valid") })),
		local("e", c(K.ComputedIndexExpression, { expression: luau.id("t"), index: luau.number(1) })),
		local("f", c(K.MethodCallExpression, { expression: luau.id("obj"), name: "Method", args: L(luau.number(1), luau.string("x")) })),
		local("g", luau.call(luau.property(luau.id("a"), "b"), [])),
		local("h", luau.call(paren(fn([], [])), [])),
		local("i", paren(paren(luau.id("x")))),
		local("j", paren(luau.binary(luau.id("x"), "+", luau.id("y")))),
		call("print", luau.globals.TS, luau.globals.table.insert, luau.strings.__index),
	)],
	["precedence", () => L(
		local("a", luau.binary(luau.binary(luau.id("a"), "+", luau.id("b")), "*", luau.id("c"))),
		local("b", luau.binary(luau.id("a"), "-", luau.binary(luau.id("b"), "-", luau.id("c")))),
		local("c", luau.binary(luau.binary(luau.id("a"), "-", luau.id("b")), "-", luau.id("c"))),
		local("d", luau.binary(luau.id("a"), "^", luau.binary(luau.id("b"), "^", luau.id("c")))),
		local("e", luau.unary("-", luau.unary("-", luau.id("x")))),
		local("f", luau.unary("not", luau.binary(luau.id("a"), "and", luau.id("b")))),
		local("g", luau.unary("#", luau.id("t"))),
		local("h", luau.binary(luau.unary("-", luau.id("a")), "^", luau.id("b"))),
		local("i", luau.binary(luau.id("a"), "or", luau.binary(luau.id("b"), "and", luau.id("c")))),
		local("j", luau.binary(luau.binary(luau.id("a"), "or", luau.id("b")), "and", luau.id("c"))),
		local("k", luau.binary(luau.id("s"), "..", luau.binary(luau.id("t"), "..", luau.id("u")))),
		local("l", luau.binary(luau.id("x"), "//", luau.number(2))),
		local("m", luau.binary(
			c(K.IfExpression, { condition: luau.id("a"), expression: luau.number(1), alternative: luau.number(2) }),
			"+",
			luau.number(3),
		)),
		local("n", c(K.IfExpression, {
			condition: luau.id("a"),
			expression: luau.number(1),
			alternative: c(K.IfExpression, { condition: luau.id("b"), expression: luau.number(2), alternative: luau.number(3) }),
		})),
		local("o", luau.unary("-", luau.number(-5))),
		local("p", luau.binary(luau.id("a"), "~=", luau.binary(luau.id("b"), "==", luau.id("c")))),
	)],
	["tables", () => L(
		local("a", luau.array()),
		local("b", luau.array([luau.number(1), luau.number(2), luau.string("x")])),
		local("c", luau.map()),
		local("d", luau.map([[luau.string("x"), luau.number(1)], [luau.number(2), luau.array([luau.number(3)])], [luau.id("k"), luau.map([[luau.string("nested"), luau.bool(true)]])]])),
		local("e", luau.set()),
		local("f", luau.mixedTable([luau.number(1), [luau.string("a"), luau.number(2)], luau.number(3)])),
		local("g", luau.mixedTable()),
		local("h", fn([luau.id("x")], [ret(luau.map([[luau.string("y"), luau.id("x")]]))])),
	)],
	["interpolated strings", () => L(
		local("a", c(K.InterpolatedString, { parts: L(
			c(K.InterpolatedStringPart, { text: "hello " }),
			luau.id("name"),
			c(K.InterpolatedStringPart, { text: "! {braces} \\{escaped\\} \\\\u{41} \\u{41} line\nbreak\r\nwin" }),
			luau.array([luau.number(1)]),
			luau.binary(luau.id("a"), "+", luau.id("b")),
		) })),
	)],
	["statements", () => L(
		c(K.Comment, { text: " single line" }),
		c(K.Comment, { text: "multi\n  line ]] comment\n\n   \nend" }),
		local(luau.id("x")),
		c(K.VariableDeclaration, { left: L(luau.id("a"), luau.id("b")), right: L(luau.number(1), luau.number(2)) }),
		c(K.VariableDeclaration, { left: L(luau.id("c"), luau.id("d")), right: undefined }),
		c(K.Assignment, { left: luau.id("x"), operator: "=", right: luau.number(1) }),
		c(K.Assignment, { left: luau.id("x"), operator: "+=", right: luau.number(1) }),
		c(K.Assignment, { left: luau.id("x"), operator: "..=", right: luau.string("s") }),
		c(K.Assignment, { left: L(luau.id("a"), luau.property(luau.id("t"), "b")), operator: "=", right: L(luau.id("b"), luau.id("a")) }),
		c(K.DoStatement, { statements: L(call("f"), c(K.DoStatement, { statements: L() })) }),
		c(K.WhileStatement, { condition: luau.bool(true), statements: L(c(K.BreakStatement, {})) }),
		c(K.RepeatStatement, { condition: luau.id("done"), statements: L(call("step"), c(K.ContinueStatement, {})) }),
		c(K.NumericForStatement, { id: luau.id("i"), start: luau.number(1), end: luau.number(10), step: undefined, statements: L(call("print", luau.id("i"))) }),
		c(K.NumericForStatement, { id: luau.id("i"), start: luau.number(1), end: luau.number(10), step: luau.number(1), statements: L() }),
		c(K.NumericForStatement, { id: luau.id("i"), start: luau.number(10), end: luau.number(1), step: luau.number(-1), statements: L() }),
		c(K.NumericForStatement, { id: luau.id("i"), start: luau.number(0), end: luau.number(1), step: c(K.NumberLiteral, { value: "1.0" }), statements: L() }),
		c(K.ForStatement, { ids: L(luau.id("k"), luau.id("v")), expression: luau.call(luau.globals.pairs, [luau.id("t")]), statements: L(call("print", luau.id("k"), luau.id("v"))) }),
		c(K.ForStatement, { ids: L(), expression: luau.id("iter"), statements: L() }),
		c(K.IfStatement, { condition: luau.id("a"), statements: L(call("f")), elseBody: L() }),
		c(K.IfStatement, { condition: luau.id("a"), statements: L(call("f")), elseBody: L(call("g")) }),
		c(K.IfStatement, {
			condition: luau.id("a"),
			statements: L(call("f")),
			elseBody: c(K.IfStatement, {
				condition: luau.id("b"),
				statements: L(call("g")),
				elseBody: c(K.IfStatement, { condition: luau.id("c"), statements: L(), elseBody: L(call("h")) }),
			}),
		}),
		c(K.FunctionDeclaration, { localize: true, name: luau.id("foo"), parameters: L(luau.id("a"), luau.id("b")), hasDotDotDot: true, statements: L(ret(luau.id("a"))) }),
		c(K.FunctionDeclaration, { localize: false, name: luau.property(luau.id("M"), "bar"), parameters: L(), hasDotDotDot: false, statements: L() }),
		c(K.MethodDeclaration, { expression: luau.id("Class"), name: "method", parameters: L(luau.id("x")), hasDotDotDot: false, statements: L(ret(L(luau.id("self"), luau.id("x")))) }),
		local("fe", fn([], [], true)),
		local("fe2", fn([luau.id("a")], [call("print", luau.id("a"), c(K.VarArgsLiteral, {}))], true)),
		ret(L()),
	)],
	["semicolons", () => L(
		local("a", luau.id("b")),
		c(K.CallStatement, { expression: luau.call(paren(fn([], [])), []) }),
		call("f"),
		c(K.Comment, { text: "comment between" }),
		c(K.CallStatement, { expression: c(K.MethodCallExpression, { expression: paren(luau.id("x")), name: "m", args: L() }) }),
		c(K.Assignment, { left: luau.id("y"), operator: "=", right: luau.binary(luau.id("a"), "+", luau.id("b")) }),
		c(K.Assignment, { left: luau.property(paren(luau.binary(luau.id("p"), "or", luau.id("q"))), "z"), operator: "=", right: luau.number(1) }),
		local(luau.id("u")),
		c(K.Assignment, { left: L(c(K.ComputedIndexExpression, { expression: paren(luau.call(luau.id("g"), [])), index: luau.number(1) })), operator: "=", right: luau.number(2) }),
		c(K.VariableDeclaration, { left: L(luau.id("m"), luau.id("n")), right: undefined }),
		c(K.CallStatement, { expression: luau.call(paren(luau.id("simple")), []) }),
		local("v", luau.number(1)),
		c(K.CallStatement, { expression: luau.call(paren(fn([], [])), []) }),
		local("w", c(K.IfExpression, { condition: luau.id("a"), expression: luau.number(1), alternative: luau.id("b") })),
		c(K.CallStatement, { expression: luau.call(paren(fn([], [])), []) }),
	)],
	["temp ids", () => {
		const t1 = luau.tempId("value");
		const t2 = luau.tempId("value");
		const t3 = luau.tempId();
		const t4 = luau.tempId();
		const t5 = luau.tempId("result");
		const t6 = luau.tempId("value");
		const t7 = luau.tempId("i");
		return L(
			local("_value"),
			local(t1, luau.number(1)),
			local(t2, luau.number(2)),
			local(t3, luau.number(3)),
			local(t4, t3),
			local("_result_1"),
			local(t5, luau.binary(t1, "+", t2)),
			c(K.DoStatement, { statements: L(local(t6, t1), call("print", t6)) }),
			c(K.FunctionDeclaration, {
				localize: true, name: luau.id("f"), parameters: L(luau.id("_value"), t7), hasDotDotDot: false,
				statements: L(local(luau.tempId("value"), t7), ret(t7)),
			}),
			c(K.IfStatement, {
				condition: t5,
				statements: L(local(luau.tempId("x"), luau.number(1))),
				elseBody: L(local(luau.tempId("x"), luau.number(2)), local("_x"), local(luau.tempId("x"), luau.number(3))),
			}),
			c(K.NumericForStatement, { id: luau.tempId("i"), start: luau.number(1), end: luau.number(2), step: undefined, statements: L(local(luau.tempId("i"))) }),
			c(K.ForStatement, { ids: L(luau.tempId("k"), luau.tempId("v")), expression: luau.call(luau.globals.pairs, [t5]), statements: L(call("print", t5)) }),
			c(K.WhileStatement, { condition: luau.bool(true), statements: L(local(luau.tempId("value")), c(K.BreakStatement, {})) }),
			ret(L(t1, t2, t3, t4, t5)),
		);
	}],
	["final statement then comment", () => L(
		call("f"),
		ret(luau.number(1)),
		c(K.Comment, { text: "trailing" }),
	)],
];
