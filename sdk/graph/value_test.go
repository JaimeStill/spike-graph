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
	n := graph.Fresh[gate](g, "gate")
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
	ready := graph.Fresh[gate](g, "readiness")
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
	ready := graph.Fresh[gate](g, "readiness")
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
	ready := graph.Fresh[gate](g, "readiness")
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
		mustPanic(t, "Fresh with an empty name", func() { graph.Fresh[gate](graph.New(), "") })
	})
	t.Run("value of a defined name", func(t *testing.T) {
		g := graph.New()
		g.Define("db", zero)
		mustPanic(t, `Value of duplicate name "db"`, func() { g.Value("db", 1) })
	})
	t.Run("fresh of a group's name", func(t *testing.T) {
		g := graph.New()
		g.Group("db")
		mustPanic(t, `Fresh of duplicate name "db"`, func() { graph.Fresh[gate](g, "db") })
	})
	t.Run("define of a value's name", func(t *testing.T) {
		g := graph.New()
		g.Value("db", 1)
		mustPanic(t, `Define of duplicate name "db"`, func() { g.Define("db", zero) })
	})
	t.Run("define of a fresh's name", func(t *testing.T) {
		g := graph.New()
		graph.Fresh[gate](g, "db")
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
		mustPanic(t, `Fresh of "late" after Build`, func() { graph.Fresh[gate](g, "late") })
	})
	t.Run("get of a value not in the system", func(t *testing.T) {
		g := graph.New()
		out := g.Value("out", 1)
		sys := mustBuild(t, g, g.Define("top", zero))
		mustPanic(t, `Get of "out", a node not in the System`, func() { sys.Get(out) })
	})
}
