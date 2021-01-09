package v1

import "time"

// SyncFunc defines a sync function with a sync period.
type SyncFunc struct {
	f      func()
	period time.Duration
}

// NewSyncFunc returns a new SyncFunc, given a function and a sync period.
func NewSyncFunc(f func(), p time.Duration) SyncFunc {
	return SyncFunc{
		f:      f,
		period: p,
	}
}

// Run runs the SyncFunc function at the SyncFunc period.
func (sf SyncFunc) Run() {
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
