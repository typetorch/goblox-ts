# Plugins and the checker

Decision (2026-10-29): goblox keeps roblox-ts's model. Each file a plugin changed is printed and parsed
again, and the result is a new program that the Luau transform runs on. Unchanged files keep their
already parsed and bound `SourceFile`.

## What roblox-ts does

`compileFiles.ts` runs the ts-patch transformers (`ts.transformNodes`), then hits the same problem goblox
has: transformed nodes have no symbols, and there is no way to re-bind an existing file. So it prints
every transformed file and creates a proxy program from the printed text. A `CompilerHost` serves the
new text for the changed files and the old `SourceFile` for the rest. `getPreEmitDiagnostics` and the
Luau transform then run on the proxy program. On gnome-game this costs 0.54 s (print) plus 0.68 s
(program) in rbxtsc (see BENCHMARKS.md).

## What goblox does

`project.RunPlugins` (`goblox.RunPlugins`) works the same way:

1. For each compilable file, every plugin's `TransformSourceFile` runs in order. Each gets a
   `TransformContext` with the original program's checker for that file and a fresh `EmitContext`.
2. Each changed file is printed with typescript-go's printer.
3. `proxyHost` (a `CompilerHost`) serves the printed text, parsed again, for the changed files. Every
   other file gets the original `SourceFile` object (same parse options), so it is not parsed again.
   Its declarations are bound again by the new program's binder pass, which is cheap. The type check
   always starts from scratch.
4. `compiler.NewProgram` builds the proxy program. Pre-emit diagnostics and the Luau transform use its
   checkers. Every node a plugin emitted is a real parse-tree node there, with symbols and types.

## Alternatives that were rejected

- **Binding transformed nodes in place.** typescript-go's binder and checker store per-node data
  (symbols, flow nodes, node links) keyed on parse-tree nodes and the file's binding state. A factory
  node has no `Parent` chain, no position, and no flow node, so the checker crashes or returns error
  types. Faking all of that is a fork of the binder.
- **Mapping each synthetic node to its original (`EmitContext.ParseNode`) during the Luau transform.**
  This works for nodes that a plugin only updated. It fails for new code: @typetorch/transformer emits
  guard functions, DI ids and macro bodies whose types the Luau transform must know (macro dispatch,
  `isLuaTupleType`, array/map method calls). It would also break byte-identity with roblox-ts. roblox-ts
  compiles the re-parsed text, so its temp ids, comment handling and expression shapes follow the
  printed code.

## Consequences for plugin authors (also in `pkg/goblox` package docs)

- Inside `TransformSourceFile`, only query the checker with original nodes. For a node built or updated
  by a factory, call `ctx.EmitContext.ParseNode(node)` first.
- Report diagnostics against the original nodes. Positions in a re-parsed file refer to the printed
  text.
- Plugins run sequentially over files today. The checker that `GetTypeCheckerForFile` returns is shared
  by the files assigned to it. Parallelizing per checker is a later optimization and does not change the
  API.
