// NOTE: Based on
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.3/pkg/source/source.go,
// modified to enable running the source when the Kind is not registered with
// k8s API server. This is needed for the cache of external system objects.
package source

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/darkowlzz/operator-toolkit/source/internal"
)

// NewKindWithCache creates a Source without InjectCache, so that it is assured that the given cache is used
// and not overwritten. It can be used to watch objects in a different cluster by passing the cache
// from that other cluster
func NewKindWithCache(object client.Object, cache cache.Cache) source.SyncingSource {
	return &kindWithCache{kind: Kind{Type: object, cache: cache}}
}

type kindWithCache struct {
	kind Kind
}

func (ks *kindWithCache) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {
	return ks.kind.Start(ctx, handler, queue, prct...)
}

func (ks *kindWithCache) WaitForSync(ctx context.Context) error {
	return ks.kind.WaitForSync(ctx)
}

// Kind is used to provide a source of events originating inside the cluster from Watches (e.g. Pod Create)
type Kind struct {
	// Type is the type of object to watch.  e.g. &v1.Pod{}
	Type client.Object

	// cache used to watch APIs
	cache cache.Cache
}

var _ source.SyncingSource = &Kind{}

// Start is internal and should be called only by the Controller to register an EventHandler with the Informer
// to enqueue reconcile.Requests.
func (ks *Kind) Start(ctx context.Context, handler handler.EventHandler, queue workqueue.RateLimitingInterface,
	prct ...predicate.Predicate) error {

	// Type should have been specified by the user.
	if ks.Type == nil {
		return fmt.Errorf("must specify Kind.Type")
	}

	// cache should have been injected before Start was called
	if ks.cache == nil {
		return fmt.Errorf("must call CacheInto on Kind before calling Start")
	}

	// Lookup the Informer from the Cache and add an EventHandler which populates the Queue
	i, err := ks.cache.GetInformer(ctx, ks.Type)
	if err != nil {
		// NOTE: This checked is removed to enable running the source without
		// registering the Kinds with the k8s API server.
		// if kindMatchErr, ok := err.(*meta.NoKindMatchError); ok {
		//     log.Error(err, "if kind is a CRD, it should be installed before calling Start",
		//         "kind", kindMatchErr.GroupKind)
		// }
		return err
	}
	i.AddEventHandler(internal.EventHandler{Queue: queue, EventHandler: handler, Predicates: prct})
	return nil
}

func (ks *Kind) String() string {
	if ks.Type != nil && ks.Type.GetObjectKind() != nil {
		return fmt.Sprintf("kind source: %v", ks.Type.GetObjectKind().GroupVersionKind().String())
	}
	return "kind source: unknown GVK"
}

// WaitForSync implements SyncingSource to allow controllers to wait with starting
// workers until the cache is synced.
func (ks *Kind) WaitForSync(ctx context.Context) error {
	if !ks.cache.WaitForCacheSync(ctx) {
		// Would be great to return something more informative here
		return errors.New("cache did not sync")
	}
	return nil
}

var _ inject.Cache = &Kind{}

// InjectCache is internal should be called only by the Controller.  InjectCache is used to inject
// the Cache dependency initialized by the ControllerManager.
func (ks *Kind) InjectCache(c cache.Cache) error {
	if ks.cache == nil {
		ks.cache = c
	}
	return nil
}
