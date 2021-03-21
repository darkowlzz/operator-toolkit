// This is taken from
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.3/pkg/source/internal/eventsource.go.
// Since it's an internal package, it couldn't be imported for use with the
// Kind source.
package internal

import (
	"fmt"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = ctrl.Log.WithName("source").WithName("EventHandler")

var _ cache.ResourceEventHandler = EventHandler{}

// EventHandler adapts a handler.EventHandler interface to a cache.ResourceEventHandler interface
type EventHandler struct {
	EventHandler handler.EventHandler
	Queue        workqueue.RateLimitingInterface
	Predicates   []predicate.Predicate
}

// OnAdd creates CreateEvent and calls Create on EventHandler
func (e EventHandler) OnAdd(obj interface{}) {
	c := event.CreateEvent{}

	// Pull Object out of the object
	if o, ok := obj.(client.Object); ok {
		c.Object = o
	} else {
		log.Error(nil, "OnAdd missing Object",
			"object", obj, "type", fmt.Sprintf("%T", obj))
		return
	}

	for _, p := range e.Predicates {
		if !p.Create(c) {
			return
		}
	}

	// Invoke create handler
	e.EventHandler.Create(c, e.Queue)
}

// OnUpdate creates UpdateEvent and calls Update on EventHandler
func (e EventHandler) OnUpdate(oldObj, newObj interface{}) {
	u := event.UpdateEvent{}

	if o, ok := oldObj.(client.Object); ok {
		u.ObjectOld = o
	} else {
		log.Error(nil, "OnUpdate missing ObjectOld",
			"object", oldObj, "type", fmt.Sprintf("%T", oldObj))
		return
	}

	// Pull Object out of the object
	if o, ok := newObj.(client.Object); ok {
		u.ObjectNew = o
	} else {
		log.Error(nil, "OnUpdate missing ObjectNew",
			"object", newObj, "type", fmt.Sprintf("%T", newObj))
		return
	}

	for _, p := range e.Predicates {
		if !p.Update(u) {
			return
		}
	}

	// Invoke update handler
	e.EventHandler.Update(u, e.Queue)
}

// OnDelete creates DeleteEvent and calls Delete on EventHandler
func (e EventHandler) OnDelete(obj interface{}) {
	d := event.DeleteEvent{}

	// Deal with tombstone events by pulling the object out.  Tombstone events wrap the object in a
	// DeleteFinalStateUnknown struct, so the object needs to be pulled out.
	// Copied from sample-controller
	// This should never happen if we aren't missing events, which we have concluded that we are not
	// and made decisions off of this belief.  Maybe this shouldn't be here?
	var ok bool
	if _, ok = obj.(client.Object); !ok {
		// If the object doesn't have Metadata, assume it is a tombstone object of type DeletedFinalStateUnknown
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			log.Error(nil, "Error decoding objects.  Expected cache.DeletedFinalStateUnknown",
				"type", fmt.Sprintf("%T", obj),
				"object", obj)
			return
		}

		// Set obj to the tombstone obj
		obj = tombstone.Obj
	}

	// Pull Object out of the object
	if o, ok := obj.(client.Object); ok {
		d.Object = o
	} else {
		log.Error(nil, "OnDelete missing Object",
			"object", obj, "type", fmt.Sprintf("%T", obj))
		return
	}

	for _, p := range e.Predicates {
		if !p.Delete(d) {
			return
		}
	}

	// Invoke delete handler
	e.EventHandler.Delete(d, e.Queue)
}
