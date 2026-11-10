# spike-graph

A spike for the Standards Lab reference architecture. It proves a candidate API for go-core's
`graph` and `lifecycle` packages on full copies of go-web-service and its slab CLI, run against
real infrastructure. The repository is managed with the marathon workflow; start from
`context/README.md`.

- **Standards:** `STANDARDS.md`, with pointers into the architecture repository.
- **Check:** `mise run check` (hermetic). `mise run integration` runs the integration tier on its
  own compose stack; `mise run measure` prints the composition roots' before/after tables;
  `mise.toml` defines the stack, serve, and slab tasks.
- **Goals:** this spike's tasks live in the coordinator's roadmap,
  `standards-lab/context/roadmap.toml`, under `experiment.graph.spike-graph`. Its goal record is
  `context/goals/experiment.graph.spike-graph.md`.
- **Dependencies:** published versions only, never a `replace` directive.
- **References:** the repositories this spike reads are keys in the coordinator's
  `references.toml` and `references.local.toml`. The spike reads them and never writes to them.
