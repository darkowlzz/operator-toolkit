package v1

import (
	"context"
	"fmt"
	"time"

	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	disableGarbageCollector bool
	garbageCollectionPeriod time.Duration
}

// DisableGarbageCollector disables the garbage collector.
func (s *Reconciler) DisableGarbageCollector() {
	s.disableGarbageCollector = true
}

// SetGarbageCollectionPeriod sets the garbage collection period.
func (s *Reconciler) SetGarbageCollectionPeriod(period time.Duration) {
	s.garbageCollectionPeriod = period
}

// Init initializes the reconciler.
func (s *Reconciler) Init(mgr ctrl.Manager, prototype client.Object, prototypeList client.ObjectList, opts ...syncv1.ReconcilerOptions) error {
	// Add a garbage collector sync func if garbage collector is not disabled.
	if !s.disableGarbageCollector {
		// If the period is zero, use the default period.
		if s.garbageCollectionPeriod == zeroDuration {
			s.garbageCollectionPeriod = DefaultGarbageCollectionPeriod
		}

		sf := syncv1.NewSyncFunc(s.collectGarbage, s.garbageCollectionPeriod)
		sfs := []syncv1.SyncFunc{sf}

		opts = append(opts, syncv1.WithSyncFuncs(sfs))
	}

	// Initialize the base sync reconciler.
	return s.Reconciler.Init(mgr, prototype, prototypeList, opts...)
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
	kObjList, nsnErr := ExtractNamespacedNamesFromList(instances)
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

	// Diff the objects and obtain a list of external objects to delete.
	delObjs, diffErr := DiffExternalObjects(kObjList, extObjList)
	if diffErr != nil {
		s.Log.Info("failed to get diff of objects", "error", diffErr)
		return
	}

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

// ExtractNamespacedNamesFromList takes an ObjectList, extracts NamespacedName
// info from the items and returns a list of NamespacedName.
func ExtractNamespacedNamesFromList(instances client.ObjectList) ([]types.NamespacedName, error) {
	result := []types.NamespacedName{}

	items, err := apimeta.ExtractList(instances)
	if err != nil {
		return nil, fmt.Errorf("failed to extract objects from object list %v: %w", instances, err)
	}
	for _, item := range items {
		// Get meta object from the item and extract namespace/name info.
		obj, err := apimeta.Accessor(item)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessor for %v: %w", item, err)
		}
		nsn := types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}
		result = append(result, nsn)
	}

	return result, nil
}

// DiffExternalObjects takes a list of k8s objects and external objects and
// returns the list of externals objects that don't exist in k8s.
func DiffExternalObjects(kObjList, extObjList []types.NamespacedName) ([]types.NamespacedName, error) {
	result := []types.NamespacedName{}

	for _, extObj := range extObjList {
		found := false
		for _, kObj := range kObjList {
			if extObj == kObj {
				found = true
				break
			}
		}
		if !found {
			result = append(result, extObj)
		}
	}

	return result, nil
}
