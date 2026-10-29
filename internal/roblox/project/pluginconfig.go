package project

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/compiler"
	"github.com/microsoft/typescript-go/internal/module"
	"github.com/microsoft/typescript-go/internal/tsoptions"
	"github.com/microsoft/typescript-go/internal/tspath"
)

// PluginEntries returns the tsconfig's `compilerOptions.plugins` entries, decoded to plain Go values
// (map[string]any, []any, string, float64, bool, nil). typescript-go does not keep `plugins` in
// CompilerOptions, so the config files are read again. Like any compiler option, `plugins` comes from
// the nearest config of the `extends` chain that sets it (a later `extends` entry overrides an earlier
// one).
func PluginEntries(program *compiler.Program) []map[string]any {
	commandLine := program.CommandLine()
	if commandLine == nil || commandLine.ConfigFile == nil {
		return nil
	}
	fs := program.Host().FS()
	host := &parseHost{fs: fs, cwd: program.GetCurrentDirectory()}
	plugins, _ := findPluginsOption(host, commandLine.ConfigName(), map[string]bool{})
	var entries []map[string]any
	for _, plugin := range plugins {
		if entry, ok := plugin.(map[string]any); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// findPluginsOption returns the plugins option of configFileName or of the configs it extends.
func findPluginsOption(host *parseHost, configFileName string, seen map[string]bool) ([]any, bool) {
	if seen[configFileName] {
		return nil, false
	}
	seen[configFileName] = true
	text, ok := host.fs.ReadFile(configFileName)
	if !ok {
		return nil, false
	}
	raw, _ := tsoptions.ParseConfigFileTextToJson(configFileName, tspath.ToPath(configFileName, host.cwd, host.fs.UseCaseSensitiveFileNames()), text)
	config, ok := plainJSON(raw).(map[string]any)
	if !ok {
		return nil, false
	}
	if compilerOptions, ok := config["compilerOptions"].(map[string]any); ok {
		if plugins, ok := compilerOptions["plugins"].([]any); ok {
			return plugins, true
		}
	}
	var extends []string
	switch e := config["extends"].(type) {
	case string:
		extends = []string{e}
	case []any:
		for _, v := range e {
			if s, ok := v.(string); ok {
				extends = append(extends, s)
			}
		}
	}
	for _, extendedConfig := range slices.Backward(extends) {
		if extendedFileName := resolveExtendedConfig(host, configFileName, extendedConfig); extendedFileName != "" {
			if plugins, ok := findPluginsOption(host, extendedFileName, seen); ok {
				return plugins, true
			}
		}
	}
	return nil, false
}

// resolveExtendedConfig is tsoptions' getExtendsConfigPath: a path, or a package resolved like TypeScript does.
func resolveExtendedConfig(host *parseHost, configFileName string, extendedConfig string) string {
	extendedConfig = tspath.NormalizeSlashes(extendedConfig)
	basePath := tspath.GetDirectoryPath(configFileName)
	if tspath.IsRootedDiskPath(extendedConfig) || strings.HasPrefix(extendedConfig, "./") || strings.HasPrefix(extendedConfig, "../") {
		extendedFileName := tspath.GetNormalizedAbsolutePath(extendedConfig, basePath)
		if !host.fs.FileExists(extendedFileName) && !tspath.FileExtensionIs(extendedFileName, tspath.ExtensionJson) {
			extendedFileName += tspath.ExtensionJson
		}
		if host.fs.FileExists(extendedFileName) {
			return extendedFileName
		}
		return ""
	}
	resolved := module.ResolveConfig(extendedConfig, tspath.CombinePaths(basePath, "tsconfig.json"), host)
	if resolved != nil && resolved.IsResolved() {
		return resolved.ResolvedFileName
	}
	return ""
}

// plainJSON converts the ordered maps of a parsed tsconfig to map[string]any.
func plainJSON(value any) any {
	switch v := value.(type) {
	case *collections.OrderedMap[string, any]:
		m := make(map[string]any, v.Size())
		for key, item := range v.Entries() {
			m[key] = plainJSON(item)
		}
		return m
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = plainJSON(item)
		}
		return out
	}
	return value
}

var _ module.ResolutionHost = (*parseHost)(nil)

// PluginFactory creates a plugin from its tsconfig entry (`{ "transform": "<name>", ...options }`).
type PluginFactory func(entry map[string]any) (Plugin, error)

var (
	pluginRegistryMu sync.Mutex
	pluginRegistry   = map[string]PluginFactory{}
)

// RegisterPlugin makes a Go plugin available to tsconfig entries whose `transform` is name (the npm
// package of the JavaScript transformer it ports, e.g. "@typetorch/transformer"). Call it from an
// init function of the plugin's package and link the package into the goblox binary.
func RegisterPlugin(name string, create PluginFactory) {
	pluginRegistryMu.Lock()
	defer pluginRegistryMu.Unlock()
	if _, exists := pluginRegistry[name]; exists {
		panic(fmt.Sprintf("goblox: plugin %q registered twice", name))
	}
	pluginRegistry[name] = create
}

// RegisteredPlugins are the names passed to RegisterPlugin, sorted.
func RegisteredPlugins() []string {
	pluginRegistryMu.Lock()
	defer pluginRegistryMu.Unlock()
	names := make([]string, 0, len(pluginRegistry))
	for name := range pluginRegistry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnregisteredPluginError is the error of PluginsForProgram for a tsconfig entry whose transform has no
// registered Go plugin (a JavaScript transformer goblox cannot run).
type UnregisteredPluginError struct{ Name string }

func (e *UnregisteredPluginError) Error() string {
	return fmt.Sprintf("no Go plugin is registered for the transformer %q", e.Name)
}

// PluginsForProgram creates the plugins of the program's tsconfig `plugins` entries, in order. Entries
// without a `transform` are language service plugins and are skipped. An entry whose transform has no
// registered Go plugin is an *UnregisteredPluginError, and a factory error is returned wrapped.
func PluginsForProgram(program *compiler.Program) ([]Plugin, []error) {
	var plugins []Plugin
	var errs []error
	for _, entry := range PluginEntries(program) {
		name, ok := entry["transform"].(string)
		if !ok {
			continue
		}
		pluginRegistryMu.Lock()
		create := pluginRegistry[name]
		pluginRegistryMu.Unlock()
		if create == nil {
			errs = append(errs, &UnregisteredPluginError{Name: name})
			continue
		}
		plugin, err := create(entry)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
			continue
		}
		plugins = append(plugins, plugin)
	}
	return plugins, errs
}
