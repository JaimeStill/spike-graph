package web_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"

	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/lifecycle"
	"github.com/JaimeStill/spike-graph/sdk/web"
)

// failsafe bounds every wait for an event that should occur, so a broken
// runner fails the test instead of hanging it.
const failsafe = 2 * time.Second

// records is the logger node's writer: each record the handler writes is
// one line, sent on a channel as it is written and kept for the whole log.
type records struct {
	mu    sync.Mutex
	buf   bytes.Buffer
	lines chan string
}

func newRecords() *records { return &records{lines: make(chan string, 16)} }

func (r *records) Write(p []byte) (int, error) {
	r.mu.Lock()
	r.buf.Write(p)
	r.mu.Unlock()
	r.lines <- string(p)
	return len(p), nil
}

func (r *records) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// next returns the next record logged, or fails the test.
func (r *records) next(t *testing.T) string {
	t.Helper()
	select {
	case line := <-r.lines:
		return line
	case <-time.After(failsafe):
		t.Fatalf("timed out waiting for a record; log so far: %q", r.String())
		return ""
	}
}

// server is the server node's value: it binds 127.0.0.1 on an assigned
// port at Start, fails Start with startErr when one is set, and reports a
// runtime failure on Err.
type server struct {
	startErr error
	errs     chan error

	mu sync.Mutex
	ln net.Listener
}

func (s *server) Start(context.Context) error {
	if s.startErr != nil {
		return s.startErr
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()
	return nil
}

func (s *server) Shutdown(context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return nil
	}
	return s.ln.Close()
}

func (s *server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return "127.0.0.1:0"
	}
	return s.ln.Addr().String()
}

func (s *server) Err() <-chan error { return s.errs }

// service is a small graph in a service's shape: a logger node over w, a
// server node that uses it, so the server is the only root, and a lifecycle
// config node.
type service struct {
	graph           *graph.Graph
	lifecycleConfig *graph.Node[lifecycle.Config]
	logger          *graph.Node[*slog.Logger]
	server          *graph.Node[*server]
	value           *server
}

func newService(w io.Writer) *service {
	s := &service{graph: graph.New(), value: &server{errs: make(chan error, 1)}}
	s.lifecycleConfig = s.graph.Value("lifecycle config", lifecycle.Config{ShutdownTimeout: config.Duration(failsafe)})
	s.logger = s.graph.Define("logger", func(*graph.Scope) (*slog.Logger, error) {
		return slog.New(slog.NewTextHandler(w, nil)), nil
	})
	s.server = s.graph.Define("server", func(sc *graph.Scope) (*server, error) {
		sc.Use(s.logger)
		return s.value, nil
	})
	return s
}

// run starts web.Run on s in a goroutine and returns the channel its exit
// code arrives on.
func (s *service) run(ctx context.Context, stderr io.Writer) <-chan int {
	done := make(chan int, 1)
	go func() {
		done <- web.Run(ctx, s.graph, s.lifecycleConfig, s.logger, s.server, []graph.Ref{s.server}, stderr)
	}()
	return done
}

// exit returns the exit code from done, or fails the test.
func exit(t *testing.T, done <-chan int) int {
	t.Helper()
	select {
	case code := <-done:
		return code
	case <-time.After(failsafe):
		t.Fatal("timed out waiting for Run to return")
		return 0
	}
}

// Once startup completes, Run logs "server ready" once with the address
// the server bound, the assigned port of a port 0; when the context ends
// and the System drains cleanly, it logs "server stopped" and returns 0.
// Nothing goes to stderr.
func TestRun_ReadyThenCleanStop(t *testing.T) {
	log := newRecords()
	var stderr bytes.Buffer
	s := newService(log)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := s.run(ctx, &stderr)

	ready := log.next(t)
	addr := s.value.Addr()
	if strings.HasSuffix(addr, ":0") {
		t.Fatalf("server Addr = %q after ready, want an assigned port", addr)
	}
	if !strings.Contains(ready, `msg="server ready"`) || !strings.Contains(ready, "addr="+addr) {
		t.Errorf("first record = %q, want server ready with addr=%s", ready, addr)
	}

	cancel()
	if code := exit(t, done); code != 0 {
		t.Errorf("Run = %d, want 0 on a clean drain", code)
	}
	if out := log.String(); strings.Count(out, "server ready") != 1 || !strings.Contains(out, `msg="server stopped"`) {
		t.Errorf("log = %q, want one ready record and a stopped record", out)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing", stderr.String())
	}
}

// A failed startup logs "service failed" with an error naming the failing
// node and returns 1. Nothing reports ready.
func TestRun_StartupFailureExitsOne(t *testing.T) {
	log := newRecords()
	var stderr bytes.Buffer
	s := newService(log)
	s.value.startErr = errors.New("bind refused")

	if code := exit(t, s.run(context.Background(), &stderr)); code != 1 {
		t.Errorf("Run = %d, want 1 on a failed startup", code)
	}
	out := log.String()
	if !strings.Contains(out, `msg="service failed"`) || !strings.Contains(out, "startup: server: bind refused") {
		t.Errorf("log = %q, want a service failure naming the server node and its error", out)
	}
	if strings.Contains(out, "server ready") {
		t.Errorf("log = %q, want no ready record", out)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing", stderr.String())
	}
}

// A monitored participant's runtime failure ends the run: Run logs
// "service failed" with the error and returns 1 after the ready record.
// The candidate lifecycle wraps a monitored failure "run:" without the
// node's name, so the record carries the participant's own error.
func TestRun_RuntimeFailureExitsOne(t *testing.T) {
	log := newRecords()
	var stderr bytes.Buffer
	s := newService(log)
	done := s.run(context.Background(), &stderr)

	if ready := log.next(t); !strings.Contains(ready, `msg="server ready"`) {
		t.Fatalf("first record = %q, want server ready", ready)
	}
	s.value.errs <- errors.New("listener lost")

	if code := exit(t, done); code != 1 {
		t.Errorf("Run = %d, want 1 on a runtime failure", code)
	}
	out := log.String()
	if !strings.Contains(out, `msg="service failed"`) || !strings.Contains(out, "run: listener lost") {
		t.Errorf("log = %q, want a service failure carrying the runtime error", out)
	}
	if strings.Contains(out, "server stopped") {
		t.Errorf("log = %q, want no stopped record after a failure", out)
	}
}

// A constructor's error fails the Build: Run writes "service failed:" and
// the error, naming the failing node, to stderr and returns 1. It logs
// nothing, since the node that failed may be the logger, and starts
// nothing, so nothing reports ready.
func TestRun_BuildFailureOnStderr(t *testing.T) {
	log := newRecords()
	var stderr bytes.Buffer
	s := newService(log)
	s.graph.Replace(s.logger, func(*graph.Scope) (*slog.Logger, error) {
		return nil, errors.New("logger unavailable")
	})

	// Cancelled up front, so a Build that wrongly succeeds drains at once
	// and fails the exit-code check instead of hanging.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := exit(t, s.run(ctx, &stderr)); code != 1 {
		t.Errorf("Run = %d, want 1 on a failed Build", code)
	}
	if got, want := stderr.String(), "service failed: logger: logger unavailable\n"; got != want {
		t.Errorf("stderr = %q, want %q, naming the logger node and its error", got, want)
	}
	if out := log.String(); out != "" {
		t.Errorf("log = %q, want nothing", out)
	}
	if s.value.Addr() != "127.0.0.1:0" {
		t.Error("the server started despite the failed Build")
	}
}
