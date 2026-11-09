package graph_test

import (
	"slices"
	"testing"

	"github.com/JaimeStill/spike-graph/sdk/graph"
)

// refNames returns each ref's name, for a failure message.
func refNames(refs []graph.Ref) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Name()
	}
	return out
}

// layered defines a graph that, built from server, clock, and tier, has
// the layers 0: cfg, clock, seed, gate; 1: cache, db; 2: server, each in
// definition order. gate is a Fresh node the server Uses, seed a Value node
// the db Uses, and clock a Value node built only as a root; the server
// orders After tier, whose members are cache and db. lonely is never
// reached. It returns the graph and its handles.
func layered() (g *graph.Graph, h struct {
	cfg, db, cache, server, lonely *graph.Node[int]
	clock                          *graph.Node[string]
	gate                           *graph.Node[*bool]
	seed                           *graph.Node[int]
	tier                           *graph.Group
}) {
	g = graph.New()
	h.tier = g.Group("tier")
	h.cfg = g.Define("cfg", func(*graph.Scope) (int, error) { return 1, nil })
	h.clock = g.Value("clock", "tick")
	h.seed = g.Value("seed", 7)
	h.gate = graph.Fresh[bool](g, "gate")
	h.lonely = g.Define("lonely", func(*graph.Scope) (int, error) { return 0, nil })
	h.cache = g.Define("cache", func(s *graph.Scope) (int, error) { return s.Use(h.cfg), nil }, graph.In(h.tier))
	h.db = g.Define("db", func(s *graph.Scope) (int, error) { return s.Use(h.cfg) + s.Use(h.seed), nil }, graph.In(h.tier))
	h.server = g.Define("server", func(s *graph.Scope) (int, error) {
		_ = s.Use(h.gate)
		return s.Use(h.db), nil
	}, graph.After(h.tier))
	return g, h
}

func TestDependency_RefIsTheHandle(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server, h.clock, h.tier)

	want := map[string]graph.Ref{
		"cfg": h.cfg, "clock": h.clock, "seed": h.seed, "gate": h.gate,
		"cache": h.cache, "db": h.db, "server": h.server,
	}
	seen := 0
	for _, layer := range sys.Layers() {
		for _, d := range layer {
			seen++
			ref, ok := want[d.Name]
			if !ok {
				t.Fatalf("unexpected node %q in the System", d.Name)
			}
			// Interface equality: the same *Node[T] Define, Value, or Fresh
			// returned, not a copy carrying the same node.
			if d.Ref != ref {
				t.Errorf("%s: Ref = %#v, want the handle %#v", d.Name, d.Ref, ref)
			}
			if d.Ref != nil && d.Ref.Name() != d.Name {
				t.Errorf("%s: Ref.Name() = %q", d.Name, d.Ref.Name())
			}
		}
	}
	if seen != len(want) {
		t.Fatalf("saw %d nodes, want %d", seen, len(want))
	}
	// The Ref asserts back to the typed handle, which Get takes.
	for _, layer := range sys.Layers() {
		for _, d := range layer {
			if n, ok := d.Ref.(*graph.Node[*bool]); ok {
				if n != h.gate || sys.Get(n) != d.Value {
					t.Errorf("gate's Ref does not reach its value through Get")
				}
			}
		}
	}
}

func TestObserve_PassesValueAndFreshHandles(t *testing.T) {
	g, h := layered()
	var seen []graph.Ref
	g.Observe(func(r graph.Ref) { seen = append(seen, r) })
	mustBuild(t, g, h.server)
	want := []graph.Ref{h.server, h.gate, h.db, h.cfg, h.seed}
	if !slices.Equal(seen, want) {
		t.Fatalf("observed %v, want %v", refNames(seen), refNames(want))
	}
}

func TestLayerOf_MatchesLayers(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server, h.clock, h.tier)

	for i, layer := range sys.Layers() {
		for _, d := range layer {
			if got := sys.LayerOf(d.Ref); got != i {
				t.Errorf("LayerOf(%s) = %d, want %d, its place in Layers; layers:\n%s", d.Name, got, i, sys)
			}
		}
	}
	// Pinned, so a numbering that agreed with a wrong Layers still fails.
	for ref, want := range map[graph.Ref]int{h.cfg: 0, h.clock: 0, h.seed: 0, h.gate: 0, h.cache: 1, h.db: 1, h.server: 2} {
		if got := sys.LayerOf(ref); got != want {
			t.Errorf("LayerOf(%s) = %d, want %d; layers:\n%s", ref.Name(), got, want, sys)
		}
	}
}

func TestLayerOf_Panics(t *testing.T) {
	g, h := layered()
	sys := mustBuild(t, g, h.server)

	t.Run("unreached node", func(t *testing.T) {
		mustPanic(t, `LayerOf of "lonely", a node not in the System`, func() { sys.LayerOf(h.lonely) })
	})
	t.Run("node not built in this System", func(t *testing.T) {
		mustPanic(t, `LayerOf of "clock", a node not in the System`, func() { sys.LayerOf(h.clock) })
	})
	t.Run("node of another Graph", func(t *testing.T) {
		other := graph.New()
		cfg := other.Define("cfg", func(*graph.Scope) (int, error) { return 0, nil })
		mustBuild(t, other, cfg)
		mustPanic(t, `LayerOf of "cfg", a node not in the System`, func() { sys.LayerOf(cfg) })
	})
	t.Run("group", func(t *testing.T) {
		mustPanic(t, `LayerOf of "tier", a group, which has no layer`, func() { sys.LayerOf(h.tier) })
	})
	t.Run("nil ref", func(t *testing.T) {
		mustPanic(t, "LayerOf of a nil node", func() { sys.LayerOf(nil) })
	})
	t.Run("nil node handle", func(t *testing.T) {
		var n *graph.Node[int]
		mustPanic(t, "LayerOf of a nil node", func() { sys.LayerOf(n) })
	})
	t.Run("nil group handle", func(t *testing.T) {
		var grp *graph.Group
		mustPanic(t, "LayerOf of a nil group", func() { sys.LayerOf(grp) })
	})
}

func TestString_PrintsTheLayerTable(t *testing.T) {
	g, h := layered()
	// The roots list the server first, so a table in build order would put
	// gate before cfg; the table is in definition order within a layer.
	sys := mustBuild(t, g, h.server, h.clock, h.tier)
	want := "0: cfg, clock, seed, gate\n1: cache, db\n2: server"
	if got := sys.String(); got != want {
		t.Fatalf("String() =\n%s\nwant\n%s", got, want)
	}

	empty := mustBuild(t, graph.New())
	if got := empty.String(); got != "" {
		t.Fatalf("String() of an empty System = %q, want empty", got)
	}
}
