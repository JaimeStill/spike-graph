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
8. One tier is written both as `*Nodes` methods and with arity adapters, with a verdict on
   whether adapters ship.
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

## The path

The sub-goal's tasks, in order: `baseline` (the copies and the stack, green on go-core v0.10.0),
`candidate` (evidence 1), `web` (2, 3, 5, 6, 7), `cli` (4), and `adapters` (8, 9), which gives
the answer.

## References

Read, never written, keyed in the coordinator's `references.toml`: go-core (with its design
note), go-cli-sdk, go-web-sdk, go-web-sdk-template, go-web-service (with slab), and architecture.
