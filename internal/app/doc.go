// Package app is the composition root. It describes the service as one
// dependency graph, on the candidate graph package (sdk/graph), run by the
// candidate lifecycle (sdk/lifecycle). Each architecture layer has one
// file, which defines the layer's nodes in its define function, so the
// layer files are the architecture's layer list, in the order [New]
// defines them:
//
//   - app.go: the App's inputs, the configuration and the logger's
//     writer, each a fixed-value node;
//   - telemetry.go: defineTelemetry, the trace and meter providers, which
//     start before the connections and flush after them, the meter
//     provider the service's own instruments are created on, and the
//     sweep metrics, the sweeper's per-pass hook over its instruments,
//     sweep.passes and sweep.backlogged. It is defined ahead of the
//     infrastructure beneath it, because the connections order after its
//     node at their Define, and an After or In panics on a handle defined
//     later;
//   - infrastructure.go: defineInfrastructure, the lifecycle config, the
//     logger, the database pool, the object store, the NATS broker, and
//     the two the domains see the first two through, the sql session and
//     the files store;
//   - admin.go: defineAdmin, the quiesce gate and the schema service, and
//     the /admin mount;
//   - domain.go: defineDomain, the domain services, and the /api mount;
//   - reactors.go: defineReactors, the reactors group, the general
//     sweeper and the nudge the document layer reaches it through, and the
//     messaging runtime and its relay;
//   - server.go: defineServer, the request edge: the readiness the probes
//     report, the router, and the server;
//   - routes.go: the list of mounts;
//   - middleware.go: the router-level middleware stack, outermost first.
//
// [Nodes] holds one handle per node, the single description of what the
// service is composed of: each define function fills its own part of one
// Nodes value, and its constructors, methods on Nodes, read the other
// layers' nodes from it with Use. [App] is the described process. [New] is
// the cold start: it describes the graph, layer by layer, constructs
// nothing, and cannot fail. [App.Graph] and [App.Nodes] publish the graph
// and its handles, so a caller can Observe or Replace a node before Run,
// as internal/apptest's fixtures do, and [App.Roots] publishes the root
// set: the server, the schema, telemetry, and the reactors group. [App.Run]
// is the hot start plus shutdown, a single call to the candidate runner
// (sdk/web): it builds the root set and the lifecycle config node, hands
// the System to the lifecycle Coordinator configured from that node, and
// returns the process exit code. A constructor's error is reported there,
// on the stderr Run is given, as the service failing; a wiring mistake
// panics during Build or, for an option, at Define.
//
// What a node takes part in is inferred from its value's methods: a
// lifecycle.Starter or Stopper is started or stopped, a Subsystem is both,
// a Starter that is also a ReadinessChecker (a lifecycle.ReadyStarter)
// joins the readiness probe, though a Ready alone does not, and a Monitored
// has its runtime error watched. Nodes start in layer order and drain in
// reverse.
// No layer file registers anything with the Coordinator. The order is
// computed from what each node uses, plus the After edges each node
// declares at its Define, which order it without passing a value: the
// database, the object store, and the broker after telemetry, the sweeper
// and the relay after the schema, and the server after the reactors group
// and the schema, so it is alone in the top layer, starts last, and drains
// first. No constructor orders anything. A node only ordered against is in
// the root set, so a Build of it drops no edge. The drain therefore runs the
// server, then the reactors (the relay with its last pass, which publishes
// what the server's requests committed while they drained), then the
// schema, then the broker and the connections. The probes report the
// Coordinator's status under the "lifecycle" name, then the database, the
// object store, the broker, the schema, the sweeper, and the relay, in
// layer, then definition, order. The readiness, the quiesce gate, and the
// nudge are fresh nodes, so each Build's value starts zero; the readiness
// is bound to the Coordinator its System is handed to. [Telemetry] and
// [Nudge] are the two node values the package owns: telemetry as a
// participant whose shutdown never fails the run, and the sweeper's nudge,
// which has no Ready, so the probe gains no check for it.
//
// Routes and reactors are the two ways work enters the running process: a
// caller drives a route, and an occurrence drives a reactor. A reactor
// commonly dispatches to a domain service; the sweeper's dispatches to its
// reclaimers, the libraries' own (the README's Sweep section), and the
// relay's publishes to the broker.
// What a producer of events must order itself after is defineReactors'
// drain rule. Extending the service means defining a node in its layer's
// define function, a reactor In the reactors group; the signatures,
// cmd/server, and [App.Run] stay untouched.
package app
