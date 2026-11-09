// Package graph is a typed dependency graph. A [Graph] describes [Node]
// values inertly, each a name and a constructor; [Graph.Build] constructs
// what its roots reach and returns it as a [System] of [Dependency] values
// in computed layers. The package knows nothing of lifecycles: a consumer
// that runs a System, starting its layers in order and stopping them in
// reverse, reads each node's part from its value, as the lifecycle package's
// Coordinator does.
//
// The package exports:
//
//   - [Graph], the description of the nodes, and [New], which returns an
//     empty one
//   - [Graph.Define], which adds a node with its constructor and its
//     options, and runs nothing
//   - [Graph.Value], which adds a node whose value is a given value in
//     every Build, and [Fresh], which adds a node whose value is a new zero
//     pointer in each Build; both take no options and return a Node like
//     Define's
//   - [Option], which places a node at Define: [After], which orders it
//     after nodes or groups without their values, and [In], which makes it
//     a member of a group
//   - [Graph.Group], which defines a [Group], a named set of nodes, and
//     [Group.Name], its name
//   - [Graph.Replace], which swaps a node's constructor before Build, for a
//     test's substitute, keeping its options
//   - [Graph.Observe], which adds a function a Build calls with each node's
//     handle as it begins to construct it, for tracing what a Build reaches
//   - [Graph.Build], which constructs what the roots reach into a System
//   - [Node], a typed handle on one node, and [Node.Name], its name
//   - [Ref], any Node whatever its type or a Group, sealed to the two, and
//     its Name
//   - [Scope], what a constructor receives
//   - [Scope.Use], which returns a dependency's value, building it if
//     needed
//   - [System], what one Build constructed
//   - [System.Get], which returns a node's value
//   - [System.Layers], which returns the built nodes in layers
//   - [System.LayerOf], which returns one node's layer
//   - [System.String], which prints the layer table, one line per layer,
//     for a failing test's message
//   - [Dependency], one built node: its name, value, and handle
//
// # Discovery
//
// A constructor declares its dependencies by using them: [Scope.Use] builds
// the dependency if this Build has not, and records the edge, so the edges
// are discovered as the constructors run, depth-first from the roots, and
// can depend on what the constructors see. Each node is built at most once
// per Build, so a node shared by several dependents is constructed once and
// they all receive its value. A node that is neither a root nor reached by
// a Use is never constructed, so a Build of a subset of the roots brings up
// only that subset's dependencies. Each Build is independent and constructs
// fresh values: a [Graph.Value] node's value is the same v in every Build,
// and a [Fresh] node's is a new zero pointer in each, so a value a run
// mutates starts zero in the next System.
//
// # Order and groups
//
// A node's order-only edges are declared at Define, beside its name, and
// not in its constructor: [After] orders the node after each ref it names.
// An order-only edge passes no value and builds nothing, and holds only
// when its target is in the System by some other path, so a node only
// ordered against is named as a root. [Graph.Group] defines a [Group], and
// [In] makes a node its member. A group is a [Ref] that stands for its
// members: a Build given a group as a root builds every member, in
// definition order, and After of a group orders after each member in the
// System. The membership is complete by the first Build, since Define
// panics after it, so a node may order after a group whose members are
// defined later; an empty group builds and orders nothing.
//
// # Layers
//
// The layers are the longest-path layering of the discovered Use edges
// and the declared After edges, as [System.Layers] states: every node's dependencies sit in lower
// layers, so the nodes of one layer can start together once the layers
// below them have.
//
// # Introspection
//
// What a Build reports names a node by its handle, the *Node[T] that
// [Graph.Define], [Graph.Value], or [Fresh] returned, so a test compares
// handles rather than names: [Dependency.Ref] is that handle, as is the
// [Ref] an observer added by [Graph.Observe] receives, and
// [System.LayerOf] takes one and returns the node's layer, numbered as
// Layers numbers them. [System.String] prints the layer table, a line such
// as "1: cache, db" per layer, so a failing test can show the whole
// layering.
//
// # Errors and panics
//
// A constructor's error fails the Build, labelled with its node's name and
// wrapping the error; a failing dependency aborts every constructor whose
// Use reached it, and Build reports the dependency's error.
//
// [Scope.Use] returns the dependency's value itself, so the abort unwinds
// by panic, which Build recovers. A constructor must not recover it: one
// that returns, with a value or an error, after a Use it made aborted would
// lose the failure, so Build panics naming the constructor and the Use.
//
// Wiring mistakes panic with a "graph: " message, as each symbol's
// documentation states: a dependency cycle, a node or group used on a
// Graph it was not defined on, a Scope used after its constructor returned,
// a recovered abort, a Define, Value, Fresh, Group, or Replace after Build, a Get or
// LayerOf of a node not in the System, a LayerOf of a group, which has no
// layer, a nil constructor, option, or observer, a nil
// ref given to After, In, or Build, and an empty or duplicate name, nodes
// and groups sharing one name space. Define checks its options, for a nil
// ref or one defined on another Graph, before it changes the Graph. A cycle
// closed by an order-only edge, through a group included, panics at Build
// with its path. Build checks its roots, for a nil ref or one defined on
// another Graph, before it constructs any of them, so a mistake in any root
// panics before a constructor runs.
package graph
