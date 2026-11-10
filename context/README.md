# spike-graph

spike-graph proves the candidate API for go-core's `graph` and `lifecycle` packages, set out in
go-core's design note (`go-core/context/graph-ergonomics.md`, "The candidate API" and "The open
fork"), on full copies of go-web-service and its slab CLI, run against real infrastructure. Its
answer is evidence for standards-lab's experiment.graph intake, which decides the API go-core's
breaking release ships and graph-ergonomics migrates its consumers to; the spike decides nothing
itself.

The spike is the sub-goal `experiment.graph.spike-graph` in the coordinator's roadmap,
`standards-lab/context/roadmap.toml`, where its tasks live. Its answer lands in
`standards-lab/context/graph-ergonomics.md`. It is a prerequisite of capstone.data-intelligence.

## The question

Does the candidate API, built on today's graph semantics, make go-web-service's real composition
root and slab's real CLI shorter, and their layer placement legible, run against real
infrastructure, with arity adapters and methods on `*Nodes` compared on one tier?

Its answer decides the exact API go-core's breaking release ships: which of graph-ergonomics'
decisions 9–14 survive as written, whether arity adapters ship, and three details the design note
leaves open: the rule the name check applies; what `web.Run` returns and reports, and where the
ready, failed, and stopped logs go; and how a `*Nodes` method reaches a plain value.

## The evidence

1. A candidate `graph` package, forked from go-core v0.10.0, implements `graph.After` and
   `graph.In` at Define, groups a Build expands, `g.Value` and `g.Fresh` with Define's options,
   `Ref` on Dependency, `Observe(func(Ref))`, `LayerOf`, `String`, and `Dropped`, with
   `Scope.After` removed; go-core's graph and lifecycle tests, ported, pass: edges discovered
   through Use, build-once, longest-path layers, panics for wiring mistakes, the participation
   interfaces, and startup, readiness, and shutdown.
2. go-web-service's whole service runs on the candidate: its composition root migrated, every
   node's layer stated at its Define line and no constructor body ordering anything; its unit
   tests, and its integration tests against the spike's Postgres, Azurite, NATS, and otel stack,
   pass.
3. A candidate `web.Run` replaces the root's ~30-line Run, settling what it returns, how it
   reports ready and failed, and who logs ready, failed, and stopped.
4. slab, migrated: its commands declare their nodes, and the lifecycle config is a node on both
   sides; help and usage errors build nothing; its scenarios run against the migrated service on
   the stack.
5. The layer tests are rewritten on `Roots()`, `LayerOf`, and handles: `buildAsRun`, `layerOf`,
   and `names` are gone, a failing test prints the layer table, and the full root set drops no
   `After` (`Dropped()` is empty).
6. A fixture name check replaces the `// "name"` comments, with its rule settled.
7. How a `*Nodes` method reaches a plain New argument (the logger's `io.Writer`) is settled.
8. A verdict on whether arity adapters ship. The plan was to write one tier both as `*Nodes`
   methods and with arity adapters; the architect decided without it (The answer, 8).
9. A before/after table for each root, from the diff against the baseline commit: lines,
   closures, wrapper constructors, name comments, and ordering sites, with decisions 9–14 each
   marked kept, changed, or dropped.

## How it is built

- **The roots.** The root module, `github.com/JaimeStill/spike-graph`, holds go-web-service and
  slab, both copied whole at 5bb590e. slab is a package tree under `tools/slab` with no `go.mod`
  of its own, so it reaches the candidate packages without a replace directive. The copies differ
  from 5bb590e only in the module path, gofmt's import order, the compose project name, and
  dependencies and image pins upgraded to latest, with go-core held at v0.10.0. The baseline
  task's merge commit is the measurement baseline. Later tasks migrate the roots in place, so the
  tree holds one version of each file and the measurements come from the diff against that commit.
  Both composition roots, the service's `internal/app` and slab's `tools/slab/internal/app`, run on
  the candidate packages. The service's `internal/config` imports go-core's `lifecycle` for
  Config, which `sdk/lifecycle` aliases; no other package outside `sdk/` imports go-core's `graph`
  or `lifecycle`.
- **The candidate packages.** Each candidate library package lives under `sdk/<package>`, beside
  the copied service's own `sdk` package. Everything else is a published version, never a replace
  directive.
  - `sdk/graph` is go-core v0.10.0's `graph`, forked whole. Its `graphtest` package holds the
    name check, `CheckNames`, which the service's and slab's tests run on their Nodes.
  - `sdk/lifecycle` forks Coordinator and New, which take graph's types, and Readiness, since
    go-core's Readiness binds only through go-core's unexported bind. Check, Config, and the
    participation interfaces alias go-core's, so published go-web-sdk, whose `server.go` and
    `health.go` name them, works unchanged. It diverges from v0.10.0 in two ways: New rejects
    go-core's Readiness, and a Monitored value's runtime failure names its node, as
    "run: <node>: <error>".
  - `sdk/web` holds `web.Run`, the service runner, which App.Run calls. The service imports it as
    `sdkweb`, since go-web-sdk's package is also `web`.
  - `sdk/cli` and `sdk/cli/scenario` are go-cli-sdk v0.2.0's `cli` and `scenario`, forked whole
    because `WithGraph` takes a graph. They differ from v0.2.0 only in their imports and in one
    behavior: a value that fails while a command runs is reported as "run: <node>: <error>", as
    `sdk/lifecycle` reports it. slab runs on them.
- **The stack.** The spike runs go-web-service's compose stack (Postgres, Azurite, NATS, and the
  observability profile) as project `spike-graph`, and the integration tier runs it as
  `spike-graph-integration`. `mise.toml`'s `[env]` sets every host port and the service's and
  slab's settings, so the copied files keep go-web-service's defaults; run the service and slab
  through `mise run`.
- **The measurement.** `mise run measure` runs `tools/measure`, which parses each composition
  root's production files and its `graph_test.go` at the baseline commit (2999659) and in the
  working tree, and prints a before/after table for each root. It reads the baseline through git,
  leaving the checkout untouched. Its `count` package defines each column.

## The path

The sub-goal's tasks, in order: `baseline` (the copies and the stack, green on go-core v0.10.0),
`candidate` (evidence 1), `web` (2, 3, 5, 6, 7), `cli` (4), and `adapters` (8, and 9 through
`mise run measure`), which gives the answer.

## The answer

**Question.** Does the candidate API, on today's graph semantics, make go-web-service's
composition root and slab's CLI shorter, and their layer placement legible, run against real
infrastructure, with arity adapters and methods on `*Nodes` compared on one tier?

**Answer.** Yes: both real roots are shorter and their layer placement legible, with every node's
layer stated at its Define line, no constructor body ordering anything, and the service root's
wrapper constructors, closures, and name comments gone; arity adapters don't ship, so methods on
`*Nodes` are the only constructor form.

### Where the evidence is proven

1. sdk/graph's and sdk/lifecycle's tests, go-core's ported and the candidate's own, under
   `mise run check`; both roots run on the packages (2, 4).
2. internal/app's tests under `mise run check`; `mise run integration` passed on the
   spike-graph-integration stack in the web task; the migrated service served the cli task's
   slab scenarios on the spike-graph stack (4).
3. sdk/web's tests (a ready then clean stop, startup, runtime, and Build failures) and
   internal/app's Run tests; the running service logged "server ready" in the cli task. "service
   failed" and "server stopped" are proven only by tests.
4. `mise run slab` ran sqlate, domain, problems, storage, events, and sweeper against the
   migrated service on the spike-graph stack in the cli task, all exiting 0. That help and usage
   errors build nothing is proven only by tests: slab's and sdk/cli's.
5. Proven only by tests: internal/app's layer tests and slab's `TestGraph_ScenarioLayers`, whose
   guards each fail on a fixture that breaks them and print the layer table.
6. Proven only by tests: `graphtest.CheckNames` and its own tests, run on both roots' Nodes by
   `TestGraph_NodesNamedByTheirFields`.
7. The logger Uses the "writer" node, a `graph.Value`: internal/app's tests prove it, and the
   running service of 2 and 3 logs through it.
8. Decided, not measured: no tier was written with arity adapters. The architect decided they
   don't ship. The Define-line changes already made the graph legible and fixed the reliability
   findings, so a second constructor form isn't worth its cost: one adapter per arity, a
   projection node or wrapper for every constructor that takes part of the config (five of the
   service's seven infrastructure constructors do), and no-error variants. If constructors become
   a problem, the way nodes are defined can be revisited.
9. `mise run measure`, the baseline commit (2999659) against the working tree:

   | service (`internal/app`) | before | after | delta |
   | --- | --: | --: | --: |
   | code lines | 460 | 411 | −49 |
   | raw lines | 1034 | 1015 | −19 |
   | closures | 21 | 0 | −21 |
   | wrapper constructors | 14 | 0 | −14 |
   | name comments | 21 | 0 | −21 |
   | `s.After` sites | 7 | 0 | −7 |
   | `graph.After` sites | 0 | 6 | +6 |
   | layer tests, code lines | 392 | 423 | +31 |
   | layer tests, raw lines | 565 | 615 | +50 |

   | slab (`tools/slab/internal/app`) | before | after | delta |
   | --- | --: | --: | --: |
   | code lines | 207 | 202 | −5 |
   | raw lines | 438 | 438 | 0 |
   | closures | 11 | 7 | −4 |
   | wrapper constructors | 3 | 1 | −2 |
   | name comments | 12 | 0 | −12 |
   | `s.After` sites | 1 | 0 | −1 |
   | `graph.After` sites | 0 | 1 | +1 |
   | layer tests, code lines | 0 | 89 | +89 |
   | layer tests, raw lines | 0 | 140 | +140 |

   The rows above the layer tests count each root's production files, leaving out doc.go. Code
   lines exclude blank, comment, and import lines. Raw lines barely move because the doc comments
   were rewritten to the spike's standards. The layer tests, each root's `graph_test.go`, grew on
   purpose: they gained handles, the layer table printed on failure, and the Dropped guard, and
   slab's layer test is new. slab's remaining wrapper is `flags.setting`, which returns a
   constructor as Sketch 2 proposes. slab's 7 remaining closures are that constructor; `env`, a
   local helper `bindRootFlags` uses to name each flag's variable; `preRun`'s `VisitAll`
   callback; and four callbacks the CLI takes: the root's `Footer`, `version`'s `Run`, and
   `outputSource`'s `New` with the color check nested in it.

### Decisions 9–14

- **9, order and groups are declared at Define: kept, with additions.** `System.Dropped()` lists
  each After that ordered nothing, and After needs a handle defined earlier, so the service
  defines its telemetry layer first.
- **10, constructors are methods on `*Nodes`: kept.** `flags.setting` returns a constructor, as
  Sketch 2 proposes. The service's `newLifecycleConfig` is a `*Nodes` method like the other
  constructors; slab's `newLifecycleConfig` and the service's `newMeterProvider` stay plain
  functions. The open fork, arity adapters, is dropped by the architect's judgment, not measured
  (8).
- **11, values and fresh values: changed.** Fresh is the method `g.Fresh[T](name)`; Value and
  Fresh take Define's options, so a participant can join a group; a `*Nodes` method reaches a
  plain value through a `graph.Value` node.
- **12, names stay explicit: kept, rule settled.** `graphtest.CheckNames` checks that a node's or
  group's name, lowercased with spaces removed, equals its Nodes field's name lowercased.
- **13, introspection exposes handles: kept, with additions.** `Dropped`; `LayerOf` panics for a
  node not in the System, a nil ref, and a group handle.
- **14, one runner on each side: changed.** `web.Run[S Server](ctx, g, lifecycleConfig, logger,
  server, roots, stderr) int` logs "server ready", "service failed", and "server stopped" on the
  logger node and returns the exit code, as `cli.Run` does; the design note had it return an
  error and left the logs open.

The three details the design note leaves open are settled: the name check's rule (12); `web.Run`
returns the exit code and logs ready, failed, and stopped on the logger node, a Build failure
going to stderr (14); and a `*Nodes` method reaches a plain value through a `graph.Value` node
(11).

The goal record's "noted for intake" decisions are open findings for intake. The main ones:

- `LayerOf` panics for a non-member, so the guards scan `Layers()`; a `Has(ref)` or an
  `(int, bool)` form would serve them, and no System method lists the built nodes.
- `Dropped()` is the only signal that a partial Build lost an After, and nothing in sdk/cli
  reads it.
- `g.Value` infers `T` from the argument's static type.
- Observers persist across Builds of one Graph; concurrent Builds race on `Graph.built`.
- Replace can't remove an edge or a membership, and there is no `Group.Members()`.
- The runner takes four handles and the roots; a promoted runner might take a struct or options.

## References

Read, never written, keyed in the coordinator's `references.toml`: go-core (with its design
note), go-cli-sdk, go-web-sdk, go-web-sdk-template, go-web-service (with slab), and architecture.
