package app

import (
	"context"
	"io"
	"log/slog"

	"github.com/standards-lab/go-core/event"
	"github.com/standards-lab/go-core/reactor"
	"github.com/standards-lab/go-database"
	"github.com/standards-lab/go-database/admin"
	"github.com/standards-lab/go-database/sweep"
	"github.com/standards-lab/go-messaging"
	"github.com/standards-lab/go-messaging/nats"
	"github.com/standards-lab/go-storage"
	"github.com/standards-lab/go-web-sdk"
	"go.opentelemetry.io/otel/metric"

	"github.com/JaimeStill/spike-graph/data"
	"github.com/JaimeStill/spike-graph/domain/document"
	"github.com/JaimeStill/spike-graph/domain/organization"
	"github.com/JaimeStill/spike-graph/internal/config"
	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/lifecycle"
	sdkweb "github.com/JaimeStill/spike-graph/sdk/web"
)

// App is the application layer: the dependency graph that describes the
// service and the [Nodes] that name its parts.
type App struct {
	graph *graph.Graph
	nodes Nodes
	ran   bool
}

// Nodes is the App's graph, one handle per node: the single description of
// what the service is composed of. Each layer file's define function fills
// its own part of one Nodes value, and its constructors read the lower
// layers' nodes from that same value. Each node's name is its field's
// name lowercased, which TestGraph_NodesNamedByTheirFields holds; it is
// the name the lifecycle labels its errors with and the readiness probe
// reports its check under. A value only one node uses is
// built inside that node's constructor rather than as a node of its own.
type Nodes struct {
	// The App's inputs (app.go): the configuration and the logger's
	// writer New was given, each the same value in every Build.
	Config *graph.Node[*config.Config]
	Writer *graph.Node[io.Writer]

	// Telemetry (telemetry.go).
	Telemetry     *graph.Node[*Telemetry]
	MeterProvider *graph.Node[metric.MeterProvider]
	SweepMetrics  *graph.Node[sweep.Hook]

	// Infrastructure (infrastructure.go).
	LifecycleConfig *graph.Node[lifecycle.Config]
	Logger          *graph.Node[*slog.Logger]
	Database        *graph.Node[*database.DB]
	Storage         *graph.Node[*storage.Store]
	Broker          *graph.Node[*nats.Broker]
	SQL             *graph.Node[*data.Database]
	Files           *graph.Node[*data.Storage]

	// Admin (admin.go).
	Gate   *graph.Node[*reactor.Gate]
	Schema *graph.Node[*admin.Service]

	// Domain (domain.go).
	Organization *graph.Node[*organization.Service]
	Document     *graph.Node[*document.Service]

	// Reactors (reactors.go). Reactors is the group every reactor is a
	// member of: no node Uses a reactor, so the group is a Build root,
	// and the server orders itself after it so it stays the top layer.
	Reactors *graph.Group
	Nudge    *graph.Node[*Nudge]
	Sweeper  *graph.Node[*sweep.Sweeper]
	Runtime  *graph.Node[*messaging.Runtime]
	Relay    *graph.Node[*reactor.Reactor[event.Event]]

	// The request edge (server.go).
	Readiness *graph.Node[*lifecycle.Readiness]
	Router    *graph.Node[*web.Router]
	Server    *graph.Node[*web.Server]
}

// New describes the graph over cfg and a writer for the service's logger,
// and returns the App. It is cold: it constructs nothing and cannot fail;
// [App.Run] builds. cfg and w are fixed-value nodes, the config node and
// the writer node, which the layers reach with Use like any other; then
// each layer file describes its nodes. The telemetry layer is described
// first, ahead of the infrastructure beneath it, because the connections
// order themselves after its node at their Define, and an After takes a
// handle already defined; every other layer follows lowest first.
func New(cfg *config.Config, w io.Writer) *App {
	a := &App{graph: graph.New()}
	a.nodes.Config = a.graph.Value("config", cfg)
	a.nodes.Writer = a.graph.Value("writer", w)
	defineTelemetry(a.graph, &a.nodes)
	defineInfrastructure(a.graph, &a.nodes)
	defineAdmin(a.graph, &a.nodes)
	defineDomain(a.graph, &a.nodes)
	defineReactors(a.graph, &a.nodes)
	defineServer(a.graph, &a.nodes)
	return a
}

// Graph returns the graph Run builds from, as [New] described it. The
// service itself never calls it; it is published so a caller can, before
// Run, observe what the Build constructs with graph.Graph.Observe, or
// Replace a node's constructor with a substitute.
func (a *App) Graph() *graph.Graph { return a.graph }

// Nodes returns a handle on each of a's graph nodes, for a caller's Replace
// or Observe.
func (a *App) Nodes() Nodes { return a.nodes }

// Roots returns the service's root set, the nodes no Use reaches and the
// set a Build of the whole service starts from: the server, the schema,
// telemetry, and the reactors group. The schema and telemetry are only
// ordered against, by After, so they are roots for the edge to hold; a
// Build of this set drops no After. [App.Run] builds it, plus the
// lifecycle config node the Coordinator is configured from.
func (a *App) Roots() []graph.Ref {
	n := a.nodes
	return []graph.Ref{n.Server, n.Schema, n.Telemetry, n.Reactors}
}

// Run runs the service with [web.Run], the candidate runner, and returns
// the process exit code. It builds [App.Roots] and the lifecycle config
// node, hands the System to a lifecycle Coordinator configured from that
// node, and runs it until ctx ends; the logger and the server it reads are
// reached from the server. Once the server has bound, the ready record
// names its address. Run logs "server stopped" and returns 0 after a clean
// drain, or logs "service failed" with the error and returns 1 on a
// startup, runtime, or shutdown failure. A Build failure is written to
// stderr instead, "service failed: " and the error, since the logger node
// may be what failed. An App runs once: a second Run panics before it
// builds anything.
func (a *App) Run(ctx context.Context, stderr io.Writer) int {
	if a.ran {
		panic("app: Run called twice; an App runs once")
	}
	a.ran = true

	n := a.nodes
	return sdkweb.Run(ctx, a.graph, n.LifecycleConfig, n.Logger, n.Server, a.Roots(), stderr)
}
