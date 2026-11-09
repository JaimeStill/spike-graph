package graph

import "fmt"

// Group is a named set of nodes, defined by [Graph.Group]: a [Ref] that
// stands for its members. A node joins it with [In] at Define. A Build
// given a group, as a root or as an [After] target, expands it to its
// members in definition order; since Define panics after the first Build,
// the membership is complete by then. An empty group builds and orders
// nothing.
type Group struct {
	graph *Graph
	name  string
	// members are the nodes defined with In of the group, in definition
	// order.
	members []*node
}

// Group defines a group named name on g and returns its handle. Groups
// share the name space with nodes: name may not be taken by either. Group
// panics once g has run a Build, or when name is empty or already defined
// on g.
func (g *Graph) Group(name string) *Group {
	if g.built {
		panic(fmt.Sprintf("graph: Group of %q after Build", name))
	}
	if name == "" {
		panic("graph: Group with an empty name")
	}
	if g.defined(name) {
		panic(fmt.Sprintf("graph: Group of duplicate name %q", name))
	}
	grp := &Group{graph: g, name: name}
	g.groups = append(g.groups, grp)
	return grp
}

// Name returns the name the group was defined with.
func (grp *Group) Name() string { return grp.name }

func (grp *Group) expand() (*Graph, []*node) {
	if grp == nil {
		return nil, nil
	}
	return grp.graph, grp.members
}

func (*Group) kind() string { return "group" }
