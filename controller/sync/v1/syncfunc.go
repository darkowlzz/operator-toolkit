package v1

import (
	"time"
)

const (
	// defaultStartupSyncDelay is the default delay period before starting the
	// sync ticker.
	defaultStartupSyncDelay time.Duration = 10 * time.Second

	zeroDuration time.Duration = 0 * time.Minute
)

// SyncFunc defines a sync function with a sync period.
type SyncFunc struct {
	f                func()
	period           time.Duration
	startupSyncDelay time.Duration
}

// NewSyncFunc returns a new SyncFunc, given a function and a sync period.
func NewSyncFunc(f func(), p time.Duration, d time.Duration) SyncFunc {
	// NOTE: This is not allowed to be set to zero to avoid running the sync
	// before the controller has been fully initialized. It results in errors
	// like: "the cache is not started, can not read objects".
	if d == zeroDuration {
		d = defaultStartupSyncDelay
	}

	return SyncFunc{
		f:                f,
		period:           p,
		startupSyncDelay: d,
	}
}

// Run runs the SyncFunc function at the SyncFunc period.
func (sf SyncFunc) Run() {
	// Wait before starting the sync func.
	time.Sleep(sf.startupSyncDelay)

	// Run the sync function before starting a ticker based run.
	sf.Call()

	// Start a ticker with the given period at which the sync function is
	// called.
	ticker := time.NewTicker(sf.period)
	defer ticker.Stop()

	for {
		<-ticker.C
		sf.Call()
	}
}

// Call calls the SyncFunc function.
func (sf SyncFunc) Call() {
	sf.f()
}
