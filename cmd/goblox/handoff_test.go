package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/json"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

func TestHandoffFrames(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	frames := []struct {
		kind    byte
		payload string
	}{{frameStdout, "out"}, {frameStderr, ""}, {frameExit, `{"code":2}`}}
	for _, f := range frames {
		if err := writeFrame(&buf, f.kind, []byte(f.payload)); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range frames {
		kind, payload, err := readFrame(&buf)
		if err != nil || kind != want.kind || string(payload) != want.payload {
			t.Fatalf("readFrame = %q %q %v, want %q %q", kind, payload, err, want.kind, want.payload)
		}
	}
	if _, _, err := readFrame(&buf); !errors.Is(err, io.EOF) {
		t.Errorf("readFrame at the end: %v", err)
	}

	var header [5]byte
	header[0] = frameStdout
	binary.BigEndian.PutUint32(header[1:], handoffMaxFrame+1)
	if _, _, err := readFrame(bytes.NewReader(header[:])); err == nil {
		t.Error("readFrame accepted an oversized frame")
	}
	if _, _, err := readFrame(bytes.NewReader([]byte{frameStdout, 0, 0, 0, 4, 'a'})); err == nil {
		t.Error("readFrame accepted a truncated frame")
	}
}

func TestHandoffSettings(t *testing.T) { //nolint:paralleltest // t.Setenv
	tsConfigPath := "/game/tsconfig.json"
	base := func() project.BuildOptions {
		return project.BuildOptions{
			CurrentDirectory: "/game",
			Overrides:        project.PartialProjectOptions{"rojo": "/game/default.project.json"},
			Incremental:      &project.IncrementalSession{Persist: true},
		}
	}
	settings := func(opts project.BuildOptions) string {
		t.Helper()
		text, err := handoffSettingsOf(opts, tsConfigPath)
		if err != nil {
			t.Fatal(err)
		}
		return text
	}
	want := settings(base())

	same := map[string]func(*project.BuildOptions){
		"verbose":          func(o *project.BuildOptions) { o.Overrides["verbose"] = true },
		"writeOnlyChanged": func(o *project.BuildOptions) { o.Overrides["writeOnlyChanged"] = true },
		"watch session": func(o *project.BuildOptions) {
			o.Incremental = &project.IncrementalSession{Persist: true, Memory: true}
		},
	}
	different := map[string]func(*project.BuildOptions){
		"luau":           func(o *project.BuildOptions) { o.Overrides["luau"] = true },
		"rojo":           func(o *project.BuildOptions) { o.Overrides["rojo"] = "/game/other.project.json" },
		"checkers":       func(o *project.BuildOptions) { o.Checkers = 2 },
		"noIncremental":  func(o *project.BuildOptions) { o.Incremental = nil },
		"emitFirst":      func(o *project.BuildOptions) { o.EmitFirst = true },
		"working folder": func(o *project.BuildOptions) { o.CurrentDirectory = "/" },
	}
	for name, change := range same {
		opts := base()
		change(&opts)
		if got := settings(opts); got != want {
			t.Errorf("%s changed the settings:\n%s\n%s", name, got, want)
		}
	}
	for name, change := range different {
		opts := base()
		change(&opts)
		if got := settings(opts); got == want {
			t.Errorf("%s did not change the settings: %s", name, got)
		}
	}
	if other, err := handoffSettingsOf(base(), "/other/tsconfig.json"); err != nil || other == want {
		t.Errorf("another tsconfig has the same settings: %s", other)
	}

	t.Setenv("GOBLOX_NO_HANDOFF", "1")
	if got := settings(base()); got != want {
		t.Errorf("GOBLOX_NO_HANDOFF changed the settings: %s", got)
	}
	if !handoffDisabledByEnv() {
		t.Error("GOBLOX_NO_HANDOFF=1 did not disable the handoff")
	}
	t.Setenv("GOBLOX_NO_HANDOFF", "0")
	if handoffDisabledByEnv() {
		t.Error("GOBLOX_NO_HANDOFF=0 disabled the handoff")
	}
	for _, name := range []string{"TYPETORCH_DEBUG_MACROS", "GOBLOX_TEST_SETTING", "TS_GO_TEST_SETTING"} { //nolint:paralleltest // t.Setenv
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, "0")
			if got := settings(base()); got == want {
				t.Errorf("%s did not change the settings", name)
			}
		})
	}
}

type caseInsensitiveFS struct{ vfs.FS }

func (caseInsensitiveFS) UseCaseSensitiveFileNames() bool { return false }

type caseSensitiveFS struct{ vfs.FS }

func (caseSensitiveFS) UseCaseSensitiveFileNames() bool { return true }

func TestHandoffSocketPath(t *testing.T) {
	t.Parallel()
	insensitive := caseInsensitiveFS{project.DefaultFS()}
	a, ok := handoffSocketPath(insensitive, "/Game/tsconfig.json")
	if !ok {
		t.Skip("no socket folder")
	}
	if len(a) > maxSocketPath {
		t.Errorf("socket path of %d bytes: %s", len(a), a)
	}
	if b, _ := handoffSocketPath(insensitive, "/game/TSCONFIG.json"); a != b {
		t.Errorf("case-insensitive paths have two sockets: %s, %s", a, b)
	}
	sensitive := caseSensitiveFS{project.DefaultFS()}
	if c, d := mustSocketPath(t, sensitive, "/Game/tsconfig.json"), mustSocketPath(t, sensitive, "/game/tsconfig.json"); c == d {
		t.Errorf("case-sensitive paths share a socket: %s", c)
	}
	if e := mustSocketPath(t, insensitive, "/other/tsconfig.json"); e == a {
		t.Errorf("two projects share a socket: %s", e)
	}
}

func mustSocketPath(t *testing.T, fs vfs.FS, tsConfigPath string) string {
	t.Helper()
	path, ok := handoffSocketPath(fs, tsConfigPath)
	if !ok {
		t.Fatal("no socket path")
	}
	return path
}

// newHandoffTestDir is a folder with a tsconfig.json, which is all a handshake needs.
func newHandoffTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	dir = filepath.ToSlash(dir)
	if err := os.WriteFile(filepath.Join(dir, "tsconfig.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func handoffTestOptions(dir string) project.BuildOptions {
	return project.BuildOptions{
		Project:          dir,
		CurrentDirectory: tspath.NormalizePath(dir),
		FS:               project.DefaultFS(),
		Overrides:        project.PartialProjectOptions{},
		Incremental:      &project.IncrementalSession{Persist: true},
	}
}

// fakeWatcher listens for dir and answers every accepted build with serve.
func fakeWatcher(t *testing.T, opts project.BuildOptions, serve func(conn net.Conn)) *handoffServer {
	t.Helper()
	server := newHandoffServer(opts, false)
	if server == nil {
		t.Skip("no handoff socket on this system")
	}
	if !server.listen() {
		t.Fatal("listen failed")
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		server.close()
		close(done)
	})
	go func() {
		for {
			select {
			case job := <-server.requests():
				serve(job.conn)
				close(job.done)
			case <-done:
				return
			}
		}
	}()
	return server
}

type handoffTestOutput struct {
	mu       sync.Mutex
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	combined bytes.Buffer
}

type handoffTestWriter struct {
	out    *handoffTestOutput
	stream *bytes.Buffer
}

func (w handoffTestWriter) Write(p []byte) (int, error) {
	w.out.mu.Lock()
	defer w.out.mu.Unlock()
	w.stream.Write(p)
	return w.out.combined.Write(p)
}

func (o *handoffTestOutput) console() consoleOutput {
	return consoleOutput{stdout: handoffTestWriter{o, &o.stdout}, stderr: handoffTestWriter{o, &o.stderr}}
}

func TestHandoffFallbacks(t *testing.T) {
	t.Parallel()

	t.Run("no watcher", func(t *testing.T) {
		t.Parallel()
		var out handoffTestOutput
		start := time.Now()
		if _, ok := handOff(handoffTestOptions(newHandoffTestDir(t)), out.console()); ok {
			t.Fatal("handed off without a watcher")
		}
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Errorf("took %v", elapsed)
		}
	})

	t.Run("stale socket", func(t *testing.T) {
		t.Parallel()
		opts := handoffTestOptions(newHandoffTestDir(t))
		server := newHandoffServer(opts, false)
		if server == nil {
			t.Skip("no handoff socket on this system")
		}
		// a watcher that was killed: its socket file stays
		listener, err := net.Listen("unix", server.path)
		if err != nil {
			t.Fatal(err)
		}
		listener.(*net.UnixListener).SetUnlinkOnClose(false)
		listener.Close()
		if _, err := os.Lstat(server.path); err != nil {
			t.Skipf("the socket file went away: %v", err)
		}
		var out handoffTestOutput
		if _, ok := handOff(opts, out.console()); ok {
			t.Fatal("handed off to a stale socket")
		}
		if !server.listen() {
			t.Fatal("a new watcher did not take over the stale socket")
		}
		server.close()
		if _, err := os.Lstat(server.path); !os.IsNotExist(err) {
			t.Errorf("close left the socket: %v", err)
		}
	})

	t.Run("second watcher", func(t *testing.T) {
		t.Parallel()
		opts := handoffTestOptions(newHandoffTestDir(t))
		fakeWatcher(t, opts, func(net.Conn) {})
		if second := newHandoffServer(opts, false); second.listen() {
			second.close()
			t.Fatal("a second watcher took the socket of a live one")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()
		if server := newHandoffServer(handoffTestOptions(newHandoffTestDir(t)), true); server.listen() || server.requests() != nil {
			t.Fatal("a disabled watcher listens")
		}
	})

	t.Run("other options", func(t *testing.T) {
		t.Parallel()
		opts := handoffTestOptions(newHandoffTestDir(t))
		fakeWatcher(t, opts, func(net.Conn) { t.Error("built with other options") })
		client := opts
		client.Overrides = project.PartialProjectOptions{"luau": true}
		var out handoffTestOutput
		if _, ok := handOff(client, out.console()); ok {
			t.Fatal("handed off with other options")
		}
	})

	t.Run("verbose with plugins", func(t *testing.T) {
		t.Parallel()
		registerHandoffPrinter()
		opts := handoffTestOptions(newHandoffTestDir(t))
		fakeWatcher(t, opts, func(net.Conn) { t.Error("built a verbose build") })
		client := opts
		client.Overrides = project.PartialProjectOptions{"verbose": true}
		var out handoffTestOutput
		if _, ok := handOff(client, out.console()); ok {
			t.Fatal("handed off a verbose build")
		}
	})

	for name, serve := range map[string]func(conn net.Conn){
		"watcher gone mid-build": func(conn net.Conn) {
			_ = writeFrame(conn, frameStdout, []byte("half"))
		},
		"fallback": func(conn net.Conn) {
			_ = writeFrame(conn, frameStdout, []byte("printed by the watcher"))
			_ = writeJSONFrame(conn, frameExit, &handoffExit{Code: 0, Fallback: "test"})
		},
		"unknown frame": func(conn net.Conn) {
			_ = writeFrame(conn, 'z', nil)
		},
		"bad exit": func(conn net.Conn) {
			_ = writeFrame(conn, frameExit, []byte("{"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			opts := handoffTestOptions(newHandoffTestDir(t))
			fakeWatcher(t, opts, serve)
			var out handoffTestOutput
			if _, ok := handOff(opts, out.console()); ok {
				t.Fatal("handed off")
			}
			if out.combined.Len() != 0 {
				t.Errorf("printed %q before falling back", out.combined.String())
			}
		})
	}

	t.Run("replay", func(t *testing.T) {
		t.Parallel()
		opts := handoffTestOptions(newHandoffTestDir(t))
		fakeWatcher(t, opts, func(conn net.Conn) {
			_ = writeFrame(conn, frameStdout, []byte("one "))
			_ = writeFrame(conn, frameStderr, []byte("two "))
			_ = writeFrame(conn, frameStdout, []byte("three"))
			_ = writeJSONFrame(conn, frameExit, &handoffExit{Code: 3})
		})
		var out handoffTestOutput
		code, ok := handOff(opts, out.console())
		if !ok || code != 3 {
			t.Fatalf("handOff = %d, %v", code, ok)
		}
		if out.combined.String() != "one two three" || out.stdout.String() != "one three" || out.stderr.String() != "two " {
			t.Errorf("printed %q (stdout %q, stderr %q)", out.combined.String(), out.stdout.String(), out.stderr.String())
		}
	})
}

// TestHandoffFrameStream: consecutive writes to a stream are one frame, and the order is kept.
func TestHandoffFrameStream(t *testing.T) {
	t.Parallel()
	server, client := net.Pipe()
	defer client.Close()
	frames := &frameStream{conn: server}
	go func() {
		defer server.Close()
		stdout, stderr := frames.writer(frameStdout), frames.writer(frameStderr)
		_, _ = io.WriteString(stdout, "a")
		_, _ = io.WriteString(stdout, "b")
		_, _ = io.WriteString(stderr, "c")
		_, _ = io.WriteString(stdout, "d")
		frames.flush()
		frames.writeJSON(frameExit, &handoffExit{Code: 1})
	}()
	var got []string
	for {
		kind, payload, err := readFrame(client)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, string(kind)+string(payload))
		if kind == frameExit {
			break
		}
	}
	if want := []string{"oab", "ec", "od", `x{"code":1}`}; !slices.Equal(got, want) {
		t.Errorf("frames %q, want %q", got, want)
	}
}

// printerPlugin prints to the process's stdout, as typetorch's logger does.
type printerPlugin struct{}

func (printerPlugin) Name() string { return "handoff-printer" }

func (printerPlugin) TransformSourceFile(_ *project.TransformContext, file *ast.SourceFile) *ast.SourceFile {
	if strings.Contains(file.Text(), "PRINT_ME") {
		_, _ = io.WriteString(os.Stdout, "[printer]: "+filepath.Base(file.FileName())+"\n")
	}
	return file
}

var registerHandoffPrinter = sync.OnceFunc(func() {
	project.RegisterPlugin("handoff-printer", func(map[string]any) (project.Plugin, error) { return printerPlugin{}, nil })
})

// handoffOracleTypes is @rbxts of the oracle's test project (GOBLOX_ORACLE, see docs/goblox/PORTING.md).
func handoffOracleTypes(t *testing.T) string {
	t.Helper()
	oracle := os.Getenv("GOBLOX_ORACLE")
	if oracle == "" {
		oracle = `C:/Users/phasenull/.t3/scratch/2026-10-29-act-as-the-implementation-sub-37576c8e/oracle`
	}
	rbxts := filepath.Join(oracle, "tests", "node_modules", "@rbxts")
	if _, err := os.Stat(filepath.Join(rbxts, "compiler-types")); err != nil {
		t.Skipf("no @rbxts/compiler-types in the oracle (set GOBLOX_ORACLE): %v", err)
	}
	return rbxts
}

const handoffProjectJSON = `{
	"name": "fixture",
	"tree": {
		"$className": "DataModel",
		"ReplicatedStorage": {
			"$className": "ReplicatedStorage",
			"include": { "$path": "include", "node_modules": { "$className": "Folder", "@rbxts": { "$path": "node_modules/@rbxts" } } },
			"shared": { "$path": "out/shared" }
		},
		"ServerScriptService": { "$className": "ServerScriptService", "server": { "$path": "out/server" } },
		"StarterPlayer": { "$className": "StarterPlayer", "StarterPlayerScripts": { "$className": "StarterPlayerScripts", "client": { "$path": "out/client" } } }
	}
}
`

func handoffTsconfig(plugins string) string {
	return `{
	"compilerOptions": {
		"allowSyntheticDefaultImports": true,
		"downlevelIteration": true,
		"module": "commonjs",
		"moduleResolution": "Node",
		"noLib": true,
		"moduleDetection": "force",
		"strict": true,
		"target": "ESNext",
		"experimentalDecorators": true,
		"typeRoots": ["node_modules/@rbxts"],
		"types": ["types", "compiler-types"],
		"rootDir": "src",
		"baseUrl": "src",
		"outDir": "out",
		"incremental": true,
		"tsBuildInfoFile": "out/tsconfig.tsbuildinfo",
		` + plugins + `
		"allowUnreachableCode": false
	},
	"include": ["src"]
}
`
}

// newHandoffProject writes a game project below a new folder.
func newHandoffProject(t *testing.T, rbxts string, plugins string) string {
	t.Helper()
	dir := t.TempDir()
	if long, err := filepath.EvalSymlinks(dir); err == nil {
		dir = long
	}
	dir = filepath.ToSlash(dir)
	for _, name := range []string{"compiler-types", "types"} {
		if err := os.CopyFS(filepath.Join(dir, "node_modules", "@rbxts", name), os.DirFS(filepath.Join(rbxts, name))); err != nil {
			t.Fatal(err)
		}
	}
	for name, text := range map[string]string{
		"package.json":         `{ "name": "fixture" }`,
		"default.project.json": handoffProjectJSON,
		"tsconfig.json":        handoffTsconfig(plugins),
		"src/shared/net.ts":    "export const PORT = 1;\nexport function connect(port: number): number {\n\treturn port + PORT;\n}\n",
		"src/server/main.server.ts": "import { connect, PORT } from \"shared/net\";\n" +
			"print(connect(PORT));\n",
		"src/client/user.client.ts": "import { connect } from \"shared/net\";\nprint(connect(2));\n",
		"src/client/leaf.ts":        "export const LEAF = 21;\n",
		"src/client/quiet.ts":       "export const QUIET = 1;\n",
	} {
		writeHandoffTestFile(t, dir, name, text)
	}
	return dir
}

func writeHandoffTestFile(t *testing.T, dir string, name string, text string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func editHandoffTestFile(t *testing.T, dir string, name string, old string, replacement string) {
	t.Helper()
	text, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), old) {
		t.Fatalf("%s has no %q:\n%s", name, old, text)
	}
	writeHandoffTestFile(t, dir, name, strings.Replace(string(text), old, replacement, 1))
}

// readOutputs reads out and include, without the build info and incremental cache files.
func readOutputs(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	for _, root := range []string{"out", "include"} {
		err := filepath.WalkDir(filepath.Join(dir, root), func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return nil
				}
				return err
			}
			if entry.IsDir() || strings.HasSuffix(path, ".tsbuildinfo") {
				return nil
			}
			text, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			files[filepath.ToSlash(relative)] = string(text)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return files
}

// testWatcher is a `goblox -w` of a project in this process: its builds, and the builds handed off to it,
// run on one goroutine as in runWatch.
type testWatcher struct {
	session *watchSession
	server  *handoffServer
	run     chan func()
}

func startTestWatcher(t *testing.T, dir string) *testWatcher {
	t.Helper()
	opts := handoffTestOptions(dir)
	server := newHandoffServer(opts, false)
	if server == nil {
		t.Skip("no handoff socket on this system")
	}
	w := &testWatcher{session: newWatchSession(opts), server: server, run: make(chan func())}
	var discard handoffTestOutput
	if _, errorCount, ok := w.session.build(w.session.opts, discard.console()); !ok || errorCount > 0 {
		t.Fatalf("the watcher's first build failed:\n%s", discard.combined.String())
	}
	if !server.listen() {
		t.Fatal("listen failed")
	}
	done := make(chan struct{})
	t.Cleanup(func() {
		server.close()
		close(done)
		w.session.opts.Incremental.WaitForSave()
	})
	go func() {
		for {
			select {
			case job := <-server.requests():
				w.session.handOff(job)
			case f := <-w.run:
				f()
			case <-done:
				return
			}
		}
	}()
	return w
}

// do runs f on the watch loop.
func (w *testWatcher) do(f func()) {
	finished := make(chan struct{})
	w.run <- func() {
		defer close(finished)
		f()
	}
	<-finished
}

// rebuild is a build of the watcher's own, after it saw a change.
func (w *testWatcher) rebuild() {
	w.do(func() {
		var discard handoffTestOutput
		w.session.build(w.session.opts, discard.console())
	})
}

func (w *testWatcher) idle() {
	w.do(w.session.opts.Incremental.WaitForSave)
}

type buildRun struct {
	code   int
	stdout string
	stderr string
}

// buildInProcess is `goblox build` without a watcher.
func buildInProcess(dir string) buildRun {
	var out handoffTestOutput
	restore := project.LogService.Redirect(out.console().stdout)
	code := buildExitCode(buildOnce(handoffTestOptions(dir), out.console()))
	restore()
	return buildRun{code: code, stdout: out.stdout.String(), stderr: out.stderr.String()}
}

// TestHandoffEquivalence builds one project with a watcher and handed-off builds, and a copy of it with
// one-shot builds in this process, through the same changes: the outputs, what the builds print and their
// exit codes have to be the same.
func TestHandoffEquivalence(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	rbxts := handoffOracleTypes(t)
	watched := newHandoffProject(t, rbxts, "")
	oneShot := newHandoffProject(t, rbxts, "")
	if run := buildInProcess(oneShot); run.code != 0 {
		t.Fatalf("first one-shot build: %+v", run)
	}
	w := startTestWatcher(t, watched)

	sameSize := func(t *testing.T, dir string) {
		path := filepath.Join(dir, "src", "shared", "net.ts")
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		editHandoffTestFile(t, dir, "src/shared/net.ts", "PORT = 1;", "PORT = 7;")
		if err := os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
			t.Fatal(err)
		}
		if after, err := os.Stat(path); err != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
			t.Fatalf("the edit changed the size or the time: %v", err)
		}
	}

	steps := []struct {
		name    string
		apply   func(t *testing.T, dir string)
		rebuild bool
		code    int
		check   func(t *testing.T, outputs map[string]string)
	}{
		{name: "no change", apply: func(*testing.T, string) {}},
		{name: "edit", apply: func(t *testing.T, dir string) {
			editHandoffTestFile(t, dir, "src/client/quiet.ts", "QUIET = 1", "QUIET = 2")
		}},
		{name: "watcher rebuild, then handoff", rebuild: true, apply: func(t *testing.T, dir string) {
			editHandoffTestFile(t, dir, "src/client/leaf.ts", "LEAF = 21", "LEAF = 22")
		}},
		{name: "error", code: 1, apply: func(t *testing.T, dir string) {
			editHandoffTestFile(t, dir, "src/client/leaf.ts", "export const LEAF", "const BAD: number = \"bad\";\nexport const LEAF")
		}},
		{name: "error again", code: 1, apply: func(*testing.T, string) {}},
		{name: "error, watcher rebuild", code: 1, rebuild: true, apply: func(t *testing.T, dir string) {
			writeHandoffTestFile(t, dir, "src/client/quiet.ts", "export const QUIET = 3;\n")
		}},
		{name: "fixed", apply: func(t *testing.T, dir string) {
			editHandoffTestFile(t, dir, "src/client/leaf.ts", "const BAD: number = \"bad\";\n", "")
		}},
		{name: "deleted file", apply: func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "src", "client", "quiet.ts")); err != nil {
				t.Fatal(err)
			}
		}, check: func(t *testing.T, outputs map[string]string) {
			if _, ok := outputs["out/client/quiet.luau"]; ok {
				t.Error("the output of the deleted file is still there")
			}
		}},
		{name: "added file", apply: func(t *testing.T, dir string) {
			writeHandoffTestFile(t, dir, "src/client/added.ts", "export const ADDED = 1;\n")
		}},
		{name: "rm -rf out", apply: func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, "out")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "rm -rf include", apply: func(t *testing.T, dir string) {
			if err := os.RemoveAll(filepath.Join(dir, "include")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "edited output", apply: func(t *testing.T, dir string) {
			writeHandoffTestFile(t, dir, "out/client/leaf.luau", "-- edited\n")
		}},
		{name: "same size and time", apply: sameSize, check: func(t *testing.T, outputs map[string]string) {
			if !strings.Contains(outputs["out/shared/net.luau"], "PORT = 7") {
				t.Errorf("the change was missed:\n%s", outputs["out/shared/net.luau"])
			}
		}},
		{name: "no change at the end", apply: func(*testing.T, string) {}},
	}
	for _, step := range steps {
		w.idle()
		step.apply(t, watched)
		step.apply(t, oneShot)
		if step.rebuild {
			w.rebuild()
			w.idle()
		}

		want := buildInProcess(oneShot)
		var out handoffTestOutput
		code, handedOff := handOff(handoffTestOptions(watched), out.console())
		if !handedOff {
			t.Fatalf("%s: not handed off", step.name)
		}
		got := buildRun{code: code, stdout: out.stdout.String(), stderr: out.stderr.String()}
		if got != want {
			t.Errorf("%s: handed off %+v, in process %+v", step.name, got, want)
		}
		if want.code != step.code {
			t.Errorf("%s: exit code %d, want %d", step.name, want.code, step.code)
		}
		gotOutputs, wantOutputs := readOutputs(t, watched), readOutputs(t, oneShot)
		if !maps.Equal(gotOutputs, wantOutputs) {
			for name := range gotOutputs {
				if gotOutputs[name] != wantOutputs[name] {
					t.Errorf("%s: %s is %q, in process %q", step.name, name, gotOutputs[name], wantOutputs[name])
				}
			}
			for name := range wantOutputs {
				if _, ok := gotOutputs[name]; !ok {
					t.Errorf("%s: no %s", step.name, name)
				}
			}
		}
		if step.check != nil {
			step.check(t, gotOutputs)
		}
	}
}

// TestHandoffPluginOutput: a plugin that prints sends the build back to the client, which then compiles what
// the plugin printed about again.
func TestHandoffPluginOutput(t *testing.T) { //nolint:paralleltest // Build uses the global LogService and DiagnosticService.
	rbxts := handoffOracleTypes(t)
	registerHandoffPrinter()
	dir := newHandoffProject(t, rbxts, `"plugins": [{ "transform": "handoff-printer" }],`)
	w := startTestWatcher(t, dir)

	// nothing printed: handed off
	editHandoffTestFile(t, dir, "src/client/leaf.ts", "LEAF = 21", "LEAF = 22")
	var out handoffTestOutput
	if code, ok := handOff(handoffTestOptions(dir), out.console()); !ok || code != 0 {
		t.Fatalf("handOff = %d, %v: %s", code, ok, out.combined.String())
	}

	w.idle()
	editHandoffTestFile(t, dir, "src/client/leaf.ts", "LEAF = 22", "LEAF = 23; // PRINT_ME")
	if _, ok := handOff(handoffTestOptions(dir), out.console()); ok {
		t.Fatal("handed off a build whose plugin printed")
	}
	if out.combined.Len() != 0 {
		t.Errorf("printed %q", out.combined.String())
	}
	w.idle()
	// the client's build has to run the plugin on leaf.ts again
	matches, _ := filepath.Glob(filepath.Join(dir, "out", "*.goblox.tsbuildinfo"))
	if len(matches) != 0 {
		t.Errorf("the incremental state is still there: %v", matches)
	}
}

// TestHandoffSettingsJSON: the request survives the round trip, overrides included.
func TestHandoffSettingsJSON(t *testing.T) {
	t.Parallel()
	request := handoffRequest{Version: "1", Identity: "id", Settings: "{}", Project: "/game", Overrides: project.PartialProjectOptions{"verbose": true, "rojo": "/game/default.project.json"}}
	text, err := json.Marshal(&request)
	if err != nil {
		t.Fatal(err)
	}
	var back handoffRequest
	if err := json.Unmarshal(text, &back); err != nil {
		t.Fatal(err)
	}
	if back.Overrides["verbose"] != true || back.Overrides["rojo"] != "/game/default.project.json" || back.Project != "/game" {
		t.Errorf("round trip: %+v", back)
	}
}
