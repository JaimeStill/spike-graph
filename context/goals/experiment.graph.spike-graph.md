# goal · experiment.graph.spike-graph

- **State:** building
- **Task:** candidate
- **Branch:** candidate

## Tasks

1. [x] baseline
2. [ ] candidate
3. [ ] web
4. [ ] cli
5. [ ] adapters

## Task brief · candidate

```
Problem       The web, cli, and adapters tasks migrate the roots onto a candidate
              graph and lifecycle that doesn't exist yet. Decisions 9-13 of go-core's
              design note need an implementation forked from go-core v0.10.0, proven
              by go-core's own tests, before any root can be measured against the
              baseline.
Behaviors     1. Define takes options. graph.After(refs...) orders the node after each
                 ref in the System: it starts after the ref and stops before it. It
                 never builds the ref, and a ref outside the System is dropped, as
                 Scope.After does today.
              2. g.Group(name) returns a group handle that is a Ref; graph.In(group)
                 makes a node a member. A Build given a group as a root builds every
                 member. graph.After(group) orders after each member in the System.
                 Membership is complete when Build expands it, and an empty group
                 builds and orders nothing.
              3. Scope has Use only; Scope.After is gone.
              4. g.Value(name, v) defines a node whose value is v in every Build.
              5. g.Value(name, v, opts...) and g.Fresh[T](name, opts...) take the
                 same options as Define. Fresh is a method; its node's value is a new
                 zero *T in each Build, so two Builds give two distinct pointers.
                 Value's doc says a lifecycle participant or a Readiness belongs in
                 Fresh or Define, since Value hands every Build the same value.
              6. Dependency is {Name, Value, Ref}. Ref equals the handle Define,
                 Value, or Fresh returned.
              7. Observe(func(Ref)) fires exactly when it does today: once for each
                 node a Build begins, before its constructor, depth-first, including
                 a node whose constructor fails. It passes that node's handle.
              8. System.LayerOf(ref) returns the node's layer. It panics with a
                 "graph: " message for a node not in the System, and for a group.
              9. System.String() prints the layer table, one line per layer: the layer
                 number, then the node names in definition order.
              10. Wiring mistakes still panic with "graph: " messages: every v0.10.0
                  case, plus these. After or In given a nil ref or one from another
                  Graph panics at Define. A Group after Build, or with an empty or
                  duplicate name, panics; groups share the node namespace. A cycle
                  closed by order-only edges, through a group included, panics at
                  Build with its path.
              11. Replace swaps only the constructor: the node keeps the After and In
                  it was defined with.
              12. The forked lifecycle's New takes a candidate System and returns a
                  Coordinator with all of v0.10.0's behavior: startup, readiness,
                  OnReady, Monitor and Monitored, Exec, Run, shutdown. Readiness is
                  forked and bound by New. Check, ReadinessChecker, Config, Env,
                  NewEnv, Starter, Stopper, Subsystem, ReadyStarter, and Monitored
                  alias go-core's, so a go-web-sdk Server takes part, and the forked
                  Coordinator and Readiness each satisfy go-web-sdk's Doctor.
              13. go-core's graph and lifecycle tests, ported with Scope.After rewritten
                  as graph.After at Define, pass under -race. Config's tests stay
                  with go-core, since Config is aliased.
              14. The candidates live under sdk/ as sdk/graph and sdk/lifecycle. The
                  roots are untouched and still on go-core v0.10.0, and
                  `mise run check` passes.
              15. System.Dropped() reports each After that orders nothing in the
                  System: a node target not in it, or a group with no member in it,
                  as the declaring node's and the target's handles. A full build
                  reports none.
              16. A zero-value Node or Group literal given to After, In, Build, or
                  LayerOf panics with a "graph: " message. A nil After or In ref's
                  message says its Define may come later. lifecycle.New panics when
                  a node's value is go-core's *Readiness, which it can't bind.
Test seams    The candidate graph's exported API (Graph, Node, Group, Scope, System),
              and the forked lifecycle's New over a System the candidate graph built.
Slices        (no upgrade slice: spike-graph is current; go-core is held)
              1. Fork graph verbatim into sdk/graph, with go-core's graph tests
                 ported and passing: the fork point, green.
              2. Order and groups at Define; Scope.After removed; the After tests
                 rewritten; new tests for groups and the Define-time panics.
              3. Value and Fresh.
              4. Introspection: Dependency.Ref, Observe(func(Ref)), LayerOf, String.
              5. Fork lifecycle's Coordinator, New, and Readiness into sdk/lifecycle,
                 aliasing the rest; go-core's lifecycle tests ported and passing,
                 plus a compile-time check that the forked Coordinator and Readiness
                 are go-web-sdk Doctors.
              6. Options on Value and Fresh; Fresh as a method (Fable review).
              7. System.Dropped() (Fable review).
              8. Zero-literal handles, the nil-After hint, and the go-core Readiness
                 guard (Fable review).
Out of scope  Migrating the composition root and its layer tests, web.Run, the name
              check, and how a *Nodes method reaches a plain value (web); the cli fork
              and slab (cli); arity adapters and the measurements (adapters);
              any change to go-core or to the copied sdk package.
Door          two-way: new packages in the spike repository that nothing imports yet;
              no release.
```

## Progress

slices 5/8 committed · standards — · spec — · editor — (redirected by the Fable review; slices 6–8 added)

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
- candidate: noted for web: After needs a handle defined earlier; telemetry is defined after database, so the root defines telemetry first or forward-declares a group; the root's layer test asserts Dropped() is empty for the full root set.
- candidate: noted for intake: concurrent Builds race on Graph.built (documented, as in go-core); Replace can't remove an edge or membership; no Group.Members(); String is ambiguous for names with ", "; an observer's panic escapes Build.

## Pending edits

- coordinator · experiment.graph intake: plan go-web-service's dependency upgrades to the versions spike-graph adopted in its baseline.
- coordinator · roadmap: spike-graph's summary says lifecycle forks Coordinator, New, and Readiness, and the forks live under sdk/.

