package app_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JaimeStill/spike-graph/internal/app"
	"github.com/JaimeStill/spike-graph/internal/config/configtest"
	"github.com/JaimeStill/spike-graph/sdk/graph"
)

// The suite is hermetic: no live database or object store exists, so it
// proves the cold start and the startup contract: the Build performs no
// I/O, and a failed start of the connections fails startup before the
// schema starts, instead of serving unready. The serve-probes-drain path
// and every behavior that needs a real engine are the integration tier's,
// the root integration package.

// failsafe bounds every wait for an event that should occur, so a broken
// composition fails the test instead of hanging it.
const failsafe = 2 * time.Second

// syncBuffer serializes writes so the app's logging goroutines and the
// test's reads stay race-free.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// New and Run's Build perform no I/O — they succeed with nothing
// listening — and Run then fails startup at the connections' layer,
// draining to exit 1 with the failure named in the log, never flipping
// readiness. The database, the object store, and the broker start
// together, and the first to fail cancels the others, so the log names
// whichever failed first. The failure is the log's alone: nothing goes to
// stderr, which only a Build failure reaches.
func TestRun_FailsStartupWithoutInfrastructure(t *testing.T) {
	buf := &syncBuffer{}
	stderr := &syncBuffer{}
	a := app.New(configtest.Config(t), buf)

	done := make(chan int, 1)
	go func() { done <- a.Run(context.Background(), stderr) }()

	select {
	case code := <-done:
		if code != 1 {
			t.Errorf("Run = %d, want 1 on a failed startup", code)
		}
	case <-time.After(failsafe):
		t.Fatal("timed out waiting for Run to fail startup")
	}

	out := buf.String()
	if !strings.Contains(out, "startup: database") && !strings.Contains(out, "startup: storage") && !strings.Contains(out, "startup: broker") {
		t.Errorf("failure log names no connection: %q", out)
	}
	if strings.Contains(out, "server ready") {
		t.Error("log carries a ready record despite the failed startup")
	}
	if got := stderr.String(); got != "" {
		t.Errorf("stderr = %q, want nothing on a startup failure", got)
	}
}

// A constructor's error fails Run's Build: Run returns 1 and writes the
// failure to stderr, "service failed: " and the error labelled with the
// failing node's name, and logs nothing, since the node that failed may be
// the logger itself. Nothing reports ready.
func TestRun_BuildFailureOnStderr(t *testing.T) {
	log := &syncBuffer{}
	stderr := &syncBuffer{}
	a := app.New(configtest.Config(t), log)
	a.Graph().Replace(a.Nodes().Logger, func(*graph.Scope) (*slog.Logger, error) {
		return nil, errors.New("logger unavailable")
	})

	// Cancelled up front, so a Build that wrongly succeeds drains at once
	// and fails the exit-code check instead of hanging.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := a.Run(ctx, stderr); code != 1 {
		t.Errorf("Run = %d, want 1", code)
	}
	failure := regexp.MustCompile(`^service failed: [^\n]*logger[^\n]*logger unavailable\n$`)
	if got := stderr.String(); !failure.MatchString(got) {
		t.Errorf("stderr = %q, want a service failure naming the logger node and its error", got)
	}
	if out := log.String(); out != "" {
		t.Errorf("log = %q, want nothing", out)
	}
}

// An App runs once. Run reports a Build or lifecycle failure as an exit
// code, but a second Run is a programming error and panics before it
// builds anything: the graph observes no node in it.
func TestRun_TwicePanics(t *testing.T) {
	a := app.New(configtest.Config(t), &syncBuffer{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := a.Run(ctx, io.Discard); code != 0 {
		t.Fatalf("first Run = %d, want 0 on a context ended before startup", code)
	}

	var built []string
	a.Graph().Observe(func(r graph.Ref) { built = append(built, r.Name()) })
	defer func() {
		if recover() == nil {
			t.Error("a second Run did not panic")
		}
		if len(built) != 0 {
			t.Errorf("the second Run built %v before it panicked, want nothing", built)
		}
	}()
	_ = a.Run(ctx, io.Discard)
}
