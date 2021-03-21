package informer

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type CreateListWatcherFunc func(gvk schema.GroupVersionKind, namespace string, scheme *runtime.Scheme) (*cache.ListWatch, error)

// MapEntry contains the cached data for an Informer.
type MapEntry struct {
	// Informer is the cached informer
	Informer cache.SharedIndexInformer

	// CacheReader wraps Informer and implements the CacheReader interface for a single type
	Reader CacheReader
}

// InformersMap create and caches Informers for (runtime.Object, schema.GroupVersionKind) pairs.
type InformersMap struct {
	// structured InformersMap.
	informersByGVK map[schema.GroupVersionKind]*MapEntry

	// Scheme maps runtime.Objects to GroupVersionKinds.
	Scheme *runtime.Scheme

	// stop is the stop channel to stop informers
	stop <-chan struct{}

	// resync is the base frequency the informers are resynced
	// a 10 percent jitter will be added to the resync period between informers
	// so that all informers will not send list requests simultaneously.
	resync time.Duration

	// mu guards access to the map
	mu sync.RWMutex

	// start is true if the informers have been started
	started bool

	// startWait is a channel that is closed after the
	// informer has been started.
	startWait chan struct{}

	// createClient knows how to create a client and a list object,
	// and allows for abstracting over the particulars of structured vs
	// unstructured objects.
	createListWatcher CreateListWatcherFunc

	// namespace is the namespace that all ListWatches are restricted to
	// default or empty string means all namespaces
	namespace string
}

// NewInformersMap creates a new InformersMap that can create informers for
// objects.
func NewInformersMap(scheme *runtime.Scheme, resync time.Duration, namespace string, createLW CreateListWatcherFunc) *InformersMap {
	return &InformersMap{
		Scheme:            scheme,
		resync:            resync,
		namespace:         namespace,
		createListWatcher: createLW,
		informersByGVK:    make(map[schema.GroupVersionKind]*MapEntry),
		startWait:         make(chan struct{}),
	}
}

// Start calls Run on each of the informers and sets started to true.  Blocks
// on the context.
func (m *InformersMap) Start(ctx context.Context) error {
	go func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		// Set the stop channel so it can be passed to informers that are added later
		m.stop = ctx.Done()

		for _, informer := range m.informersByGVK {
			go informer.Informer.Run(ctx.Done())
		}

		// Set started to true so we immediately start any informers added later.
		m.started = true
		close(m.startWait)
	}()
	<-ctx.Done()
	return nil
}

// WaitForCacheSync waits until all the caches have been started and synced.
func (m *InformersMap) WaitForCacheSync(ctx context.Context) bool {
	syncedFuncs := append([]cache.InformerSynced(nil), m.HasSyncedFuncs()...)

	if !m.waitForStarted(ctx) {
		return false
	}

	return cache.WaitForCacheSync(ctx.Done(), syncedFuncs...)
}

// HasSyncedFuncs returns all the HasSynced functions for the informers in this
// map.
func (m *InformersMap) HasSyncedFuncs() []cache.InformerSynced {
	m.mu.Lock()
	defer m.mu.Unlock()

	syncedFuncs := make([]cache.InformerSynced, 0, len(m.informersByGVK))
	for _, informer := range m.informersByGVK {
		syncedFuncs = append(syncedFuncs, informer.Informer.HasSynced)
	}

	return syncedFuncs
}

func (m *InformersMap) waitForStarted(ctx context.Context) bool {
	select {
	case <-m.startWait:
		return true
	case <-ctx.Done():
		return false
	}
}

// Get will create a new Informer and add it to the map of informers if none
// exists.  Returns the Informer from the map.
func (m *InformersMap) Get(ctx context.Context, gvk schema.GroupVersionKind, obj runtime.Object) (bool, *MapEntry, error) {
	// Return the informer if it is found.
	i, started, ok := func() (*MapEntry, bool, bool) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		i, ok := m.informersByGVK[gvk]
		return i, m.started, ok
	}()

	if !ok {
		var err error
		if i, started, err = m.addInformerToMap(gvk, obj); err != nil {
			return started, i, err
		}
	}

	if started && !i.Informer.HasSynced() {
		if !cache.WaitForCacheSync(ctx.Done(), i.Informer.HasSynced) {
			return started, nil, apierrors.NewTimeoutError(fmt.Sprintf("failed waiting for %T Informer to sync", obj), 0)
		}
	}

	return started, i, nil
}

func (m *InformersMap) addInformerToMap(gvk schema.GroupVersionKind, obj runtime.Object) (*MapEntry, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check the cache to see if we already have an Informer.  If we do, return the Informer.
	// This is for the case where 2 routines tried to get the informer when it wasn't in the map
	// so neither returned early, but the first one created it.
	if i, ok := m.informersByGVK[gvk]; ok {
		return i, m.started, nil
	}

	// Create a NewSharedIndexInformer and add it to the map.
	// var lw *cache.ListWatch
	lw, err := m.createListWatcher(gvk, m.namespace, m.Scheme)
	if err != nil {
		return nil, false, err
	}
	ni := cache.NewSharedIndexInformer(lw, obj, resyncPeriod(m.resync)(), cache.Indexers{
		cache.NamespaceIndex: cache.MetaNamespaceIndexFunc,
	})

	// RESTScope based on the cache namespace.
	var scope apimeta.RESTScopeName
	if m.namespace == "" {
		scope = apimeta.RESTScopeNameRoot
	} else {
		scope = apimeta.RESTScopeNameNamespace
	}

	i := &MapEntry{
		Informer: ni,
		Reader:   CacheReader{indexer: ni.GetIndexer(), groupVersionKind: gvk, scopeName: scope},
	}
	m.informersByGVK[gvk] = i

	if m.started {
		go i.Informer.Run(m.stop)
	}
	return i, m.started, nil
}

// resyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func resyncPeriod(resync time.Duration) func() time.Duration {
	return func() time.Duration {
		// the factor will fall into [0.9, 1.1)
		factor := rand.Float64()/5.0 + 0.9
		return time.Duration(float64(resync.Nanoseconds()) * factor)
	}
}
