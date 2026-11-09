# goal · experiment.graph.spike-graph

- **State:** idle
- **Task:** none
- **Branch:** none

## Tasks

1. [x] baseline
2. [ ] candidate
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

## Pending edits

- coordinator · experiment.graph intake: plan go-web-service's dependency upgrades to the versions spike-graph adopted in its baseline.
