# pkg/goblox API changes

The API is stable once committed. Every change after the first version is listed here, newest first,
with what a plugin has to do about it.

## v3 (2026-10-30)

All additions; nothing a v2 plugin uses changed.

- `PluginOptions.PrintUnchanged`: print and parse again the files no plugin changed too, as rbxtsc does.
  Off by default: `RunPlugins` keeps the parsed and bound `SourceFile` of a file no plugin changed, unless
  a reprint would change the file's Luau. `PluginResult.Transformed` lists the printed files, so it can
  have a file no plugin changed, and it doesn't have every file a plugin left alone. A plugin that returns
  its input file when nothing changed saves that file a parse and a bind.

## v2 (2026-10-29)

All additions; nothing a v1 plugin uses changed.

- `LoadProgram`: when the tsconfig (and the configs it extends) has no `types`, every package of the
  type roots is included, as in TypeScript 5. A package whose package.json has `"typings": null` is
  skipped. This is done by passing `types: ["*"]`, TypeScript 7's wildcard. An explicit `types` (even
  `[]`) is left alone. Plugins can drop their `automaticTypeDirectiveNames` workaround. (REQUESTS.md 1)
- `NodeBuilderFlags*` constants, generated from typescript-go's `nodebuilder.Flags*`. For example,
  `NodeBuilderFlagsInTypeAlias | NodeBuilderFlagsIgnoreErrors` for `Checker.TypeToTypeNode`.
  (REQUESTS.md 2)
- `PluginEntries(program) []map[string]any`: the tsconfig's `compilerOptions.plugins` entries as plain
  Go values (`map[string]any`, `[]any`, `string`, `float64`, `bool`, `nil`). They are taken from the
  nearest config of the `extends` chain that sets `plugins`, the same rule as any other compiler option.
  Comments and trailing commas in the tsconfig are fine. (REQUESTS.md 3)
- Plugin registry: `PluginFactory`, `RegisterPlugin(name, factory)`, `RegisteredPlugins()` and
  `PluginsForProgram(program) ([]Plugin, []error)`.
  - The name is the entry's `transform`, i.e. the npm package of the JS transformer (`"@typetorch/transformer"`).
  - Register from an `init` function. The goblox CLI runs the plugins the tsconfig lists, in order.
  - Entries without `transform` are language service plugins and are skipped.
  - A `transform` with no registered Go plugin is an error.

## v1 (2026-10-29, first version)

- Superset of the `gobloxshim` stand-in in typetorch-transformer's `go-port` branch: same type
  aliases, generated constants (`gen/main.go`, prefixes `Kind`, `SymbolFlags`, `NodeFlags`,
  `ModifierFlags`, `TokenFlags`, `CheckFlags`, `TypeFlags`, `ObjectFlags`, `SignatureKind`,
  `ElementFlags`, `TypeFormatFlags`, `GeneratedIdentifierFlags`, `EmitFlags`, `Category`), free
  functions, `TransformContext` and `Plugin`. A port written against the shim switches by changing
  the import path to `github.com/microsoft/typescript-go/pkg/goblox`.
- `TransformContext` and `Plugin` are aliases of `internal/roblox/project`'s types, so the compiler
  runs plugins without adapters.
- Added: `LoadOptions`, `LoadProgram`, `GetConfigDiagnostics`, `GetPreEmitDiagnostics`,
  `PluginOptions`, `PluginResult`, `RunPlugins`, `CompilableSourceFiles`,
  `SortAndDeduplicateDiagnostics`, `WriteDiagnostics`.
