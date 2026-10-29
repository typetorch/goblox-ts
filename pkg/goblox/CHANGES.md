# pkg/goblox API changes

The API is stable once committed. Every change after the first version is listed here, newest first,
with what a plugin has to do about it.

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
