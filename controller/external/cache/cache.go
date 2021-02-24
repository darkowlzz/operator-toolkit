package cache

import "sigs.k8s.io/controller-runtime/pkg/client"

// Cache is an event cache. It is used by the event handler to decide if an
// object should be reconciled or ignored.
type Cache interface {
	// CacheMiss takes an object and checks it with the cache. It returns true
	// when the object is not available in the cache or the cached item needs
	// to be updated. It returns false when there's no change in the cached
	// object.
	CacheMiss(client.Object) bool
}
