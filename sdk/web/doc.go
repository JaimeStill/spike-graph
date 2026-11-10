// Package web is the spike's candidate runner for a long-running web
// service built on the candidate graph (sdk/graph) and run by the candidate
// lifecycle (sdk/lifecycle): the service's counterpart to go-cli-sdk's
// cli.Run, which does the same for a command.
//
// The package exports:
//
//   - [Run], which builds a service's root set and the lifecycle config
//     node, runs the System under a lifecycle Coordinator until its context
//     ends, reports the outcome on the logger node or, when the Build
//     fails, on stderr, and returns the process exit code
//   - [Server], the constraint on the server node's value: it reports the
//     address it bound
//
// The package name is the library it is bound for, go-web-sdk, whose own
// package is also named web; a file that imports both aliases one.
package web
