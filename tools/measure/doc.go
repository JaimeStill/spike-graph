// Command measure prints, for each of the spike's two composition roots,
// the service's internal/app and slab's tools/slab/internal/app, a table
// comparing the baseline commit against the working tree: the code and raw
// lines of the root's production files and the graph shapes they write,
// and the code and raw lines of its layer tests.
//
// The baseline is read from git (ls-tree and show), so the checkout and the
// working tree are never touched. measure runs from anywhere in the
// repository: `mise run measure`.
package main
