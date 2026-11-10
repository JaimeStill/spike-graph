# goal · experiment.graph.spike-graph

- **State:** brief ready
- **Task:** web
- **Branch:** web

## Tasks

1. [x] baseline
2. [x] candidate
3. [ ] web
4. [ ] cli
5. [ ] adapters

## Task brief · web

**Problem.** The candidate graph and lifecycle API is so far proven only by the candidate packages' own tests. go-web-service's real composition root still runs on go-core's graph:
- order is set by `s.After` calls inside constructor bodies;
- telemetry is pulled into the Build by a Use whose value is thrown away;
- the reactors are a hand-kept list of roots;
- 14 wrapper constructors and 6 closures exist only to reach handles or return fixed values;
- 21 `// "name"` comments repeat the Define strings;
- the layer tests copy Run's root list and match nodes by name;
- a 30-line Run builds, runs, and reports by hand.

Until this root runs on the candidate against the real stack, the intake can't say which of decisions 9–14 survive. It also can't settle three open details:
- what `web.Run` returns, and who logs;
- the rule the name check applies;
- how a `*Nodes` method reaches a plain value.

**Behaviors**

1. The whole service builds and runs on the candidate graph and lifecycle packages, and no part of the service reaches go-core's graph. The unit tier and the integration tier pass against the spike's stack.
2. Every edge that orders without passing a value is declared at its node's Define, and no constructor body orders anything:
   - the database, the object store, and the broker order after telemetry;
   - the sweeper and the relay order after the schema;
   - the server orders after the reactors group and the schema.
3. The sweeper and the relay are members of a reactors group. Telemetry is a root, so no constructor Uses it only to bring it into the Build.
4. Node placement and readiness are unchanged:
   - the server is alone in the top layer;
   - telemetry sits beneath every other lifecycle participant;
   - the database, the object store, and the broker share one layer beneath the schema;
   - the sweeper and the relay sit above the schema, and the relay also sits above the broker;
   - the readiness checks run in this order: database, storage, broker, schema, sweeper, relay;
   - each lifecycle participant is the value of exactly one node.
5. The telemetry layer is defined first, ahead of infrastructure, so the connections can order after it.
6. Every constructor in the composition root is a method on the Nodes value or a plain function. No function returns a constructor.
7. The configuration and the logger's writer are fixed-value nodes named "config" and "writer". The logger Uses the writer node, and its records reach the writer the App was created with.
8. The readiness, the quiesce gate, and the nudge are fresh nodes, so each Build gets its own zero value. The readiness is the candidate lifecycle's Readiness, and the probes report the Coordinator it is bound to.
9. A lifecycle-config node supplies the Coordinator's configuration from the config node's lifecycle block.
10. The App publishes its root set: the server, the schema, telemetry, and the reactors group. Run builds exactly that set plus the lifecycle-config node. Building the published root set drops no ordering edge, so `Dropped()` is empty.
11. The candidate `web.Run` takes a context, the graph, the lifecycle-config node, the logger node, the server node, and the roots. It returns the process exit code:
    - once startup completes, it logs "server ready" once, on the logger node, with the address the server bound; on port 0 that is the assigned port;
    - when the context ends after a clean drain, it logs "server stopped" and returns 0;
    - when startup fails, or a monitored participant fails at runtime, it logs "service failed" on the logger node with an error that names the failing node, and returns 1; nothing reports ready;
    - when the Build fails, it writes "service failed: <error>" to stderr, naming the failing node, and returns 1; it logs nothing and reports nothing ready, since the logger node may be what failed.
12. App.Run is a single call to `web.Run`, and the App keeps no fallback logger. The server entrypoint passes its stderr through and returns Run's exit code. A second Run on one App panics before it builds anything.
13. The layer tests read placement through the published root set, the System's layer of each handle, and the handles on each built node. A failing layer assertion prints the System's layer table. Each guard is shown to fail on a fixture that breaks it:
    - a node defined above the server;
    - a participant with no edge to telemetry;
    - an ordering edge that orders nothing.
14. A shared name-check test helper in a candidate graph test package takes a struct of node and group handles. It reports each exported handle field that holds no handle, and each one whose name, lowercased with spaces removed, differs from the field name lowercased. The helper's own tests show:
    - it accepts web-style names ("meterprovider" for MeterProvider) and slab-style names ("nats stream" for NATSStream);
    - it rejects a mismatched name and an undefined handle.
15. The service's Nodes pass the name check, and the 21 `// "name"` comments are gone. No node is renamed, so the error labels and the readiness check names stay as they are.

**Test seams.** The main seam is the App: New, its graph, its Nodes, its root set, and Run. The unit tier exercises it, and the integration tier exercises it through the built server binary. Two smaller seams have their own tests: `web.Run`, called directly on a small graph, and the name-check helper, called on fixture structs.

**Slices** (no upgrade slice; the currency command reports nothing)

1. **The root on the candidate.** The composition root, its fixtures, and its tests move onto the candidate packages, with Run still written by hand. Behaviors 1–10 hold, apart from Run's shape. Done when the check and the integration tier pass on the stack, and the service no longer reaches go-core's graph.
2. **`web.Run`.** The candidate runner reports ready, failed, and stopped. App.Run becomes one call, and the entrypoint passes stderr through. Behaviors 11–12. Done when:
   - `web.Run`'s own tests cover ready, clean stop, startup failure, runtime failure, and Build failure;
   - the App's tests show the Build failure on stderr;
   - the integration tier's ready-address, port-in-use, and drain cases pass.
3. **Layer tests on handles.** The layer tests are rewritten on the root set, the layer of each handle, and the handles on built nodes. The old helpers that copied the root list and matched names are gone. Behavior 13, with behavior 4 pinned. Done when each guard fails on its breaking fixture and prints the layer table, `Dropped()` is empty for the full root set, and the check passes.
4. **The name check.** The shared helper and its own tests, with the service's Nodes checked by it and the comments removed. Behaviors 14–15. Done when:
   - the helper's tests pass for both naming styles and both failure cases;
   - the service's name check passes, no `// "name"` comment remains, and the check passes.

**Out of scope**
- slab and the CLI fork, which belong to the `cli` task.
- Arity adapters and the before/after measurement table, which belong to the `adapters` task.
- Any change to node names, error labels, readiness names, or log record names, apart from the Build failure's move to stderr.
- Changes outside the composition root, its fixtures, the entrypoint, and the code that runs a graph.
- Changes to published go-web-sdk or go-core. go-core stays held at v0.10.0.
- The intake findings already noted: concurrent Builds, Replace removing edges, Group.Members, String's ambiguity, and observer panics.

**Door.** Two-way. Every change is code on a branch of the spike, measured against the baseline commit, and nothing is released.

## Progress

slices 4/4 committed · standards done · spec done · editor done

## Decisions

- setup: full copies of go-web-service and slab at 5bb590e, proven against the spike's own copy of go-web-service's compose stack, not cut-down miniatures with fakes.
- setup: the copies are committed unchanged as a baseline on go-core v0.10.0, then migrated in place; the before/after table comes from that diff. Superseded by baseline: the copy's dependencies are upgraded, and the measurement baseline is the baseline task's merge commit.
- setup: fork go-core's `graph` whole; fork only `lifecycle`'s Coordinator and New and alias the rest, so published go-web-sdk works unchanged; fork go-cli-sdk v0.2.0's `cli`; `web.Run` in the spike's own `web` package.
- setup: the stack is the spike's own copy (own project name and ports, observability profile included), never go-web-service's dev stack.
- baseline: the copied service is the root module github.com/JaimeStill/spike-graph; slab stays its own module .../tools/slab.
- baseline: stack ports come from the mise environment; the copied files change only by the path rewrite, gofmt import order, and the compose project name.
- baseline: the copy's dependencies and image pins are upgraded to latest; go-web-service's own upgrades are planned at intake.
- baseline: held go-core at v0.10.0: the candidate's fork point; latest v0.10.0.
- baseline: go-web-service's currency script replaces the spike's.
- baseline: go-web-service's repository-level files and CI are left out; package-level docs are copied as-is.
- baseline: slab's scenarios run once on the baseline as demo evidence.
- baseline: the measurement baseline is this task's merge commit; the copy-as-is commit stays in history.
- baseline: noted for cli: slab must reach the candidate forks in the root module without a replace directive, by folding slab into the root module or a go.work; folding puts slab on the root module's sqlate version.
- baseline: [env] sets APP_ENV=local, so serve reads config.local.json.
- baseline: [env] derives every APP_* and SLAB_* setting from the port variables, so each port is set in one place.
- baseline: the stack tasks read only the spike's [env]; no task falls back to go-web-service's ports.
- baseline: integration keeps [env]'s APP_OBSERVABILITY_ENDPOINT; its telemetry reaches the spike's dev collector when that is up, as go-web-service's does.
- baseline: go-web-service's vet, test, fmt, tidy, and lint tasks are not copied; the spike's one check covers them.
- baseline: the root module's indirect golang.org/x/sys stays at v0.48.0; currency doesn't scan indirect requirements.
- baseline: slab runs through `mise run slab`; a bare slab binary defaults to go-web-service's ports.
- candidate: fork lifecycle's Readiness with Coordinator and New; go-core's Readiness binds only through go-core's unexported bind, and go-web-sdk takes it only through Doctor. Amends setup's "fork only Coordinator and New".
- candidate: LayerOf panics with a "graph: " message for a node not in the System, as Get does.
- candidate: LayerOf panics for a group handle; a group has no layer.
- candidate: library candidate packages live under sdk/<package> (sdk/graph, sdk/lifecycle); later forks follow the same rule.
- candidate: Replace keeps a node's After and In; in v0.10.0 a substitute constructor dropped its Scope.After edges, so a test fixture's substitute now keeps its order.
- candidate: groups and nodes share one namespace.
- candidate: a cycle's panic path names the groups it passes through.
- candidate: Define validates every option before it changes the Graph, and panics on a nil option.
- candidate: Value's and Fresh's panic messages name the call that was made.
- candidate: After doesn't deduplicate its targets as Scope.After did; layering is unaffected.
- candidate: a group's members build in definition order.
- candidate: String prints `N: a, b` per layer with no trailing newline; the empty System prints "".
- candidate: LayerOf also panics for a nil ref.
- candidate: sdk/lifecycle's aliases live in one alias.go; the forked files keep go-core's names.
- candidate: Fable review: Value and Fresh take Define's options and Fresh is a method, g.Fresh[T](name); a participant must be able to join a group, and changing it after release is a breaking change.
- candidate: Fable review: After stays soft for subset builds; System.Dropped() exposes the Afters that order nothing.
- candidate: Fable review: zero-literal handles panic with "graph: ", the nil-After message hints a later Define, and lifecycle.New rejects go-core's *Readiness.
- candidate: System.Get of a nil *Node panics "graph: Get of a nil node"; v0.10.0 said `Get of "<nil>", a node not in the System`.
- candidate: Replace of a nil or zero-value handle panics with a "graph: " message, checked before the after-Build check; v0.10.0 crashed on a nil dereference.
- candidate: Name() on a nil or zero-value handle returns ""; v0.10.0 crashed.
- candidate: the later-Define hint appears only in the panics for Define's, Value's, and Fresh's options, and asks "is its Group later?" for a group.
- candidate: Dropped checks only the After refs of built nodes, in definition order and then declaration order; String doesn't show drops.
- candidate: noted for web: After needs a handle defined earlier; telemetry is defined after database, so the root defines telemetry first or forward-declares a group; the root's layer test asserts Dropped() is empty for the full root set.
- candidate: noted for intake: concurrent Builds race on Graph.built (documented, as in go-core); Replace can't remove an edge or membership; no Group.Members(); String is ambiguous for names with ", "; an observer's panic escapes Build.
- web: `web.Run` logs "server ready", "service failed", and "server stopped" on the logger node and returns the exit code, as cli.Run does; a Build failure goes to stderr; it takes the lifecycle-config, logger, and server nodes and the roots; App.Run is one call, and App drops its fallback logger.
- web: a `*Nodes` method reaches a plain value through a graph.Value node; the logger's writer is the "writer" node, which the logger Uses.
- web: the name check's rule: a node's or group's name, lowercased with spaces removed, equals its Nodes field's name lowercased; no node is renamed.
- web: the name check is a shared test helper in a candidate graph test package, with its own tests, so slab reuses it in the cli task.
- web: the telemetry layer is defined first, ahead of infrastructure, so the connections can order after it; no group is forward-declared.
- web: noted for adapters: the spike's copy has 14 wrapper constructors plus the plain newMeterProvider, not the design note's 17; the table counts from the spike's own diff.
- web: escalation: a participant that fails at runtime is named in "service failed"; sdk/lifecycle labels each Monitored channel with its node's name. The fork diverges from v0.10.0 here, on purpose: sdk/lifecycle works out how lifecycle should evolve for go-core's release.
- web: New defines the config and writer nodes in app.go, ahead of defineTelemetry; the lifecycle config node sits in infrastructure.go beside the logger.
- web: `web.Run[S Server](ctx, g, lifecycleConfig, logger, server, roots, stderr) int`, with `Server` an `Addr() string` interface; App.Run(ctx, stderr) returns it, and the App keeps neither cfg nor w.
- web: internal/app imports sdk/web as `sdkweb`, since go-web-sdk's package is also `web`.
- web: web.Run panics through Get when roots don't reach the logger and the server; a Build's wiring panic doesn't reach stderr; a failed shutdown logs "service failed" and returns 1.
- web: a channel registered with Monitor belongs to no node, so its failure stays "run: <err>".
- web: graphtest exports NameErrors(nodes any) []error and CheckNames(t testing.TB, nodes any); a handle field is one whose type implements graph.Ref; a nil or zero-value handle reports "no handle"; a non-struct is one error.
- web: apptest.ManualMeter keeps its Replace closure; it is a test substitute, not a wrapper constructor.
- web: noted for intake: g.Value infers T from the argument's static type; Writer is a Node[io.Writer] only because New's w is an io.Writer.
- web: noted for intake: telemetry is defined ahead of the logger it Uses, so definition order runs opposite to its Use; an After needs an earlier handle, and a Use doesn't.
- web: noted for intake: an After that mixes a group and a node, `graph.After(n.Reactors, n.Schema)`, reads cleanly.
- web: noted for intake: LayerOf panics for a non-member, so the layer guards scan Layers(); a Has(ref) or a LayerOf returning (int, bool) would serve them.
- web: noted for intake: no System method lists the built nodes; the tests flatten Layers() with slices.Concat.
- web: noted for intake: observers persist across Builds of one Graph, a trap for tests that build twice.
- web: noted for intake: Dropped() is the only signal that a partial Build lost an After.
- web: noted for intake: the runner takes four handles plus the roots; a promoted runner might take a struct or options.
- web: noted for intake: the sealed exported Ref is a clean reflection seam for tooling such as graphtest.
- web: noted for intake: graphtest relies on Define, Value, Fresh, and Group refusing the empty name, so only a zero-value literal has one; the refusal is part of the contract.
- web: noted for intake: Fresh's per-Build zero value is proven by sdk/graph's tests, not by the service's.
- web: noted for cli: slab's name check reuses sdk/graph/graphtest.
- web: noted for cli: internal/config still imports go-core's lifecycle for Config, which sdk/lifecycle aliases.

## Pending edits

- coordinator · experiment.graph intake: plan go-web-service's dependency upgrades to the versions spike-graph adopted in its baseline.
- coordinator · roadmap: spike-graph's summary says lifecycle forks Coordinator, New, and Readiness, and the forks live under sdk/.
- coordinator · roadmap: the candidate task's summary says `graph.Fresh`; it is the method `g.Fresh[T]`.
