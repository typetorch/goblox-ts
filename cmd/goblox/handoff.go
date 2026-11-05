package main

// Build handoff: a one-shot `goblox build` asks a running `goblox -w` of the same project to run the build,
// so it reuses the watcher's parsed and bound files and its incremental state instead of starting cold.
//
// Every watcher listens on a local socket named after its project's tsconfig (handoffSocketPath). A build is
// handed off only when the watcher builds exactly what this process would: the same executable, the same
// tsconfig and working directory, the same options except the ones that only change what is printed, the
// same colors and the same GOBLOX_*, TYPETORCH_* and TS_GO_* variables (handoffSettings). The watcher runs
// the build between its own builds, comparing the text of every file it would reuse with the disk's, and
// sends what the build printed and its exit code. The client prints that only once the build is over, so
// in every other case (no watcher, another goblox, other options, a watcher that went away mid-build, a
// plugin that wrote to the console) nothing has been printed yet and the client builds in its own process,
// as without a watcher.

import (
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/microsoft/typescript-go/internal/json"
	"github.com/microsoft/typescript-go/internal/roblox/diagnostics"
	"github.com/microsoft/typescript-go/internal/roblox/project"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs"
)

const (
	// handoffDialTimeout is how long a build waits for a watcher to answer the connection
	handoffDialTimeout = 200 * time.Millisecond
	// handoffAnswerTimeout bounds the handshake; the build itself, and the wait for the watcher's own build
	// to finish, have no limit
	handoffAnswerTimeout = 2 * time.Second
	// handoffWriteTimeout gives up on a client that stopped reading
	handoffWriteTimeout = 10 * time.Second
	handoffMaxFrame     = 64 << 20
	// maxSocketPath is below the smallest sun_path (104 bytes on macOS, with its NUL)
	maxSocketPath = 100
)

// A frame is its kind, the big-endian length of its payload and the payload.
const (
	frameRequest byte = 'q' // client: handoffRequest
	frameAnswer  byte = 'a' // watcher: handoffAnswer
	frameStdout  byte = 'o' // watcher: what the build wrote to stdout
	frameStderr  byte = 'e' // watcher: what the build wrote to stderr
	frameExit    byte = 'x' // watcher: handoffExit, the last frame
)

type handoffRequest struct {
	Version  string `json:"version"`
	Identity string `json:"identity"`
	// Settings is handoffSettingsOf's text, which has to be the watcher's
	Settings string `json:"settings"`
	// Project and Overrides are the build's (Overrides also with the options that only change printing)
	Project   string                        `json:"project"`
	Overrides project.PartialProjectOptions `json:"overrides"`
}

type handoffAnswer struct {
	Accepted bool `json:"accepted"`
	// Reason says why not, for debugging
	Reason string `json:"reason,omitzero"`
}

type handoffExit struct {
	Code int `json:"code"`
	// Fallback, when set, says why the client has to build itself after all
	Fallback string `json:"fallback,omitzero"`
}

// handoffSettings is everything outside the project's files that a build's outputs, diagnostics and exit
// code depend on.
type handoffSettings struct {
	TsConfigPath     string                        `json:"tsConfigPath"`
	CurrentDirectory string                        `json:"currentDirectory"`
	Overrides        project.PartialProjectOptions `json:"overrides"`
	Checkers         int                           `json:"checkers"`
	Incremental      bool                          `json:"incremental"`
	EmitFirst        bool                          `json:"emitFirst"`
	// Color is baked into diagnostic messages
	Color bool              `json:"color"`
	Env   map[string]string `json:"env"`
}

// printOnlyOverrides change what a build prints, or whether it rewrites an output with the bytes it already
// has, not the outputs' bytes. The handed-off build uses the client's.
var printOnlyOverrides = map[string]bool{"verbose": true, "writeOnlyChanged": true}

// handoffEnvPrefixes are the variables goblox, its plugins and typescript-go read.
var handoffEnvPrefixes = []string{"GOBLOX_", "TYPETORCH_", "TS_GO_"}

func handoffSettingsOf(opts project.BuildOptions, tsConfigPath string) (string, error) {
	overrides := project.PartialProjectOptions{}
	for name, value := range opts.Overrides {
		if !printOnlyOverrides[name] {
			overrides[name] = value
		}
	}
	settings := handoffSettings{
		TsConfigPath:     tsConfigPath,
		CurrentDirectory: opts.CurrentDirectory,
		Overrides:        overrides,
		Checkers:         opts.Checkers,
		Incremental:      opts.Incremental != nil,
		EmitFirst:        opts.EmitFirst,
		Color:            diagnostics.ColorEnabled,
		Env:              handoffEnv(),
	}
	text, err := json.Marshal(&settings, json.Deterministic(true))
	return string(text), err
}

func handoffEnv() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		name, value, _ := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		if name == "GOBLOX_NO_HANDOFF" {
			continue
		}
		for _, prefix := range handoffEnvPrefixes {
			if strings.HasPrefix(name, prefix) {
				env[name] = value
			}
		}
	}
	return env
}

// handoffDisabledByEnv is GOBLOX_NO_HANDOFF=1.
func handoffDisabledByEnv() bool {
	value := os.Getenv("GOBLOX_NO_HANDOFF")
	return value != "" && value != "0"
}

// handoffSocketPath is the socket of the watchers of tsConfigPath, in a folder only this user can use
// (handoffDir). ok is false when there is no such folder.
func handoffSocketPath(fs vfs.FS, tsConfigPath string) (path string, ok bool) {
	dir, ok := handoffDir()
	if !ok {
		return "", false
	}
	sum := sha256.Sum256([]byte(tspath.GetCanonicalFileName(tsConfigPath, fs.UseCaseSensitiveFileNames())))
	path = filepath.Join(dir, hex.EncodeToString(sum[:8])+".sock")
	return path, len(path) <= maxSocketPath
}

func writeFrame(w io.Writer, kind byte, payload []byte) error {
	frame := make([]byte, 5, 5+len(payload))
	frame[0] = kind
	binary.BigEndian.PutUint32(frame[1:], uint32(len(payload)))
	_, err := w.Write(append(frame, payload...))
	return err
}

func writeJSONFrame(w io.Writer, kind byte, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeFrame(w, kind, payload)
}

func readFrame(r io.Reader) (kind byte, payload []byte, err error) {
	var header [5]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return 0, nil, err
	}
	size := binary.BigEndian.Uint32(header[1:])
	if size > handoffMaxFrame {
		return 0, nil, fmt.Errorf("handoff frame of %d bytes", size)
	}
	payload = make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return 0, nil, err
	}
	return header[0], payload, nil
}

// handOff asks a watcher of the project to run the build and prints what it printed. handedOff is false when
// this process has to build instead; nothing was printed then.
func handOff(opts project.BuildOptions, out consoleOutput) (code int, handedOff bool) {
	tsConfigPath, err := project.FindTsConfigPath(opts.FS, opts.CurrentDirectory, opts.Project)
	if err != nil {
		return 0, false
	}
	path, ok := handoffSocketPath(opts.FS, tsConfigPath)
	if !ok {
		return 0, false
	}
	settings, err := handoffSettingsOf(opts, tsConfigPath)
	if err != nil {
		return 0, false
	}
	conn, err := net.DialTimeout("unix", path, handoffDialTimeout)
	if err != nil {
		return 0, false
	}
	defer conn.Close()

	request := handoffRequest{Version: version, Identity: project.BuildIdentity(), Settings: settings, Project: opts.Project, Overrides: opts.Overrides}
	if err := conn.SetDeadline(time.Now().Add(handoffAnswerTimeout)); err != nil {
		return 0, false
	}
	if err := writeJSONFrame(conn, frameRequest, &request); err != nil {
		return 0, false
	}
	reader := bufio.NewReader(conn)
	kind, payload, err := readFrame(reader)
	var answer handoffAnswer
	if err != nil || kind != frameAnswer || json.Unmarshal(payload, &answer) != nil || !answer.Accepted {
		return 0, false
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return 0, false
	}

	type chunk struct {
		kind byte
		data []byte
	}
	var printed []chunk
	for {
		kind, payload, err = readFrame(reader)
		if err != nil {
			// the watcher went away: build here, from what is on disk
			return 0, false
		}
		switch kind {
		case frameStdout, frameStderr:
			printed = append(printed, chunk{kind, payload})
		case frameExit:
			var exit handoffExit
			if json.Unmarshal(payload, &exit) != nil || exit.Fallback != "" {
				return 0, false
			}
			for _, c := range printed {
				if c.kind == frameStdout {
					_, _ = out.stdout.Write(c.data)
				} else {
					_, _ = out.stderr.Write(c.data)
				}
			}
			return exit.Code, true
		default:
			return 0, false
		}
	}
}

// handoffServer is the watcher's side.
type handoffServer struct {
	settings string
	identity string
	path     string
	listener net.Listener
	jobs     chan *handoffJob
}

type handoffJob struct {
	conn    net.Conn
	request handoffRequest
	done    chan struct{}
}

// newHandoffServer prepares the endpoint of a watcher of opts, which listen opens. It is nil when the
// watcher should not have one.
func newHandoffServer(opts project.BuildOptions, disabled bool) *handoffServer {
	if disabled {
		return nil
	}
	tsConfigPath, err := project.FindTsConfigPath(opts.FS, opts.CurrentDirectory, opts.Project)
	if err != nil {
		return nil
	}
	path, ok := handoffSocketPath(opts.FS, tsConfigPath)
	if !ok {
		return nil
	}
	settings, err := handoffSettingsOf(opts, tsConfigPath)
	if err != nil {
		return nil
	}
	return &handoffServer{settings: settings, identity: project.BuildIdentity(), path: path, jobs: make(chan *handoffJob)}
}

// listen opens the socket, unless another live watcher of the project has it. The socket of a watcher that
// is gone is removed first.
func (s *handoffServer) listen() bool {
	if s == nil {
		return false
	}
	if _, err := os.Lstat(s.path); err == nil {
		if conn, err := net.DialTimeout("unix", s.path, handoffDialTimeout); err == nil {
			conn.Close()
			return false
		}
		if err := os.Remove(s.path); err != nil {
			return false
		}
	}
	listener, err := net.Listen("unix", s.path)
	if err != nil {
		return false
	}
	// the folder is the user's already
	_ = os.Chmod(s.path, 0o600)
	s.listener = listener
	go s.accept()
	return true
}

// close removes the socket.
func (s *handoffServer) close() {
	if s != nil && s.listener != nil {
		s.listener.Close()
	}
}

// requests receives the accepted builds; nil (never) without a socket.
func (s *handoffServer) requests() <-chan *handoffJob {
	if s == nil || s.listener == nil {
		return nil
	}
	return s.jobs
}

// closeOnSignal removes the socket when the watcher is interrupted, and then ends it as the signal would
// have.
func (s *handoffServer) closeOnSignal() {
	if s == nil || s.listener == nil {
		return
	}
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, exitSignals...)
	go func() {
		sig := <-signals
		s.close()
		exitOnSignal(sig)
	}()
}

func (s *handoffServer) accept() {
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		go s.handle(conn)
	}
}

func (s *handoffServer) handle(conn net.Conn) {
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(handoffAnswerTimeout)); err != nil {
		return
	}
	kind, payload, err := readFrame(conn)
	if err != nil || kind != frameRequest {
		return
	}
	var request handoffRequest
	if err := json.Unmarshal(payload, &request); err != nil {
		return
	}
	reason := s.check(&request)
	if err := writeJSONFrame(conn, frameAnswer, &handoffAnswer{Accepted: reason == "", Reason: reason}); err != nil || reason != "" {
		return
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return
	}
	job := &handoffJob{conn: conn, request: request, done: make(chan struct{})}
	s.jobs <- job
	<-job.done
}

// check is why the watcher can't build request as the client would, "" if it can.
func (s *handoffServer) check(request *handoffRequest) string {
	switch {
	case request.Version != version:
		return "the watcher is goblox " + version
	case request.Identity != s.identity:
		return "the watcher is another goblox executable"
	case request.Settings != s.settings:
		return "the watcher has other options: " + s.settings
	case request.Overrides["verbose"] == true && len(project.RegisteredPlugins()) > 0:
		// plugins log to the process's stdout, and in color only on a terminal: their verbose lines can't
		// reach the client as they would have printed
		return "verbose builds with plugins are not handed off"
	}
	return ""
}

// handOff runs a handed-off build on the watch loop, between the watcher's own builds.
func (w *watchSession) handOff(job *handoffJob) {
	defer close(job.done)
	reportWatchStatus("Build requested by goblox build.")

	frames := &frameStream{conn: job.conn}
	out := consoleOutput{stdout: frames.writer(frameStdout), stderr: frames.writer(frameStderr)}
	opts := w.opts
	opts.Project = job.request.Project
	opts.Overrides = job.request.Overrides
	if opts.Overrides == nil {
		opts.Overrides = project.PartialProjectOptions{}
	}
	// what this build reads goes to no watch list: the watcher's next poll compares with its own build's
	opts.Watch = nil
	// a cached file whose time and size did not change can still have other text
	opts.VerifySourceFiles = true
	if opts.Incremental != nil {
		// the state is saved before the build returns, as in a one-shot build: the cache file is written
		// when the client exits, and what the save logs is the client's too
		background := opts.Incremental.Background
		opts.Incremental.Background = false
		defer func() { opts.Incremental.Background = background }()
	}

	exit := handoffExit{}
	capture, err := captureConsole()
	if err != nil {
		exit.Fallback = "could not capture the console: " + err.Error()
	} else {
		restoreLog := project.LogService.Redirect(out.stdout)
		exit.Code = buildExitCode(w.build(opts, out))
		restoreLog()
		if capture.stop() {
			// what a plugin printed went to this process's console, not the client's. The client builds
			// again, and has to compile everything for the plugin to print it again: the state this build
			// left would have it compile nothing.
			exit.Fallback = "a plugin wrote to the console"
			if opts.Incremental != nil {
				opts.Incremental.Forget()
			}
		}
	}
	frames.flush()
	frames.writeJSON(frameExit, &exit)
}

// frameStream sends what a build prints as frames, in order, merging consecutive writes to one stream.
// It drops what it can't send: the client then builds itself.
type frameStream struct {
	mu   sync.Mutex
	conn net.Conn
	kind byte
	buf  []byte
	err  error
}

type frameWriter struct {
	stream *frameStream
	kind   byte
}

func (w frameWriter) Write(p []byte) (int, error) {
	s := w.stream
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.kind != w.kind {
		s.flushLocked()
		s.kind = w.kind
	}
	s.buf = append(s.buf, p...)
	if len(s.buf) >= 64<<10 {
		s.flushLocked()
	}
	return len(p), nil
}

func (s *frameStream) writer(kind byte) io.Writer {
	return frameWriter{stream: s, kind: kind}
}

func (s *frameStream) flushLocked() {
	if len(s.buf) == 0 {
		return
	}
	if s.err == nil {
		s.err = s.conn.SetWriteDeadline(time.Now().Add(handoffWriteTimeout))
	}
	if s.err == nil {
		s.err = writeFrame(s.conn, s.kind, s.buf)
	}
	s.buf = s.buf[:0]
}

func (s *frameStream) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushLocked()
}

func (s *frameStream) writeJSON(kind byte, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err == nil {
		s.err = s.conn.SetWriteDeadline(time.Now().Add(handoffWriteTimeout))
	}
	if s.err == nil {
		s.err = writeJSONFrame(s.conn, kind, value)
	}
}

// consoleCapture stands in for the process's stdout and stderr during a handed-off build. goblox itself
// prints to the client; anything else that prints (a plugin's logger) lands here.
type consoleCapture struct {
	stdout, stderr *os.File
	pipes          []*os.File
	wrote          atomic.Bool
	done           sync.WaitGroup
}

func captureConsole() (*consoleCapture, error) {
	c := &consoleCapture{stdout: os.Stdout, stderr: os.Stderr}
	for range 2 {
		reader, writer, err := os.Pipe()
		if err != nil {
			for _, pipe := range c.pipes {
				pipe.Close()
			}
			c.done.Wait()
			return nil, err
		}
		c.pipes = append(c.pipes, writer)
		c.done.Go(func() {
			if n, _ := io.Copy(io.Discard, reader); n > 0 {
				c.wrote.Store(true)
			}
			reader.Close()
		})
	}
	os.Stdout, os.Stderr = c.pipes[0], c.pipes[1]
	return c, nil
}

// stop puts the console back and reports whether anything was printed to it.
func (c *consoleCapture) stop() bool {
	os.Stdout, os.Stderr = c.stdout, c.stderr
	for _, pipe := range c.pipes {
		pipe.Close()
	}
	c.done.Wait()
	return c.wrote.Load()
}
