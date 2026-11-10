// Package graphtest is the test toolkit for a program built on the graph
// package. It checks what a program's own description of its graph says
// about the nodes, so a test holds the description to the graph rather
// than a comment repeating it.
//
// The package exports:
//
//   - [CheckNames], a test's check that each handle field of a struct of
//     node and group handles is defined, under a name that is its field's
//   - [NameErrors], the errors CheckNames reports, for a caller that
//     handles them itself
package graphtest
