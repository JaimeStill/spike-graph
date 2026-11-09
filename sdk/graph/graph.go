package graph

import (
	"fmt"
	"slices"
)

// Graph describes a set of nodes and their constructors. Describing is
// inert: [Graph.Define] and [Graph.Replace] record constructors and run
// none, and only [Graph.Build] constructs anything. The first Build freezes
// the description: Define and Replace panic after it. A Graph is not safe
// for concurrent use.
type Graph struct {
	nodes []*node
	// groups are the groups [Graph.Group] defined, in definition order. They
	// share the name space with nodes.
	groups []*Group
	// built is set by the first Build, which freezes the description:
	// Define and Replace panic after it.
	built bool
	// observers are called, in the order added, with a node's handle as a
	// Build begins the node.
	observers []func(Ref)
}

// New returns an empty Graph.
func New() *Graph {
	return &Graph{}
}

// erase wraps a typed constructor to return its value as any.
func erase[T any](ctor func(*Scope) (T, error)) func(*Scope) (any, error) {
	return func(s *Scope) (any, error) {
		v, err := ctor(s)
		return v, err
	}
}

// Define adds a node named name, constructed by ctor, and returns its
// handle. It runs nothing: ctor runs only when a [Graph.Build] reaches the
// node. The options place the node: [After] orders it after other nodes or
// groups, and [In] makes it a member of a group. Define panics once g has
// run a Build, when name is empty or already defined on g as a node or a
// group, when ctor or an option is nil, or when an option names a nil ref
// or one defined on another Graph. It checks every option before it changes
// g, so a Define that panics leaves g as it was.
func (g *Graph) Define[T any](name string, ctor func(*Scope) (T, error), opts ...Option) *Node[T] {
	return define(g, "Define", name, ctor, opts)
}

// Value defines a node named name whose value is v in every Build, and
// returns its handle, a [Node] like any Define returns: a Use target, a
// root, an [After] target, and replaceable. Every Build's value is v
// itself, so a pointer or map v is shared across Builds. Value panics as
// [Graph.Define] does: once g has run a Build, or when name is empty or
// already defined on g, the message naming Value.
func (g *Graph) Value[T any](name string, v T) *Node[T] {
	return define(g, "Value", name, func(*Scope) (T, error) { return v, nil }, nil)
}

// Fresh defines a node named name on g whose value is a new zero *T in
// each Build, and returns its handle, a [Node] like any Define returns. Two
// Builds give two distinct pointers, so a value a run mutates, a readiness
// flag or a gate, starts zero in every System. Fresh panics as
// [Graph.Define] does: once g has run a Build, or when name is empty or
// already defined on g, the message naming Fresh.
func Fresh[T any](g *Graph, name string) *Node[*T] {
	return define(g, "Fresh", name, func(*Scope) (*T, error) { return new(T), nil }, nil)
}

// define is Define, Value, and Fresh: it checks the name, the constructor,
// and every option before it changes g, then adds the node. call names the
// exported function for a panic's message.
func define[T any](g *Graph, call, name string, ctor func(*Scope) (T, error), opts []Option) *Node[T] {
	if g.built {
		panic(fmt.Sprintf("graph: %s of %q after Build", call, name))
	}
	if name == "" {
		panic(fmt.Sprintf("graph: %s with an empty name", call))
	}
	if g.defined(name) {
		panic(fmt.Sprintf("graph: %s of duplicate name %q", call, name))
	}
	if ctor == nil {
		panic(fmt.Sprintf("graph: %s %q with a nil constructor", call, name))
	}
	var set settings
	for _, opt := range opts {
		if opt == nil {
			panic(fmt.Sprintf("graph: %s %q with a nil option", call, name))
		}
		opt.apply(&set)
	}
	op := fmt.Sprintf("%s %q with ", call, name)
	for _, r := range set.after {
		g.resolve(r, op+"After")
	}
	for _, grp := range set.in {
		g.resolve(grp, op+"In")
	}

	core := &node{graph: g, name: name, index: len(g.nodes), ctor: erase(ctor), after: set.after}
	g.nodes = append(g.nodes, core)
	for _, grp := range set.in {
		if !slices.Contains(grp.members, core) {
			grp.members = append(grp.members, core)
		}
	}
	n := &Node[T]{core: core}
	core.handle = n
	return n
}

// defined reports whether name is taken on g, by a node or a group.
func (g *Graph) defined(name string) bool {
	return slices.ContainsFunc(g.nodes, func(n *node) bool { return n.name == name }) ||
		slices.ContainsFunc(g.groups, func(grp *Group) bool { return grp.name == name })
}

// resolve returns the nodes r stands for on g: a node itself, or a group's
// members in definition order. It panics when r is nil or defined on
// another Graph; op names the call for the message.
func (g *Graph) resolve(r Ref, op string) []*node {
	if r == nil {
		panic(fmt.Sprintf("graph: %s of a nil node", op))
	}
	owner, nodes := r.expand()
	if owner == nil {
		panic(fmt.Sprintf("graph: %s of a nil %s", op, r.kind()))
	}
	if owner != g {
		panic(fmt.Sprintf("graph: %s of %q, a %s defined on another Graph", op, r.Name(), r.kind()))
	}
	return nodes
}

// Replace swaps n's constructor for ctor, which every later Build runs in
// its place exactly as it would a defined one. It swaps only the
// constructor: n keeps the [After] and [In] it was defined with. Replace
// panics once g has run a Build, when n was defined on another Graph, or
// when ctor is nil.
func (g *Graph) Replace[T any](n *Node[T], ctor func(*Scope) (T, error)) {
	if g.built {
		panic(fmt.Sprintf("graph: Replace of %q after Build", n.core.name))
	}
	if n.core.graph != g {
		panic(fmt.Sprintf("graph: Replace of %q, a node defined on another Graph", n.core.name))
	}
	if ctor == nil {
		panic(fmt.Sprintf("graph: Replace of %q with a nil constructor", n.core.name))
	}
	n.core.ctor = erase(ctor)
}

// Observe adds fn, which every later Build calls with a node's handle as it
// begins to construct the node, before the constructor runs: each node a
// Build reaches, once, in the order it reaches them depth-first, the one
// whose constructor fails included. The handle is the one [Graph.Define],
// [Graph.Value], or [Fresh] returned, so fn compares it with a held handle
// rather than a name. It is for tracing what a Build constructs, such as a
// test asserting which nodes a run brought up; fn cannot change the Build.
// Observe panics on a nil fn.
func (g *Graph) Observe(fn func(Ref)) {
	if fn == nil {
		panic("graph: Observe with a nil function")
	}
	g.observers = append(g.observers, fn)
}
