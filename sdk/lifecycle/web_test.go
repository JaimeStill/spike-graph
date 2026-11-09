package lifecycle_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/standards-lab/go-core/config"
	web "github.com/standards-lab/go-web-sdk"

	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/lifecycle"
)

// Both of the forked readiness sources are go-web-sdk v0.15.2's Doctor, so
// its RegisterHealth takes either. The assertion lives in a test file so the
// package itself does not import go-web-sdk.
var (
	_ web.Doctor = (*lifecycle.Coordinator)(nil)
	_ web.Doctor = (*lifecycle.Readiness)(nil)
)

// get returns the status of a GET of url, or fails the test.
func get(t *testing.T, url string) int {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

// A published go-web-sdk Server, written against go-core's lifecycle
// interfaces, takes part in the forked Coordinator unchanged: it starts and
// serves, the /readyz RegisterHealth mounts over a Readiness node reports
// the Coordinator, and it shuts down when Run's context ends.
func TestRun_ServesAGoWebSDKServer(t *testing.T) {
	g := graph.New()
	readiness := graph.Fresh[lifecycle.Readiness](g, "readiness")
	server := g.Define("server", func(s *graph.Scope) (*web.Server, error) {
		mux := http.NewServeMux()
		web.RegisterHealth(mux, s.Use(readiness), web.Problem{})
		cfg := web.Config{Host: "127.0.0.1", Port: new(0)}
		if err := cfg.Finalize(""); err != nil {
			return nil, err
		}
		return web.NewServer(cfg, mux, slog.New(slog.DiscardHandler)), nil
	})
	sys, err := g.Build(server)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	lc := lifecycle.New(sys, lifecycle.Config{ShutdownTimeout: config.Duration(failsafe)})
	srv := sys.Get(server)

	ready := make(chan struct{})
	lc.OnReady(func() { close(ready) })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := run(ctx, lc)
	recvOrFail(t, ready, "coordinator to become ready")

	if got := get(t, "http://"+srv.Addr()+web.ReadyPath); got != http.StatusOK {
		t.Errorf("GET %s once ready = %d, want %d", web.ReadyPath, got, http.StatusOK)
	}

	cancel()
	if err := recvOrFail(t, done, "Run to return"); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The Server's Err channel closes once it stops serving: it was shut
	// down, not left running.
	select {
	case err, open := <-srv.Err():
		if open {
			t.Errorf("the Server's Err channel yielded %v instead of closing after shutdown", err)
		}
	case <-time.After(failsafe):
		t.Error("the Server's Err channel is still open after Run returned")
	}
}
