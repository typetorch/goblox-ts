// Runs testdata/queries.json through the reference rojo-resolver 1.3.0 and path-translator 1.1.1.
// Paths in the output are relative to testdata (absolute paths are prefixed with "<root>").
import fs from "fs";
import path from "path";
import { PathTranslator } from "./path-translator/PathTranslator";
import { RbxPathParent, RojoResolver } from "./rojo-resolver/RojoResolver";

const root = process.argv[2];
const q = JSON.parse(fs.readFileSync(path.join(root, "queries.json"), "utf8"));
const abs = (p: string) => path.join(root, p);
const rel = (v: unknown): unknown =>
	typeof v === "string" ? v.split(root).join("<root>") : Array.isArray(v) ? v.map(rel) : v;
const relPath = (p: Array<string | symbol>) => p.map(v => (v === RbxPathParent ? "<parent>" : v));

function describe(resolver: RojoResolver, files: Array<string>) {
	return {
		warnings: rel(resolver.getWarnings()),
		isGame: resolver.isGame,
		partitions: resolver.getPartitions().map(p => ({ fsPath: rel(p.fsPath), rbxPath: p.rbxPath })),
		files: files.map(f => {
			const rbxPath = resolver.getRbxPathFromFilePath(abs(f));
			return {
				file: f,
				rbxPath: rbxPath ?? null,
				rbxType: resolver.getRbxTypeFromFilePath(abs(f)),
				networkType: rbxPath ? resolver.getNetworkType(rbxPath) : null,
				isolated: rbxPath ? resolver.isIsolated(rbxPath) : null,
			};
		}),
	};
}

const game = RojoResolver.fromPath(abs(q.projects[0].config));
const result = {
	projects: q.projects.map((p: { config: string; files: Array<string> }) =>
		describe(RojoResolver.fromPath(abs(p.config)), p.files),
	),
	find: q.find.map((p: string) => {
		const r = RojoResolver.findRojoConfigFilePath(abs(p));
		return { path: r.path === undefined ? null : rel(r.path), warnings: rel(r.warnings) };
	}),
	synthetic: describe(RojoResolver.synthetic(abs(q.synthetic.base)), q.synthetic.files),
	relative: q.relative.map(([a, b]: [Array<string>, Array<string>]) => relPath(RojoResolver.relative(a, b) as never)),
	fileRelations: q.rbxPaths.map((a: Array<string>) => q.rbxPaths.map((b: Array<string>) => game.getFileRelation(a, b))),
	networkTypes: q.rbxPaths.map((a: Array<string>) => game.getNetworkType(a)),
	translator: [false, true].flatMap(declaration =>
		[false, true].map(luau => {
			const t = new PathTranslator(abs(q.translator.rootDir), abs(q.translator.outDir), undefined, declaration, luau);
			return {
				declaration,
				luau,
				outputPath: q.translator.files.map((f: string) => rel(t.getOutputPath(abs(f)))),
				outputDeclarationPath: q.translator.files.map((f: string) => rel(t.getOutputDeclarationPath(abs(f)))),
				outputTransformedPath: q.translator.files.map((f: string) => rel(t.getOutputTransformedPath(abs(f)))),
				importPath: q.translator.files.map((f: string) => rel(t.getImportPath(abs(f)))),
				importPathNodeModule: q.translator.files.map((f: string) => rel(t.getImportPath(abs(f), true))),
				inputPaths: q.translator.outputs.map((f: string) => rel(t.getInputPaths(abs(f)))),
			};
		}),
	),
};
process.stdout.write(JSON.stringify(result, null, "\t") + "\n");
