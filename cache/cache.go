package cache

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/darkowlzz/operator-toolkit/cache/informer"
)

// Options are the optional arguments for creating a new InformersMap object.
type Options struct {
	// Scheme is the scheme to use for mapping objects to GroupVersionKinds
	Scheme *runtime.Scheme

	// Resync is the base frequency the informers are resynced.
	// Defaults to defaultResyncTime.
	// A 10 percent jitter will be added to the Resync period between informers
	// So that all informers will not send list requests simultaneously.
	Resync *time.Duration

	// Namespace restricts the cache's ListWatch to the desired namespace
	// Default watches all namespaces
	Namespace string
}

var defaultResyncTime = 10 * time.Hour

// New initializes and returns a new Cache.
func New(createLWFunc informer.CreateListWatcherFunc, opts Options) cache.Cache {
	opts = defaultOpts(opts)
	im := informer.NewInformersMap(opts.Scheme, *opts.Resync, opts.Namespace, createLWFunc)
	return &informerCache{InformersMap: im}
}

func defaultOpts(opts Options) Options {
	// Use the default Kubernetes Scheme if unset
	if opts.Scheme == nil {
		opts.Scheme = scheme.Scheme
	}

	// Default the resync period to 10 hours if unset
	if opts.Resync == nil {
		opts.Resync = &defaultResyncTime
	}
	return opts
}
