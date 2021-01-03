package v1

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// DefaultGarbageCollectionPeriod is the default period at which garbage
	// collection is executed.
	DefaultGarbageCollectionPeriod time.Duration = 5 * time.Minute
)

// SyncReconciler defines an external object sync reconciler.
type SyncReconciler struct {
	name                    string
	ctrlr                   Controller
	prototype               client.Object
	prototypeList           client.ObjectList
	client                  client.Client
	scheme                  *runtime.Scheme
	log                     logr.Logger
	enableGarbageCollector  bool
	garbageCollectionPeriod time.Duration
}

// SyncReconcilerOptions is used to configure SyncReconciler.
type SyncReconcilerOptions func(*SyncReconciler)

// WithName sets the name of the SyncReconciler.
func WithName(name string) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.name = name
	}
}

// WithClient sets the k8s client in the reconciler.
func WithClient(cli client.Client) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.client = cli
	}
}

// WithPrototype sets a prototype of the object that's reconciled.
func WithPrototype(obj client.Object) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.prototype = obj
	}
}

// WithGarbageCollectorEnabled can be used to enable garbage collector.
func WithGarbageCollectorEnabled(enable bool) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.enableGarbageCollector = enable
	}
}

// WithGarbageCollectionPeriod sets the garbage collection period.
func WithGarbageCollectionPeriod(p time.Duration) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.garbageCollectionPeriod = p
	}
}

// WithLogger sets the Logger in a SyncReconciler.
func WithLogger(log logr.Logger) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.log = log
	}
}

// WithController sets the Controller in a SyncReconciler.
func WithController(ctrlr Controller) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.ctrlr = ctrlr
	}
}

// WithScheme sets the runtime Scheme of the SyncReconciler.
func WithScheme(scheme *runtime.Scheme) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.scheme = scheme
	}
}

// Init initializes the SyncReconciler for a given Object with the given
// options.
func (s *SyncReconciler) Init(mgr ctrl.Manager, prototype client.Object, prototypeList client.ObjectList, opts ...SyncReconcilerOptions) error {
	// Use manager if provided. This is helpful in tests to provide explicit
	// client and scheme without a manager.
	if mgr != nil {
		s.client = mgr.GetClient()
		s.scheme = mgr.GetScheme()
	}

	// Use prototype and prototypeList if provided.
	if prototype != nil {
		s.prototype = prototype
	}
	if prototypeList != nil {
		s.prototypeList = prototypeList
	}

	// Add defaults.
	s.log = ctrl.Log
	s.enableGarbageCollector = true
	s.garbageCollectionPeriod = DefaultGarbageCollectionPeriod

	// Run the options to override the defaults.
	for _, opt := range opts {
		opt(s)
	}

	// If a name is set, log it as the reconciler name.
	if s.name != "" {
		s.log = s.log.WithValues("reconciler", s.name)
	}

	// Perform validation.
	if s.ctrlr == nil {
		return fmt.Errorf("must provide a Controller to the SyncReconciler")
	}

	if s.enableGarbageCollector {
		// Start the garbage collector manager as a separate goroutine.
		go s.runGarbageCollectorManager()
	}

	return nil
}

// runGarbageCollector starts a ticker for garbage collector to be executed
// periodically.
func (s *SyncReconciler) runGarbageCollectorManager() {
	ticker := time.NewTicker(s.garbageCollectionPeriod)
	defer ticker.Stop()

	for {
		<-ticker.C
		s.collectGarbage()
	}
}

// collectGarbage lists all the prototype objects in k8s and the associated
// objects in the external system and compares them. It deletes all the objects
// in the external system that don't have an associated k8s object.
func (s *SyncReconciler) collectGarbage() {
	s.log.WithValues("garbage-collector", s.name)

	controller := s.ctrlr

	// TODO: Provide option to set timeout for the garbage collection. Since
	// this runs in a goroutine, when the reconcile has a timeout duration, use
	// it with the created context.
	ctx := context.Background()

	// List all the k8s objects.
	instances := s.prototypeList.DeepCopyObject().(client.ObjectList)
	// TODO: Provide option to set a namespace and other list options.
	if listErr := s.client.List(ctx, instances); listErr != nil {
		s.log.Info("failed to list", "error", listErr)
		return
	}

	// Convert all k8s objects to list of namespaced names.
	kObjList, nsnErr := ExtractNamespacedNamesFromList(instances)
	if nsnErr != nil {
		s.log.Info("failed to extract namespaced names: %w", nsnErr)
		return
	}

	// List all the external objects.
	extObjList, listErr := controller.List(ctx)
	if listErr != nil {
		s.log.Info("failed to list external objects", "error", listErr)
		return
	}

	// Diff the objects and obtain a list of external objects to delete.
	delObjs, diffErr := DiffExternalObjects(kObjList, extObjList)
	if diffErr != nil {
		s.log.Info("failed to get diff of objects", "error", diffErr)
		return
	}

	s.log.Info("garbage collecting external objects", "objects", delObjs)

	for _, obj := range delObjs {
		// Create an instance of the object and populate with namespaced name
		// info.
		instance := s.prototype.DeepCopyObject().(client.Object)
		instance.SetName(obj.Name)
		instance.SetNamespace(obj.Namespace)
		if err := controller.Delete(ctx, instance); err != nil {
			s.log.Info("failed to delete external object", "instance", instance, "error", err)
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
