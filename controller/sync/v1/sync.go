package v1

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncReconciler defines a sync reconciler.
type SyncReconciler struct {
	Name          string
	Ctrlr         Controller
	Prototype     client.Object
	PrototypeList client.ObjectList
	Client        client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	SyncFuncs     []SyncFunc
}

// SyncReconcilerOptions is used to configure SyncReconciler.
type SyncReconcilerOptions func(*SyncReconciler)

// WithName sets the name of the SyncReconciler.
func WithName(name string) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Name = name
	}
}

// WithClient sets the k8s client in the reconciler.
func WithClient(cli client.Client) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Client = cli
	}
}

// WithPrototype sets a prototype of the object that's reconciled.
func WithPrototype(obj client.Object) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Prototype = obj
	}
}

// WithLogger sets the Logger in a SyncReconciler.
func WithLogger(log logr.Logger) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Log = log
	}
}

// WithController sets the Controller in a SyncReconciler.
func WithController(ctrlr Controller) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Ctrlr = ctrlr
	}
}

// WithScheme sets the runtime Scheme of the SyncReconciler.
func WithScheme(scheme *runtime.Scheme) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.Scheme = scheme
	}
}

// WithSyncFuncs sets the syncFuncs of the SyncReconciler.
func WithSyncFuncs(sf []SyncFunc) SyncReconcilerOptions {
	return func(s *SyncReconciler) {
		s.SyncFuncs = sf
	}
}

// Init initializes the SyncReconciler for a given Object with the given
// options.
func (s *SyncReconciler) Init(mgr ctrl.Manager, prototype client.Object, prototypeList client.ObjectList, opts ...SyncReconcilerOptions) error {
	// Use manager if provided. This is helpful in tests to provide explicit
	// client and scheme without a manager.
	if mgr != nil {
		s.Client = mgr.GetClient()
		s.Scheme = mgr.GetScheme()
	}

	// Use prototype and prototypeList if provided.
	if prototype != nil {
		s.Prototype = prototype
	}
	if prototypeList != nil {
		s.PrototypeList = prototypeList
	}

	// Add defaults.
	s.Log = ctrl.Log

	// Run the options to override the defaults.
	for _, opt := range opts {
		opt(s)
	}

	// If a name is set, log it as the reconciler name.
	if s.Name != "" {
		s.Log = s.Log.WithValues("reconciler", s.Name)
	}

	// Perform validation.
	if s.Ctrlr == nil {
		return fmt.Errorf("must provide a Controller to the SyncReconciler")
	}

	// Run the sync functions.
	s.RunSyncFuncs()

	return nil
}

// RunSyncFuncs runs all the SyncFuncs in go routines.
func (s *SyncReconciler) RunSyncFuncs() {
	for _, sf := range s.SyncFuncs {
		go sf.Run()
	}
}
