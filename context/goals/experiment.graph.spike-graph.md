# goal · experiment.graph.spike-graph

- **State:** building
- **Task:** cli
- **Branch:** cli

## Tasks

1. [x] baseline
2. [x] candidate
3. [x] web
4. [ ] cli
5. [ ] adapters

## Task brief · cli

**Problem**
Evidence 4 is still open. slab runs on go-core's `graph` and `lifecycle` and on published go-cli-sdk v0.2.0, so the candidate API hasn't been tried on a real CLI yet. Three things are needed:
- slab has to reach the spike's candidate packages without a replace directive;
- go-cli-sdk's `cli` and `scenario` packages have to be forked onto the candidate graph;
- slab's composition root has to be migrated.

The migration must keep slab's guarantee that help and usage errors build nothing, and its scenarios must still run against the service the web task migrated. The adapters task measures from this result.

**Behaviors**
1. slab is part of the root module. One `mise run check` builds, vets, formats, tidies, tests, and lints it alongside the service. `mise run slab` still runs it with the spike's `[env]` settings, and `mise run currency` exits 0.
2. The forked `cli` package keeps go-cli-sdk v0.2.0's behavior and runs on the candidate graph and lifecycle. Its ported tests pass, covering:
   - the dispatch order;
   - Use inherited along the command path and built once;
   - the lifecycle config node added to every Build;
   - help, unknown commands, flag errors, argument counts, required and exclusive flags, Validate, and a PreRun usage error, all building nothing;
   - failures reported once with the right exit code;
   - the shutdown timeout read from the config node;
   - the panics for wiring mistakes;
   - flags restored across dispatches.
3. A value that fails while a command runs is reported as `run: <node>: <error>`, naming its node, the same way the candidate lifecycle reports it for the service. This is the one intended difference from v0.2.0.
4. The forked `scenario` package keeps v0.2.0's behavior on the candidate graph: the scenario parent, one leaf per scenario declaring its nodes, the listing, narrated steps, and the step-failure report. Its ported tests pass.
5. slab imports nothing from go-core's `graph` or `lifecycle`, and nothing from go-cli-sdk.
6. Root help, a parent run alone, every usage error, and version build no node and read no configuration. The observer records nothing even when every SLAB_* setting can't be read.
7. A command builds exactly the nodes its path declares, plus the lifecycle config node. A setting is read only by a command that declares it. A flag overrides its SLAB_* variable, which overrides the default. A shutdown timeout that can't be parsed fails any command that declares nodes.
8. Order and constructors are stated at Define:
   - grafana up's order after service is stated where grafana up is defined, and no constructor body orders anything;
   - client, broker, service, and grafana up are built by methods on Nodes, following the web task's precedent;
   - the seven run settings take their constructors from a `setting(name, default, parse)` method on the root flags;
   - no wrapper constructor or Define closure remains.
9. The lifecycle config node is the node the dispatcher configures the Coordinator from, and it reads SLAB_SHUTDOWN_TIMEOUT.
10. The shared name check from the candidate graph's test package runs on slab's Nodes and passes, and the name comments are gone.
11. For every scenario, building its declared nodes plus the lifecycle config node gives a System where Dropped() is empty and grafana up sits a layer above service whenever both are present. A failure prints the layer table.
12. With nothing running, a scenario that declares both service and grafana up stops at startup on service and names `mise run serve`.
13. On the stack, with the migrated service running, all six scenarios run to completion through `mise run slab -- scenario <name>`: sqlate, domain, problems, storage, events, and sweeper.

**Test seams**
- `cli.Run` over buffered Streams, with an observed candidate graph (behaviors 2–4).
- slab's App: `New` with buffered Streams, then `Run(ctx, args)`, observed through `App.Graph()` and `App.Nodes()` (behaviors 5–12).
- The scenarios against the stack (behavior 13).

**Slices**
1. **Fold slab into the root module.**
   - slab's go.mod is removed, and no go.work is added.
   - The root takes slab's pins: nats.go v1.55.0, x/sys v0.50.0, x/term v0.48.0. x/sys becomes direct.
   - GO_MODULES becomes the root alone, and the slab task runs from the root.
   - slab still runs on go-core and go-cli-sdk.
   - Demo: check passes and covers slab, currency exits 0, and `mise run slab -- --help` prints root help.
2. **Fork cli and scenario.**
   - go-cli-sdk v0.2.0's `cli` and `scenario` are forked whole to sdk/cli and sdk/cli/scenario.
   - The only changes are the imports and the monitored-failure test and doc.
   - Demo: the ported tests pass in check (behaviors 2–4).
3. **Migrate slab onto the candidate.**
   - Every slab package moves to the candidate packages.
   - grafana up's After moves to its Define.
   - The four Nodes methods and `flags.setting` replace the wrapper constructors and closures.
   - apptest's observer takes handles.
   - go-cli-sdk leaves the root's requirements.
   - Demo: check passes with what the existing tests assert unchanged, the import listing is clean, and help and a usage error build nothing (behaviors 5–9 and 12).
4. **Name check and layer test.**
   - Demo: check passes, and each test fails readably when a name or an After is broken by hand (behaviors 10–11).
5. **Scenarios on the stack.**
   - slab's package docs say it runs on the candidate.
   - Demo: with `mise run stack:up`, `otel:up`, and `mise run serve`, the six scenarios run, and their transcripts are kept as evidence (behavior 13).

**Out of scope**
- Arity adapters, the before/after table, and the decision marks (the adapters task).
- Any change to go-cli-sdk, go-core, go-cli-sdk-template, or go-web-service.
- Command families beyond their imports.
- The service config package's go-core lifecycle import.
- An integration tier for slab.

**Door**
Two-way. Only the spike repository changes, nothing is released, and each slice can be reverted from git.

## Progress

slices 5/5 committed · standards ✓ · spec ✓ · editor ✓

## Decisions

- setup: full copies of go-web-service and slab at 5bb590e, proven against the spike's own copy of go-web-service's compose stack, not cut-down miniatures with fakes.
- setup: the copies are committed unchanged as a baseline on go-core v0.10.0, then migrated in place; the before/after table comes from that diff. Superseded by baseline: the copy's dependencies are upgraded, and the measurement baseline is the baseline task's merge commit.
- setup: fork go-core's `graph` whole; fork only `lifecycle`'s Coordinator and New and alias the rest, so published go-web-sdk works unchanged; fork go-cli-sdk v0.2.0's `cli`; `web.Run` in the spike's own `web` package.
- setup: the stack is the spike's own copy (own project name and ports, observability profile included), never go-web-service's dev stack.
- baseline: the copied service is the root module github.com/JaimeStill/spike-graph; slab stays its own module .../tools/slab. Superseded by cli: slab folds into the root module.
- baseline: stack ports come from the mise environment; the copied files change only by the path rewrite, gofmt import order, and the compose project name.
- baseline: the copy's dependencies and image pins are upgraded to latest; go-web-service's own upgrades are planned at intake.
- baseline: held go-core at v0.10.0: the candidate's fork point; latest v0.10.0.
- baseline: go-web-service's currency script replaces the spike's.
- baseline: go-web-service's repository-level files and CI are left out; package-level docs are copied as-is.
- baseline: slab's scenarios run once on the baseline as demo evidence.
- baseline: the measurement baseline is this task's merge commit; the copy-as-is commit stays in history.
- baseline: noted for cli: slab must reach the candidate forks in the root module without a replace directive, by folding slab into the root module or a go.work; folding puts slab on the root module's sqlate version. Superseded by cli: slab folds into the root module, with no go.work.
- baseline: [env] sets APP_ENV=local, so serve reads config.local.json.
- baseline: [env] derives every APP_* and SLAB_* setting from the port variables, so each port is set in one place.
- baseline: the stack tasks read only the spike's [env]; no task falls back to go-web-service's ports.
- baseline: integration keeps [env]'s APP_OBSERVABILITY_ENDPOINT; its telemetry reaches the spike's dev collector when that is up, as go-web-service's does.
- baseline: go-web-service's vet, test, fmt, tidy, and lint tasks are not copied; the spike's one check covers them.
- baseline: the root module's indirect golang.org/x/sys stays at v0.48.0; currency doesn't scan indirect requirements. Superseded by cli: the root takes slab's x/sys v0.50.0 as a direct requirement.
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
- web: noted for cli: slab's name check reuses sdk/graph/graphtest. Done in cli: slab's graph test runs graphtest.CheckNames on its Nodes.
- web: noted for cli: internal/config still imports go-core's lifecycle for Config, which sdk/lifecycle aliases. Still true after cli: the brief left that import out of scope, and slab imports no go-core graph or lifecycle.
- cli: slab folds into the root module; its go.mod is deleted and no go.work is used. Supersedes baseline's "slab stays its own module".
- cli: folding takes slab's pins into the root (nats.go v1.55.0, x/sys v0.50.0, x/term v0.48.0), so currency stays at exit 0, and x/sys becomes direct. Supersedes baseline's hold of the root's indirect x/sys at v0.48.0.
- cli: go-cli-sdk v0.2.0's scenario is forked with cli, at sdk/cli/scenario beside sdk/cli. Both are forked whole, changing only imports and the monitored-failure test, which expects sdk/lifecycle's "run: <node>: <error>".
- cli: slab's seven run settings take their constructors from a flags.setting(name, default, parse) method on the root flags, as the design note's Sketch 2 proposes.
- cli: slab gains one layer test over every scenario: Dropped() is empty, grafana up sits a layer above service, and a failure prints the layer table.
- cli: client, broker, service, and grafana up are built by methods on Nodes, following the web task's precedent.
- cli: slab's --help exits 2 (ExitUsage), v0.2.0's behavior; the fork doesn't change it.
- cli: flags.setting is a generic method on *rootFlags, which Go 1.27's method type parameters allow; each setting is one Define line.
- cli: newLifecycleConfig stays a plain function passed to Define; it reads no node, so a *Nodes receiver would go unused.
- cli: repo's nested-module test fixture stays, so the module walk still proves it passes over a nested module that declares another path.
- cli: sdk/cli's doc drops v0.2.0's depguard claim; the spike enforces no depguard on sdk/cli.
- cli: slab's layer test reads the registered scenarios through scenarios.All over App.Nodes(), not an export_test hook; tests-and-docs allows a hook only for a clock, probe, or fixture node.
- cli: the layer test fails on any dropped After, so a scenario that declares grafana up without service fails it; scenario.go's comment says so.
- cli: noted for intake: slab's migration needed no API change beyond Scope.After → graph.After and Observe(func(Ref)); the cli fork compiled on import rewrites alone.
- cli: noted for intake: cli.Command exposes no accessor for its children.
- cli: noted for intake: nothing in sdk/cli reads System.Dropped, so only slab's layer test catches a scenario's dropped After.
- cli: noted for adapters: slab's wrapper constructors are replaced: newSetting by flags.setting (7 settings), newClient and newBroker by *Nodes methods, and the service and grafana up Define closures by *Nodes methods.

## Pending edits

- coordinator · experiment.graph intake: plan go-web-service's dependency upgrades to the versions spike-graph adopted in its baseline.
- coordinator · roadmap: spike-graph's summary says lifecycle forks Coordinator, New, and Readiness, go-cli-sdk's `scenario` is forked with `cli`, and the forks live under sdk/.
- coordinator · roadmap: the candidate task's summary says `graph.Fresh`; it is the method `g.Fresh[T]`.
- coordinator · roadmap: the cli task's summary says it folds slab into the root module and forks go-cli-sdk v0.2.0's `cli` and `scenario`.
