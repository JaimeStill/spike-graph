package web

import (
	"context"
	"io"
	"log/slog"
	"slices"

	"github.com/standards-lab/go-core/process"

	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/lifecycle"
)

// Server is the value a server node must have for [Run]: one that reports
// the address it serves on. go-web-sdk's *web.Server satisfies it, reading
// back the assigned port of a configured port 0 once it has bound.
type Server interface {
	Addr() string
}

// Run builds roots and lifecycleConfig from g, hands the System to a
// lifecycle Coordinator configured with lifecycleConfig's value, and runs it
// until ctx ends, returning the process exit code. The logger and the
// server are read from the System, so roots must reach both; a root set
// that does not is a wiring mistake, and Run panics on it as the graph's
// Get does.
//
// Run reports its outcome once:
//
//   - once startup completes, it logs "server ready" on the logger node's
//     value, with the server's bound address as addr
//   - when ctx ends and the System drains cleanly, it logs "server
//     stopped" and returns [process.ExitOK]
//   - when startup, a monitored participant at runtime, or the shutdown
//     fails, it logs "service failed" with the Coordinator's error, which
//     names the failing node, and returns [process.ExitFailure]; a failed
//     startup reports nothing ready
//   - when the Build fails, it writes "service failed: <error>" to stderr,
//     the error naming the failing node, and returns
//     [process.ExitFailure]. It logs nothing, since the logger node may be
//     what failed, and starts nothing.
//
// stderr is the process's standard error, or a test's buffer, as cli.Run's
// Streams are.
func Run[S Server](
	ctx context.Context,
	g *graph.Graph,
	lifecycleConfig *graph.Node[lifecycle.Config],
	logger *graph.Node[*slog.Logger],
	server *graph.Node[S],
	roots []graph.Ref,
	stderr io.Writer,
) int {
	sys, err := g.Build(append(slices.Clip(roots), lifecycleConfig)...)
	if err != nil {
		return process.Fail(stderr, "service failed", err)
	}

	log := sys.Get(logger)
	srv := sys.Get(server)
	lc := lifecycle.New(sys, sys.Get(lifecycleConfig))
	lc.OnReady(func() {
		log.Info("server ready", "addr", srv.Addr())
	})

	if err := lc.Run(ctx); err != nil {
		log.Error("service failed", "error", err)
		return process.ExitFailure
	}
	log.Info("server stopped")
	return process.ExitOK
}
