package lifecycle

import (
	core "github.com/standards-lab/go-core/lifecycle"
)

// The declarations below are go-core v0.10.0's lifecycle, aliased rather
// than forked: they are not tied to the [Coordinator], so a value written
// against go-core's interfaces, such as a published go-web-sdk Server,
// takes part in this package's Coordinator unchanged, and a [Check] this
// package returns is go-core's. Their documentation is go-core's; it is
// restated here as this package reads it.

// ReadinessChecker reports whether a subsystem is ready to serve.
// [Coordinator] satisfies it, as does any subsystem a readiness probe
// aggregates. It is go-core's.
type ReadinessChecker = core.ReadinessChecker

// Check pairs a readiness check with the name a probe reports for it; a
// probe treats a nil Checker as not ready. It is go-core's.
type Check = core.Check

// Starter is a value that starts. A [graph.Dependency] whose Value
// implements it takes part in a [Coordinator]'s startup. It is go-core's.
type Starter = core.Starter

// Stopper is a value that shuts down. A [graph.Dependency] whose Value
// implements it takes part in a [Coordinator]'s shutdown. It is go-core's.
type Stopper = core.Stopper

// Subsystem is a value with a whole lifecycle, one that starts and shuts
// down: a [graph.Dependency] whose Value implements it takes part in both
// phases. It is go-core's.
type Subsystem = core.Subsystem

// ReadyStarter is a [Starter] that reports whether what it started is
// ready: the value [Coordinator.Checks] infers a readiness check from. A
// value with Ready and no Start is not one. It is go-core's.
type ReadyStarter = core.ReadyStarter

// Monitored is a value that reports runtime failures on a channel. A
// [graph.Dependency] whose Value implements it is watched by the
// [Coordinator] that [New] returns for its System, as a channel registered
// with [Coordinator.Monitor] is: the first non-nil error ends the run. Err
// is called once startup has completed, so a value may create its channel
// in Start. A nil error is ignored, a closed channel retires quietly, and a
// nil channel, which never yields, is not watched. It is go-core's.
type Monitored = core.Monitored

// Config is a [Coordinator]'s configuration, with Merge and Finalize. Env
// records the environment-variable names Finalize composed and read; it is
// excluded from JSON. It is go-core's, methods and all.
type Config = core.Config

// Env names the environment variable [Config.Finalize] reads. An empty name
// disables the override. It is go-core's.
type Env = core.Env

// NewEnv composes <PREFIX>_SHUTDOWN_TIMEOUT with go-core's config.EnvName;
// an empty prefix returns the zero Env. It forwards to go-core's.
func NewEnv(prefix string) Env {
	return core.NewEnv(prefix)
}
