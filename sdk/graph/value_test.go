package graph_test

import (
	"slices"
	"testing"

	"github.com/JaimeStill/spike-graph/sdk/graph"
)

// settings is a value a test hands to Value, and gate a type Fresh
// allocates.
type settings struct{ dsn string }

type gate struct{ open bool }

func TestValue_IsVInEveryBuild(t *testing.T) {
	g := graph.New()
	cfg := &settings{dsn: "postgres"}
	n := g.Value("config", cfg)
	if got := n.Name(); got != "config" {
		t.Fatalf("Name = %q, want config", got)
	}
	first := mustBuild(t, g, n)
	second := mustBuild(t, g, n)
	if first.Get(n) != cfg || second.Get(n) != cfg {
		t.Fatalf("Get = %p and %p, want %p in both Builds", first.Get(n), second.Get(n), cfg)
	}
	if got := dependency(t, first, "config").Value; got != cfg {
		t.Fatalf("config's Value = %v, want %v", got, cfg)
	}
}

func TestFresh_IsANewZeroPointerInEachBuild(t *testing.T) {
	g := graph.New()
	n := g.Fresh[gate]("gate")
	if got := n.Name(); got != "gate" {
		t.Fatalf("Name = %q, want gate", got)
	}
	first := mustBuild(t, g, n)
	a := first.Get(n)
	if a == nil || a.open {
		t.Fatalf("first Build's value = %+v, want a non-nil zero gate", a)
	}
	// Mutating one Build's value does not reach the next.
	a.open = true
	b := mustBuild(t, g, n).Get(n)
	if b == a {
		t.Fatalf("two Builds gave the same pointer %p", a)
	}
	if b == nil || b.open {
		t.Fatalf("second Build's value = %+v, want a non-nil zero gate", b)
	}
}

func TestValueAndFresh_AreUseTargets(t *testing.T) {
	g := graph.New()
	cfg := g.Value("config", settings{dsn: "postgres"})
	ready := g.Fresh[gate]("readiness")
	var seen *gate
	server := g.Define("server", func(s *graph.Scope) (string, error) {
		seen = s.Use(ready)
		return "serving " + s.Use(cfg).dsn, nil
	})

	sys := mustBuild(t, g, server)
	if got := sys.Get(server); got != "serving postgres" {
		t.Fatalf("server = %q, want serving postgres", got)
	}
	// The constructor received the System's own pointer, not another.
	if seen == nil || seen != sys.Get(ready) {
		t.Fatalf("Use gave %p, Get gives %p; want the same non-nil pointer", seen, sys.Get(ready))
	}
	want := [][]string{{"config", "readiness"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestValueAndFresh_AreAfterTargets(t *testing.T) {
	c := counter{}
	g := graph.New()
	cfg := g.Value("config", 1)
	ready := g.Fresh[gate]("readiness")
	server := g.Define("server", constant(c, "server", 0), graph.After(cfg, ready))

	// Named as roots, both order the server above them.
	sys := mustBuild(t, g, server, cfg, ready)
	want := [][]string{{"config", "readiness"}, {"server"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}

	// After builds neither.
	sys = mustBuild(t, g, server)
	if contains(sys, "config") || contains(sys, "readiness") {
		t.Fatalf("After pulled a value in: layers %v", names(sys))
	}
}

func TestValueAndFresh_CanBeReplaced(t *testing.T) {
	g := graph.New()
	cfg := g.Value("config", "real")
	ready := g.Fresh[gate]("readiness")
	open := &gate{open: true}
	g.Replace(cfg, func(*graph.Scope) (string, error) { return "fake", nil })
	g.Replace(ready, func(*graph.Scope) (*gate, error) { return open, nil })

	sys := mustBuild(t, g, cfg, ready)
	if got := sys.Get(cfg); got != "fake" {
		t.Fatalf("config = %q, want fake", got)
	}
	if got := sys.Get(ready); got != open {
		t.Fatalf("readiness = %p, want the substitute's %p", got, open)
	}
}

func TestValueAndFresh_Panics(t *testing.T) {
	t.Run("value with an empty name", func(t *testing.T) {
		mustPanic(t, "Value with an empty name", func() { graph.New().Value("", 1) })
	})
	t.Run("fresh with an empty name", func(t *testing.T) {
		mustPanic(t, "Fresh with an empty name", func() { graph.New().Fresh[gate]("") })
	})
	t.Run("value of a defined name", func(t *testing.T) {
		g := graph.New()
		g.Define("db", zero)
		mustPanic(t, `Value of duplicate name "db"`, func() { g.Value("db", 1) })
	})
	t.Run("fresh of a group's name", func(t *testing.T) {
		g := graph.New()
		g.Group("db")
		mustPanic(t, `Fresh of duplicate name "db"`, func() { g.Fresh[gate]("db") })
	})
	t.Run("define of a value's name", func(t *testing.T) {
		g := graph.New()
		g.Value("db", 1)
		mustPanic(t, `Define of duplicate name "db"`, func() { g.Define("db", zero) })
	})
	t.Run("define of a fresh's name", func(t *testing.T) {
		g := graph.New()
		g.Fresh[gate]("db")
		mustPanic(t, `Define of duplicate name "db"`, func() { g.Define("db", zero) })
	})
	t.Run("value after build", func(t *testing.T) {
		g := graph.New()
		mustBuild(t, g)
		mustPanic(t, `Value of "late" after Build`, func() { g.Value("late", 1) })
	})
	t.Run("fresh after build", func(t *testing.T) {
		g := graph.New()
		mustBuild(t, g)
		mustPanic(t, `Fresh of "late" after Build`, func() { g.Fresh[gate]("late") })
	})
	t.Run("get of a value not in the system", func(t *testing.T) {
		g := graph.New()
		out := g.Value("out", 1)
		sys := mustBuild(t, g, g.Define("top", zero))
		mustPanic(t, `Get of "out", a node not in the System`, func() { sys.Get(out) })
	})
}

func TestValueAndFresh_TakeInAndAfter(t *testing.T) {
	c := counter{}
	g := graph.New()
	workers := g.Group("workers")
	base := g.Define("base", constant(c, "base", 0))
	cfg := g.Value("config", 1, graph.In(workers), graph.After(base))
	ready := g.Fresh[gate]("readiness", graph.In(workers), graph.After(base))
	top := g.Define("top", constant(c, "top", 0), graph.After(workers))

	// The group as a root builds both members, and After orders each above
	// base and below the node ordered after the group.
	sys := mustBuild(t, g, top, workers, base)
	want := [][]string{{"base"}, {"config", "readiness"}, {"top"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
	if sys.Get(cfg) != 1 || sys.Get(ready) == nil {
		t.Fatalf("config = %v, readiness = %v; want 1 and a non-nil gate", sys.Get(cfg), sys.Get(ready))
	}

	// After builds nothing: the group alone brings up only its members.
	sys = mustBuild(t, g, workers)
	want = [][]string{{"config", "readiness"}}
	if got := names(sys); !slices.EqualFunc(got, want, slices.Equal) {
		t.Fatalf("layers = %v, want %v", got, want)
	}
}

func TestValueAndFresh_OptionPanics(t *testing.T) {
	// Each case runs once through Value and once through Fresh; the message
	// is Define's, naming the call.
	calls := []struct {
		call   string
		define func(g *graph.Graph, opts ...graph.Option)
	}{
		{"Value", func(g *graph.Graph, opts ...graph.Option) { g.Value("top", 1, opts...) }},
		{"Fresh", func(g *graph.Graph, opts ...graph.Option) { g.Fresh[gate]("top", opts...) }},
	}
	for _, c := range calls {
		t.Run(c.call, func(t *testing.T) {
			t.Run("after of a nil ref", func(t *testing.T) {
				mustPanic(t, c.call+` "top" with After of a nil node`, func() {
					c.define(graph.New(), graph.After(nil))
				})
			})
			t.Run("after of a foreign node", func(t *testing.T) {
				g, h := graph.New(), graph.New()
				foreign := h.Define("foreign", zero)
				mustPanic(t, c.call+` "top" with After of "foreign", a node defined on another Graph`, func() {
					c.define(g, graph.After(foreign))
				})
			})
			t.Run("in a nil group", func(t *testing.T) {
				mustPanic(t, c.call+` "top" with In of a nil group`, func() {
					c.define(graph.New(), graph.In(nil))
				})
			})
			t.Run("in a foreign group", func(t *testing.T) {
				g, h := graph.New(), graph.New()
				foreign := h.Group("workers")
				mustPanic(t, c.call+` "top" with In of "workers", a group defined on another Graph`, func() {
					c.define(g, graph.In(foreign))
				})
			})
			t.Run("nil option", func(t *testing.T) {
				mustPanic(t, c.call+` "top" with a nil option`, func() {
					c.define(graph.New(), nil)
				})
			})
			t.Run("a panicking call changes nothing", func(t *testing.T) {
				g := graph.New()
				workers := g.Group("workers")
				mustPanic(t, c.call+` "top" with After of a nil node`, func() {
					c.define(g, graph.In(workers), graph.After(nil))
				})
				// The name is still free, and the group gained no member.
				top := g.Define("top", zero)
				if sys := mustBuild(t, g, workers); len(sys.Layers()) != 0 {
					t.Fatalf("the failed %s joined the group: layers %v", c.call, names(sys))
				}
				mustBuild(t, g, top)
			})
		})
	}
}
