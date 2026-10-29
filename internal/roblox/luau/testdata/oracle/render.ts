// Renders cases.ts with the reference renderer and prints the fixture JSON on stdout.
import luau, { renderAST } from "LuauAST";
import { cases } from "./cases";

// getKindName returns the last enum alias for a few kinds (MixedTable is "LastExpression").
const KIND_ALIASES: Record<string, string> = {
	FirstIndexableExpression: "Identifier",
	FirstExpression: "Identifier",
	LastIndexableExpression: "ParenthesizedExpression",
	LastExpression: "MixedTable",
	FirstStatement: "Assignment",
	LastStatement: "Comment",
	FirstField: "MapField",
	LastField: "InterpolatedStringPart",
};

function serialize(value: unknown): unknown {
	if (luau.list.isList(value)) {
		return { list: luau.list.mapToArray(value, serialize) };
	}
	if (luau.isNode(value)) {
		const name = luau.getKindName(value.kind);
		const result: Record<string, unknown> = { kind: KIND_ALIASES[name] ?? name };
		for (const [key, field] of Object.entries(value)) {
			if (key === "kind" || key === "parent" || field === undefined) continue;
			result[key] = typeof field === "object" ? serialize(field) : field;
		}
		return result;
	}
	return value;
}

const fixtures = cases.map(([name, build]) => {
	const ast = build();
	const json = serialize(ast);
	let output: string;
	try {
		output = renderAST(ast);
	} catch (e) {
		output = "ERROR: " + String((e as Error).message);
	}
	return { name, ast: json, output };
});
process.stdout.write(JSON.stringify(fixtures, null, "\t") + "\n");
