package v1

import (
	"context"
	"time"

	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/object"
)

const (
	// DefaultGarbageCollectionPeriod is the default period at which garbage
	// collection is executed.
	DefaultGarbageCollectionPeriod time.Duration = 5 * time.Minute

	zeroDuration time.Duration = 0 * time.Minute
)

// Reconciler defines an external object sync reconciler based on the Sync
// reconciler with a sync function for garbage collection of the external
// objects.
type Reconciler struct {
	syncv1.Reconciler
	Ctrlr                         Controller
	garbageCollectionPeriod       time.Duration
	startupGarbageCollectionDelay time.Duration
}

// SetGarbageCollectionPeriod sets the garbage collection period.
func (s *Reconciler) SetGarbageCollectionPeriod(period time.Duration) {
	s.garbageCollectionPeriod = period
}

// SetStartupGarbageCollectionDelay sets a delay for the initial garbage
// collection at startup.
// NOTE: Setting this too low can result in failure due to uninitialized
// controller components.
func (s *Reconciler) SetStartupGarbageCollectionDelay(period time.Duration) {
	s.startupGarbageCollectionDelay = period
}

// Init initializes the reconciler.
func (s *Reconciler) Init(mgr ctrl.Manager, ctrlr Controller, prototype client.Object, prototypeList client.ObjectList, opts ...syncv1.ReconcilerOption) error {
	// Add a garbage collector sync func if garbage collection period is not
	// zero.
	if s.garbageCollectionPeriod > zeroDuration {
		sf := syncv1.NewSyncFunc(s.collectGarbage, s.garbageCollectionPeriod, s.startupGarbageCollectionDelay)
		sfs := []syncv1.SyncFunc{sf}

		opts = append(opts, syncv1.WithSyncFuncs(sfs))
	}

	// Set controller.
	s.Ctrlr = ctrlr

	// Initialize the base sync reconciler.
	return s.Reconciler.Init(mgr, ctrlr, prototype, prototypeList, opts...)
}

// collectGarbage lists all the prototype objects in k8s and the associated
// objects in the external system and compares them. It deletes all the objects
// in the external system that don't have an associated k8s object.
func (s *Reconciler) collectGarbage() {
	s.Log.WithValues("garbage-collector", s.Name)

	controller := s.Ctrlr

	// TODO: Provide option to set timeout for the garbage collection. Since
	// this runs in a goroutine, when the reconcile has a timeout duration, use
	// it with the created context.
	ctx := context.Background()

	// List all the k8s objects.
	instances := s.PrototypeList.DeepCopyObject().(client.ObjectList)
	// TODO: Provide option to set a namespace and other list options.
	if listErr := s.Client.List(ctx, instances); listErr != nil {
		s.Log.Info("failed to list", "error", listErr)
		return
	}

	// Convert all k8s objects to list of namespaced names.
	kObjList, nsnErr := object.NamespacedNames(instances)
	if nsnErr != nil {
		s.Log.Info("failed to extract namespaced names: %w", nsnErr)
		return
	}

	// List all the external objects.
	extObjList, listErr := controller.List(ctx)
	if listErr != nil {
		s.Log.Info("failed to list external objects", "error", listErr)
		return
	}

	// Get the list of external objects that are no longer in k8s.
	delObjs := object.NamespacedNamesDiff(extObjList, kObjList)

	s.Log.Info("garbage collecting external objects", "objects", delObjs)

	for _, obj := range delObjs {
		// Create an instance of the object and populate with namespaced name
		// info.
		instance := s.Prototype.DeepCopyObject().(client.Object)
		instance.SetName(obj.Name)
		instance.SetNamespace(obj.Namespace)
		if err := controller.Delete(ctx, instance); err != nil {
			s.Log.Info("failed to delete external object", "instance", instance, "error", err)
		}
	}
}
