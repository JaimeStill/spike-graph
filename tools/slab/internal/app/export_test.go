package app

import "github.com/JaimeStill/spike-graph/sdk/cli/scenario"

// Scenarios returns the scenarios a mounts under the scenario parent, built
// over a's nodes as mountScenarios builds them, so a test enumerates the
// registered set rather than a copy of it.
func Scenarios(a *App) []scenario.Scenario { return scenariosOver(&a.nodes) }
