# goal · experiment.graph.spike-graph

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] baseline
2. [x] candidate
3. [ ] web
4. [ ] cli
5. [ ] adapters

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

## Pending edits

- coordinator · experiment.graph intake: plan go-web-service's dependency upgrades to the versions spike-graph adopted in its baseline.
- coordinator · roadmap: spike-graph's summary says lifecycle forks Coordinator, New, and Readiness, and the forks live under sdk/.

