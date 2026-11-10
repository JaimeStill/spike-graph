package app_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/JaimeStill/spike-graph/sdk/cli"
	"github.com/JaimeStill/spike-graph/sdk/graph"
	"github.com/JaimeStill/spike-graph/sdk/graph/graphtest"
	"github.com/JaimeStill/spike-graph/tools/slab/internal/app"
	"github.com/JaimeStill/spike-graph/tools/slab/scenarios"
)

// This file pins the graph's structural properties the composition root
// owns: each Nodes field names its node, and each scenario's System orders
// its need nodes so a failed start names the first thing to fix. It builds
// the nodes without starting them: no constructor dials, since the service
// and grafana up nodes only bind clients, which the lifecycle's Start
// probes, so it needs nothing running. A failing layer assertion prints the
// System's layer table.

// Each node's name is its Nodes field's name lowercased, so a field names
// the node it holds without a comment repeating the name, and every field
// holds a handle once New returns. "grafana up" passes as GrafanaUp: the
// check ignores the name's spaces.
func TestGraph_NodesNamedByTheirFields(t *testing.T) {
	a := app.New(cli.Streams{})
	graphtest.CheckNames(t, a.Nodes())
}

// scenarioLayers reports, as an error carrying the layer table, what is
// wrong with sys, a scenario's System: an After edge the Build dropped,
// since then an order the root declared is silently lost, or grafana up at
// or below service's layer when both are in sys, since the lifecycle stops
// at the first layer with a failure and, with both down, would report
// Grafana's and never probe the service.
func scenarioLayers(sys *graph.System, n app.Nodes) error {
	var drops []string
	for _, d := range sys.Dropped() {
		drops = append(drops, fmt.Sprintf("%s's After of %s", d.Node.Name(), d.After.Name()))
	}
	if len(drops) > 0 {
		return layerError(sys, "dropped %s; want every After to order", strings.Join(drops, ", "))
	}
	refs := built(sys)
	if !slices.Contains(refs, graph.Ref(n.Service)) || !slices.Contains(refs, graph.Ref(n.GrafanaUp)) {
		return nil
	}
	if g, s := sys.LayerOf(n.GrafanaUp), sys.LayerOf(n.Service); g <= s {
		return layerError(sys, "%s in layer %d, %s in layer %d; want %s above %s",
			n.GrafanaUp.Name(), g, n.Service.Name(), s, n.GrafanaUp.Name(), n.Service.Name())
	}
	return nil
}

// built returns the handle of each node in sys, layer 0 first.
func built(sys *graph.System) []graph.Ref {
	var refs []graph.Ref
	for _, d := range slices.Concat(sys.Layers()...) {
		refs = append(refs, d.Ref)
	}
	return refs
}

// layerError returns an error of the formatted message followed by sys's
// layer table.
func layerError(sys *graph.System, format string, args ...any) error {
	return fmt.Errorf(format+"\nlayers:\n%s", append(args, sys)...)
}

// Every scenario slab registers builds, from the nodes it declares and the
// lifecycle config node, the set the dispatcher builds for its leaf, a
// System that drops no order and puts grafana up above service whenever it
// holds both. The scenarios are scenarios.All over the App's nodes, the set
// the composition root mounts, not a list copied here, so a scenario added
// later is checked too.
func TestGraph_ScenarioLayers(t *testing.T) {
	a := app.New(cli.Streams{})
	n := a.Nodes()
	all := scenarios.All(scenarios.Nodes{
		Service:   n.Service,
		GrafanaUp: n.GrafanaUp,
		Repo:      n.Repo,
		NoColor:   n.NoColor,
	})
	for _, s := range all {
		t.Run(s.Name, func(t *testing.T) {
			sys, err := a.Graph().Build(append(slices.Clone(s.Nodes), n.LifecycleConfig)...)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if err := scenarioLayers(sys, n); err != nil {
				t.Error(err)
			}
		})
	}
}

// The layer guard fails a System it should, naming what is wrong and
// carrying the layer table, so the test above can fail: grafana up beside
// service, as a graph whose grafana up lacks its After edge builds it, and
// an After edge the Build dropped, as a scenario that declared grafana up
// without service would build it.
func TestGraph_ScenarioLayersGuardFails(t *testing.T) {
	for _, c := range []struct {
		name    string
		after   bool // whether grafana up is ordered after service
		roots   func(app.Nodes) []graph.Ref
		mention string
	}{
		{"grafana up beside service", false, func(n app.Nodes) []graph.Ref { return []graph.Ref{n.Service, n.GrafanaUp} }, "want grafana up above service"},
		{"dropped order", true, func(n app.Nodes) []graph.Ref { return []graph.Ref{n.GrafanaUp} }, "dropped"},
	} {
		t.Run(c.name, func(t *testing.T) {
			g := graph.New()
			var n app.Nodes
			n.Service = g.Value("service", scenarios.Service{})
			var opts []graph.Option
			if c.after {
				opts = append(opts, graph.After(n.Service))
			}
			n.GrafanaUp = g.Value("grafana up", scenarios.Grafana{}, opts...)
			sys, err := g.Build(c.roots(n)...)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			err = scenarioLayers(sys, n)
			switch {
			case err == nil:
				t.Fatalf("the guard passed a fixture that breaks it\nlayers:\n%s", sys)
			case !strings.Contains(err.Error(), c.mention):
				t.Errorf("guard error = %q, want it to mention %q", err, c.mention)
			case !strings.Contains(err.Error(), "\nlayers:\n"+sys.String()):
				t.Errorf("guard error = %q, want it to carry the layer table:\n%s", err, sys)
			}
		})
	}
}
