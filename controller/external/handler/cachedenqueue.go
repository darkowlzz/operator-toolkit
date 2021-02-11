package handler

import (
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/darkowlzz/operator-toolkit/controller/external/cache"
)

var log = ctrl.Log.WithName("eventhandler").WithName("EnqueueRequestFromCache")

var _ handler.EventHandler = &EnqueueRequestFromCache{}

// EnqueueRequestFromCache enqueues events based on a cache.
type EnqueueRequestFromCache struct {
	handler.Funcs
}

// NewEnqueueRequestFromCache takes a cache, creates an EnqueueRequestFromCache
// and adds a generic event handler that adds the event object in the queue on
// cache miss.
func NewEnqueueRequestFromCache(c cache.Cache) *EnqueueRequestFromCache {
	hdler := &EnqueueRequestFromCache{}
	hdler.GenericFunc = func(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
		if evt.Object == nil {
			log.Error(nil, "GenericEvent received with no metadata", "event", evt)
			return
		}

		// Enqueue only if it's a cache miss.
		if c.CacheMiss(evt.Object) {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      evt.Object.GetName(),
				Namespace: evt.Object.GetNamespace(),
			}})
		}
	}
	return hdler
}
