package predicate

import (
	"reflect"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var log = ctrl.Log.WithName("predicate").WithName("eventFilters")

// FinalizerChangedPredicate implements a default update predicate function on
// finalizer change.
//
// This predicate will skip update events that have no change in the object's
// finalizer.
// It is intended to be used in conjunction with the
// GenerationChangedPredicate, as in the following example:
//
// Controller.Watch(
//		&source.Kind{Type: v1.MyCustomKind},
// 		&handler.EnqueueRequestForObject{},
//		predicate.Or(predicate.GenerationChangedPredicate{}, predicate.FinalizerChangedPredicate{}))
//
// This is mostly useful for controllers that needs to trigger both when the
// resource's generation is incremented (i.e., when the resource' .spec
// changes), or an finalizer changes.
type FinalizerChangedPredicate struct {
	predicate.Funcs
}

func (FinalizerChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil {
		log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if e.ObjectNew == nil {
		log.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}

	return !reflect.DeepEqual(e.ObjectNew.GetFinalizers(), e.ObjectOld.GetFinalizers())
}
