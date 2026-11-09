# Standards

The judgement calls the check can't enforce, one line each, for the standards-reviewer. What
`mise run check` enforces is never restated here.

The standard this spike follows is go-elemental. The architecture pages that apply, in the
architecture repository:

- `standards/go-elemental/README.md`
- `principles/composition-root.md`
- `principles/composition-terms.md`
- `standards/go-elemental/principles/lifecycle-and-context.md`
- `standards/go-elemental/principles/domain-files.md`
- `standards/go-elemental/principles/tests-and-docs.md`

## Composition

- The copied service and slab change only where the candidate API reaches them: composition
  roots, their fixtures, and the code that runs a graph. Everything else stays as copied, so the
  diff against the baseline commit measures the API.
- The forked packages keep go-core's and go-cli-sdk's behavior; a change beyond the candidate API
  is recorded as a decision in the goal record.
