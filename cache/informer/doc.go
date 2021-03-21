// Package informer is based on internal package
// https://github.com/kubernetes-sigs/controller-runtime/tree/v0.8.3/pkg/cache/internal,
// modified to not be kubernetes API specific. The InformersMap includes a
// CreateListWatcherFunc that can be used to pass ListWatcherFunc for any API,
// outside of kubernetes. The cache.ListWatch from the list watcher func is
// used by the informer to list and watch, and update the cache with objects
// from an API server.
// It should be possible to use this cache to watch k8s API as well, but for
// that, the controller-runtime cache package is more suitable.
package informer
