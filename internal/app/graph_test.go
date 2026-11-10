package app_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	libconfig "github.com/standards-lab/go-core/config"
	"github.com/standards-lab/go-core/process/processtest"
	"github.com/standards-lab/go-web-sdk"

	"github.com/JaimeStill/spike-graph/internal/app"
	"github.com/JaimeStill/spike-graph/internal/config/configtest"
	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/graph/graphtest"
	"github.com/JaimeStill/spike-graph/sdk/lifecycle"
)

// This file pins the graph's structural properties the composition root
// owns: the server alone in the top layer, the readiness checks the
// lifecycle infers from the node values and their order, telemetry beneath
// every other participant, the schema between the connections and the
// reactors, the relay held off by the gate, and each participant held by
// one node. It builds the graph without starting it, so it needs no live
// engine. Unlike app_test.go, these tests reach the graph through
// App.Graph and App.Nodes, and they read placement by handle: the System's
// layer of each node handle, and the handle on each built node. A failing
// layer assertion prints the System's layer table.
//
// Each placement guard is a function returning an error that carries the
// layer table, called both by the test of the App's own roots and by a
// test of a fixture that breaks it, so the fixture shows the guard can
// fail.

// buildService builds a's published root set and the lifecycle config node,
// the set web.Run builds from them, plus extra.
// TestGraph_RunBuildsTheRootSet holds that set to a real Run's.
func buildService(t *testing.T, a *app.App, extra ...graph.Ref) *graph.System {
	t.Helper()
	roots := append(a.Roots(), a.Nodes().LifecycleConfig)
	sys, err := a.Graph().Build(append(roots, extra...)...)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	return sys
}

// built returns the handle of each node in sys, layer 0 first.
func built(sys *graph.System) []graph.Ref {
	var refs []graph.Ref
	for _, d := range slices.Concat(sys.Layers()...) {
		refs = append(refs, d.Ref)
	}
	return refs
}

// layerError returns an error of the formatted message followed by sys's
// layer table.
func layerError(sys *graph.System, format string, args ...any) error {
	return fmt.Errorf(format+"\nlayers:\n%s", append(args, sys)...)
}

// wantGuardFails fails t unless err, a guard's verdict on sys, is an error
// that names mention and carries sys's layer table.
func wantGuardFails(t *testing.T, err error, sys *graph.System, mention string) {
	t.Helper()
	switch {
	case err == nil:
		t.Fatalf("the guard passed a fixture that breaks it\nlayers:\n%s", sys)
	case !strings.Contains(err.Error(), mention):
		t.Errorf("guard error = %q, want it to name %s", err, mention)
	case !strings.Contains(err.Error(), "\nlayers:\n"+sys.String()):
		t.Errorf("guard error = %q, want it to carry the layer table:\n%s", err, sys)
	}
}

// The System built from the App's published root set and the lifecycle
// config node holds exactly the nodes a real Run builds, compared by
// handle, so the tests below inspect what Run brings up rather than a set
// they chose. The context ends before startup, so the Run starts nothing.
func TestGraph_RunBuildsTheRootSet(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	var observed []graph.Ref
	a.Graph().Observe(func(r graph.Ref) { observed = append(observed, r) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code := a.Run(ctx, io.Discard); code != 0 {
		t.Fatalf("Run = %d, want 0", code)
	}
	ran := slices.Clone(observed)

	sys := buildService(t, a)
	got := built(sys)
	for _, r := range ran {
		if !slices.Contains(got, r) {
			t.Errorf("Run built %s, the root set did not", r.Name())
		}
	}
	for _, r := range got {
		if !slices.Contains(ran, r) {
			t.Errorf("the root set built %s, Run did not", r.Name())
		}
	}
	if len(got) != len(ran) {
		t.Errorf("the root set built %d nodes, Run built %d", len(got), len(ran))
	}
	if t.Failed() {
		t.Logf("layers:\n%s", sys)
	}
}

// serverAlone reports, as an error carrying the layer table, any node of
// sys at or above the layer of the server's node: the server must be alone
// in the top layer, so it starts after every other node and drains first.
func serverAlone(sys *graph.System, server graph.Ref) error {
	if !slices.Contains(built(sys), server) {
		return layerError(sys, "%s is not in the System", server.Name())
	}
	top := sys.LayerOf(server)
	var beside []string
	for _, r := range built(sys) {
		if r != server && sys.LayerOf(r) >= top {
			beside = append(beside, r.Name())
		}
	}
	if len(beside) > 0 {
		return layerError(sys, "%v at or above %s's layer %d; want %s alone in the top layer",
			beside, server.Name(), top, server.Name())
	}
	return nil
}

// The server is the only node in the top layer of the graph Run builds. A
// node defined above it, or beside it at the top, fails the guard; the
// second case shows the guard catches one.
func TestGraph_ServerAloneInTopLayer(t *testing.T) {
	t.Run("as defined", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		if err := serverAlone(buildService(t, a), a.Nodes().Server); err != nil {
			t.Error(err)
		}
	})

	t.Run("a node above the server", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		n := a.Nodes()
		above := a.Graph().Define("above", func(s *graph.Scope) (*web.Server, error) {
			return s.Use(n.Server), nil
		})
		sys := buildService(t, a, above)
		wantGuardFails(t, serverAlone(sys, n.Server), sys, "above")
	})
}

// The readiness checks the lifecycle infers from the node values, each one
// that both starts and reports Ready, are the database, the object store,
// the broker, the schema, the sweeper, and the relay, in that order;
// /readyz reports them after "lifecycle". No other value is both: not the
// nudge the document layer reaches the sweeper through, not the
// messaging runtime, which starts nothing, and not the readiness node
// itself. The readiness node the probes read is bound to the Coordinator
// the System is handed to, so it reports that Coordinator's checks.
func TestGraph_ReadinessChecks(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	n := a.Nodes()
	sys := buildService(t, a)

	checkNames := func(checks []lifecycle.Check) []string {
		var out []string
		for _, c := range checks {
			out = append(out, c.Name)
		}
		return out
	}
	got := checkNames(lifecycle.New(sys, sys.Get(n.LifecycleConfig)).Checks())
	want := []string{"database", "storage", "broker", "schema", "sweeper", "relay"}
	if !slices.Equal(got, want) {
		t.Errorf("checks = %v, want %v\nlayers:\n%s", got, want, sys)
	}
	if probed := checkNames(sys.Get(n.Readiness).Checks()); !slices.Equal(probed, want) {
		t.Errorf("readiness node checks = %v, want the Coordinator's %v", probed, want)
	}
}

// dropsNoOrder reports, as an error carrying the layer table, each After
// of sys that orders nothing, its target not in the System.
func dropsNoOrder(sys *graph.System) error {
	var drops []string
	for _, d := range sys.Dropped() {
		drops = append(drops, fmt.Sprintf("%s's After of %s", d.Node.Name(), d.After.Name()))
	}
	if len(drops) > 0 {
		return layerError(sys, "dropped %s; want every After to order", strings.Join(drops, ", "))
	}
	return nil
}

// A Build of the App's published root set drops no After: every node an
// edge orders against, the schema and telemetry above all, is in the
// System, and every reactor is in the group the server orders after. A
// root missing from the set, or a reactor missing its In, fails the guard
// instead of silently losing an order; the second case shows the guard
// catches an After whose target the roots do not build.
func TestGraph_RootsDropNoOrder(t *testing.T) {
	t.Run("as defined", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		sys, err := a.Graph().Build(a.Roots()...)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		if err := dropsNoOrder(sys); err != nil {
			t.Error(err)
		}
	})

	t.Run("an After of a node not built", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		unbuilt := a.Graph().Value("unbuilt", 0)
		orderer := a.Graph().Value("orderer", 0, graph.After(unbuilt))
		sys, err := a.Graph().Build(append(a.Roots(), orderer)...)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		wantGuardFails(t, dropsNoOrder(sys), sys, "orderer's After of unbuilt")
	})
}

// participant reports whether v takes part in the lifecycle's startup or
// shutdown.
func participant(v any) bool {
	switch v.(type) {
	case lifecycle.Starter, lifecycle.Stopper:
		return true
	}
	return false
}

// telemetryBeneath reports, as an error carrying the layer table, each
// lifecycle participant of sys other than telemetry whose layer is not
// above telemetry's: telemetry must sit beneath every other participant,
// so its providers are installed before any starts and flushed after all
// have stopped.
func telemetryBeneath(sys *graph.System, telemetry graph.Ref) error {
	if !slices.Contains(built(sys), telemetry) {
		return layerError(sys, "%s is not in the System", telemetry.Name())
	}
	tel := sys.LayerOf(telemetry)
	var below []string
	for _, d := range slices.Concat(sys.Layers()...) {
		if d.Ref != telemetry && participant(d.Value) && sys.LayerOf(d.Ref) <= tel {
			below = append(below, d.Ref.Name())
		}
	}
	if len(below) > 0 {
		return layerError(sys, "participants %v not above %s's layer %d", below, telemetry.Name(), tel)
	}
	return nil
}

// lateStarter is a participant that orders itself after nothing.
type lateStarter struct{}

func (lateStarter) Start(context.Context) error { return nil }

// Telemetry sits beneath every other lifecycle participant. A participant
// that misses the edge to telemetry fails the guard; the second case shows
// the guard catches one.
func TestGraph_TelemetryBeneathParticipants(t *testing.T) {
	t.Run("as defined", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		if err := telemetryBeneath(buildService(t, a), a.Nodes().Telemetry); err != nil {
			t.Error(err)
		}
	})

	t.Run("a participant missing the edge", func(t *testing.T) {
		a := app.New(configtest.Config(t), io.Discard)
		stray := a.Graph().Define("stray", func(*graph.Scope) (lateStarter, error) {
			return lateStarter{}, nil
		})
		sys := buildService(t, a, stray)
		wantGuardFails(t, telemetryBeneath(sys, a.Nodes().Telemetry), sys, "stray")
	})
}

// The connections, the database, the object store, and the broker, share a
// layer beneath the schema, so they start together and close together
// after it, and the sweeper sits above the schema, so it starts once every
// table it touches is verified and stops before the schema does.
func TestGraph_SchemaBetweenConnectionsAndSweeper(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	sys := buildService(t, a)
	n := a.Nodes()

	db, store, broker := sys.LayerOf(n.Database), sys.LayerOf(n.Storage), sys.LayerOf(n.Broker)
	schema, sweeper := sys.LayerOf(n.Schema), sys.LayerOf(n.Sweeper)
	if db != store || db != broker {
		t.Errorf("database in layer %d, storage in %d, broker in %d, want one layer\nlayers:\n%s", db, store, broker, sys)
	}
	if db >= schema {
		t.Errorf("connections in layer %d, schema in layer %d, want the connections below\nlayers:\n%s", db, schema, sys)
	}
	if sweeper <= schema {
		t.Errorf("sweeper in layer %d, schema in layer %d, want the sweeper above\nlayers:\n%s", sweeper, schema, sys)
	}
}

// The relay sits above both the schema and the broker: the relay starts
// once the messaging set is migrated and verified and the broker is
// connected, and it drains, its last pass included, before the schema, the
// broker, and the database stop. The document node does not Use the relay,
// so the schema, which Uses the document node for its seed, can sit beneath
// it.
func TestGraph_RelayAboveSchemaAndBroker(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	sys := buildService(t, a)
	n := a.Nodes()

	broker, schema, relay := sys.LayerOf(n.Broker), sys.LayerOf(n.Schema), sys.LayerOf(n.Relay)
	if relay <= schema {
		t.Errorf("relay in layer %d, schema in layer %d, want the relay above\nlayers:\n%s", relay, schema, sys)
	}
	if relay <= broker {
		t.Errorf("relay in layer %d, broker in layer %d, want the relay above\nlayers:\n%s", relay, broker, sys)
	}
}

// The relay holds the gate shared for each row, so while a schema verb
// holds it exclusively the relay opens no transaction: against a database
// that refuses every connection, it reports no failure until the gate is
// released, and then reports the refused connection. No live engine is
// needed: the failure the release lets through is the refusal itself.
func TestGraph_RelayHeldOffByTheGate(t *testing.T) {
	cfg := configtest.Config(t)
	cfg.Messaging.RelayPoll = libconfig.Duration(5 * time.Millisecond)
	var log syncBuffer
	a := app.New(cfg, &log)
	n := a.Nodes()
	sys, err := a.Graph().Build(n.Relay)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	failed := func() bool { return strings.Contains(log.String(), `msg="messaging failure"`) }

	release, err := sys.Get(n.Gate).Exclusive(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	relay := sys.Get(n.Relay)
	if err := relay.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Many polls' worth: a relay that ignored the gate would have dialled
	// the closed port and reported the refusal by now.
	time.Sleep(100 * time.Millisecond)
	if failed() {
		t.Fatalf("the relay reported a failure while the gate was held exclusively:\n%s", log.String())
	}

	release()
	deadline := time.Now().Add(5 * time.Second)
	for !failed() {
		if time.Now().After(deadline) {
			t.Fatalf("no relay failure after the gate's release; want the refused connection reported:\n%s", log.String())
		}
		time.Sleep(time.Millisecond)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := relay.Shutdown(ctx); err != nil {
		t.Errorf("Shutdown = %v, want nil", err)
	}
}

// The sweeper cancels a cycle still running half the shutdown timeout into
// its drain, go-core's CancelWithin rule, and its drain reports the period:
// under the 10s default, "handlers cancelled after 5s". A shorter timeout
// keeps the test fast. The cycle in flight is the files reclaimer's, its
// claim waiting on a connection to a database that accepts and never
// answers, which only the cancellation ends, so no live engine is needed.
// The drain alone does not end it: a claim, unlike a wait on the gate,
// runs until its context does.
func TestGraph_SweeperCancelsWithinTheShutdownTimeout(t *testing.T) {
	cfg := configtest.Config(t)
	cfg.ShutdownTimeout = libconfig.Duration(200 * time.Millisecond)
	port, dialled := silentPort(t)
	cfg.Database.Port = &port
	a := app.New(cfg, io.Discard)
	n := a.Nodes()
	sys, err := a.Graph().Build(n.Sweeper)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sweeper := sys.Get(n.Sweeper)
	if err := sweeper.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Every reclaimer is due at start, so the files cycle begins at once
	// and its claim dials the silent database: once the dial is accepted,
	// the cycle is in flight, waiting on an answer that never comes.
	select {
	case <-dialled:
	case <-time.After(failsafe):
		t.Fatal("the sweeper's claim never dialled the database")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = sweeper.Shutdown(ctx)
	if err == nil || !strings.Contains(err.Error(), "handlers cancelled after 100ms") {
		t.Errorf("Shutdown = %v, want the cycle cancelled after 100ms, half the shutdown timeout", err)
	}
}

// silentPort listens on a loopback port that accepts each connection and
// never answers it, so a client's handshake waits until its context ends.
// The channel is closed once the first connection is accepted. The
// listener and its connections close at cleanup.
func silentPort(t *testing.T) (int, <-chan struct{}) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	accepted := make(chan struct{})
	first := sync.OnceFunc(func() { close(accepted) })
	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			c, err := lis.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
			first()
		}
	}()
	t.Cleanup(func() {
		_ = lis.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})
	return lis.Addr().(*net.TCPAddr).Port, accepted
}

// closingPort listens on a loopback port that closes each connection as it
// accepts it, so a client's handshake fails at once. The listener closes at
// cleanup.
func closingPort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			c, err := lis.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	t.Cleanup(func() { _ = lis.Close() })
	return lis.Addr().(*net.TCPAddr).Port
}

// claimFailures counts the sweeper's records, in log, of a failed claim of
// the lock of the reclaimer named.
func claimFailures(log, reclaimer string) int {
	re := regexp.MustCompile(`sweep claim failed.*reclaimer\W+` + regexp.QuoteMeta(reclaimer) + `\b`)
	return len(re.FindAllStringIndex(log, -1))
}

// The nudge the document layer gives after each branch it marks reaches
// the sweeper's files reclaimer and no other. The database closes every
// connection, so each cycle's claim fails and is logged under its
// reclaimer's name; with an interval of an hour, a cycle after the start's
// runs only on a nudge. The nudge node the document layer is built with
// then earns the files reclaimer, alone, a second cycle: a nudge left
// unbound, or bound to another reclaimer, never logs one. The records also
// show the service's sweeper claims each reclaimer's lock: one built
// sweep.Unlocked would log no claim. Unbound, as in a graph built without
// the sweeper, a nudge does nothing and never panics.
func TestGraph_TheDocumentNudgeReachesTheFilesReclaimer(t *testing.T) {
	cfg := configtest.Config(t)
	cfg.Sweep.Interval = libconfig.Duration(time.Hour)
	port := closingPort(t)
	cfg.Database.Port = &port
	log := &syncBuffer{}
	a := app.New(cfg, log)
	n := a.Nodes()

	alone, err := a.Graph().Build(n.Document)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	alone.Get(n.Nudge).Nudge()

	sys, err := a.Graph().Build(n.Document, n.Sweeper)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sweeper := sys.Get(n.Sweeper)
	if err := sweeper.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := sweeper.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown = %v", err)
		}
	})
	// The start's cycles run in the list's order, inbox last.
	processtest.WaitFor(t, "the start's cycles", func() bool {
		return claimFailures(log.String(), "inbox") == 1
	})

	sys.Get(n.Nudge).Nudge()
	processtest.WaitFor(t, "the nudged files cycle", func() bool {
		return claimFailures(log.String(), "files") == 2
	})
	if o, i := claimFailures(log.String(), "outbox"), claimFailures(log.String(), "inbox"); o != 1 || i != 1 {
		t.Errorf("outbox and inbox cycles = %d and %d, want the start's alone:\n%s", o, i, log)
	}
}

// The sweeper registers the three reclaimers the service runs, files,
// outbox, and inbox: go-database's Sweeper panics on a nudge of a name it
// was not built with, so a reclaimer missing from the registration, or
// registered under another name, fails here. A name it was not built with
// is refused, which shows a nudge of a missing name does fail.
func TestGraph_SweeperRegistersTheReclaimers(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	sweeper := buildService(t, a).Get(a.Nodes().Sweeper)

	nudge := func(name string) (panicked bool) {
		defer func() { panicked = recover() != nil }()
		sweeper.Nudge(name)
		return false
	}
	for _, name := range []string{"files", "outbox", "inbox"} {
		if nudge(name) {
			t.Errorf("Nudge(%q) panicked; want %s registered", name, name)
		}
	}
	if !nudge("sessions") {
		t.Error(`Nudge("sessions") did not panic; want a name the sweeper was not built with refused`)
	}
}

// Each lifecycle participant is the value of one node only: the same value
// in two nodes would start twice, stop twice, and, as a ReadyStarter, be
// reported twice. The admin storage routes read the storage node's
// store rather than holding it as a node of their own.
func TestGraph_EachParticipantInOneNode(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	sys := buildService(t, a)

	held := map[any]graph.Ref{}
	for _, d := range slices.Concat(sys.Layers()...) {
		switch d.Value.(type) {
		case lifecycle.Starter, lifecycle.Stopper, lifecycle.ReadinessChecker, lifecycle.Monitored:
		default:
			continue
		}
		if other, dup := held[d.Value]; dup {
			t.Errorf("nodes %s and %s hold the same participant\nlayers:\n%s", other.Name(), d.Ref.Name(), sys)
		}
		held[d.Value] = d.Ref
	}
}

// Each node's name is its Nodes field's name lowercased, so a field names
// the node it holds without a comment repeating the name, and every field
// holds a handle once New returns. "lifecycle config" passes as
// LifecycleConfig: the check ignores the name's spaces.
func TestGraph_NodesNamedByTheirFields(t *testing.T) {
	a := app.New(configtest.Config(t), io.Discard)
	graphtest.CheckNames(t, a.Nodes())
}

// Telemetry's Shutdown is safe when its Start never ran, as the
// lifecycle's shutdown of a participant whose Start failed calls it: it
// returns nil and logs nothing, go-observability releasing the exporters
// Start never handed to a provider.
func TestTelemetry_ShutdownWithoutStart(t *testing.T) {
	var log syncBuffer
	a := app.New(configtest.Config(t), &log)
	sys, err := a.Graph().Build(a.Nodes().Telemetry)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if err := sys.Get(a.Nodes().Telemetry).Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown = %v, want nil", err)
	}
	if out := log.String(); out != "" {
		t.Errorf("log = %q, want nothing", out)
	}
}
