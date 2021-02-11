package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/darkowlzz/operator-toolkit/controller/external/builder"
	"github.com/darkowlzz/operator-toolkit/controller/external/handler"
	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
)

// NOTE: In the comments below, the term "space" is used instead of
// "external system".
// This example demonstrates usage of the external controller that uses the
// same controller-runtime components that a normal k8s controller does.
// Instead of using informers to fetch events of a target object, this creates
// a generic event channel which is populated by a goroutine which fetches data
// from an external source. The data is stored in a cache. The source event
// handler uses the cache to decide if an object should be enqueued for
// reconciliation or ignored.
// Since this contoller uses the same internal components of a k8s controller,
// the controller options can be set to configure the controller with sync
// period, reconciliation concurrency, etc.

// SpaceReconciler reconciles external object from space.
type SpaceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	// Cache is the external object cache. The data obtained from space is
	// stored in the cache. This is also used by the event handler to determine
	// if an object should be reconciled or not.
	Cache *FakeCache
}

// Reconcile is part of the main space reconciles loop which aims to move the
// current state of the cluster closer to the desired state.
func (r *SpaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("space", req.NamespacedName)

	nn := req.NamespacedName
	key := getKey(nn.Name, nn.Namespace)

	// Query the object details from the cache and operate on it.
	val, found := r.Cache.Get(key)
	if found {
		log.Info("reconciling", "obj", nn, "value", val)
	} else {
		log.Info("object not found")
	}

	return ctrl.Result{}, nil
}

func (r *SpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Create an generic event source. This is used by the Channel type source
	// to collect the events and process with source event handler.
	src := make(chan event.GenericEvent)

	// Initialize the cache.
	r.Cache = NewFakeCache(r.Log)

	// Create an event handler that uses the cache to make reconciliation
	// decisions.
	eventHandler := handler.NewEnqueueRequestFromCache(r.Cache)

	// Periodically populate the cache from space.
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			<-ticker.C
			r.Log.Info("polling space for data")
			src <- event.GenericEvent{
				Object: &appv1alpha1.Game{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-object",
						Namespace: "test-namespace",
					},
				},
			}
		}
	}()

	return builder.ControllerManagedBy(mgr).
		Named("space-controller").
		WithSource(src).
		WithEventHandler(eventHandler).
		Complete(r)
}
