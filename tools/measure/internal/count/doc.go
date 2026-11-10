// Package count measures one Go source file of a composition root: the
// lines it spends on code and the shapes the graph API asks it to write.
// It is a pure function over source, bytes in and counts out, so measure's
// git plumbing stays in its main package and the counters are tested on
// small sources with known counts.
//
// The package exports:
//
//   - [Counts], the measurements of one file or a sum of files
//   - [File], which measures one file's source
//   - [Lines], which counts only the code and raw lines, for files (the
//     layer tests) whose shapes are not measured
package count
