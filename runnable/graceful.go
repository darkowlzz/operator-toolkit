package runnable

import (
	"context"
	"sync"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("graceful-runnable")

// RunCall starts a component.
type RunCall func(context.Context) error

// StopCall stops a component.
type StopCall func() error

// Graceful implements the Runnable interface and handles graceful shutdown of
// the component. This is useful for components that don't stop immediately.
type Graceful struct {
	// Run starts the component. It's called in a goroutine and can be
	// blocking.
	run RunCall

	// Stop stops the running component.
	stop StopCall

	// RequireLeaderElection decides if the runnable requires leader election
	// before running.
	requireLeaderElection bool

	// wg is the WaitGroup for handling graceful stop.
	wg *sync.WaitGroup

	// log is the logger.
	log logr.Logger
}

// NewGraceful creates a new graceful shutdown runnable. logger is optional.
// Pass nil for the default logger.
func NewGraceful(run RunCall, stop StopCall, requireLeaderElection bool, wg *sync.WaitGroup, logger logr.Logger) *Graceful {
	// Use the default package logger if not provided.
	if logger == nil {
		logger = log
	}

	return &Graceful{
		run:                   run,
		stop:                  stop,
		requireLeaderElection: requireLeaderElection,
		wg:                    wg,
		log:                   logger,
	}
}

// Start implements the Runnable interface which enables the component to be
// managed by the controller manager.
func (g *Graceful) Start(ctx context.Context) error {
	// Handle stop in a goroutine.
	go func() {
		defer g.wg.Done()
		<-ctx.Done()
		g.log.Info("stopping gracefully")
		if err := g.stop(); err != nil {
			g.log.Error(err, "failed to stop gracefully")
		}
	}()

	return g.run(ctx)
}

// NeedLeaderElection implements the LeaderElectionRunnable interface, which
// helps the controller manager decide when to start the component.
func (g *Graceful) NeedLeaderElection() bool {
	return g.requireLeaderElection
}
