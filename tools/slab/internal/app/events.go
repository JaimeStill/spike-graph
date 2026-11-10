package app

import (
	"github.com/JaimeStill/spike-graph/sdk/cli"
	"github.com/JaimeStill/spike-graph/tools/slab/events"
)

// mountEvents mounts the events package's commands at root, handed the
// broker node from n and the output source to render through. The events
// command watches the stream the service publishes to rather than calling
// an endpoint, so it reads the broker where the domain and admin families
// read the client.
func mountEvents(root *cli.Command, n *Nodes) {
	root.Add(events.Commands(n.Broker, outputSource(n))...)
}
