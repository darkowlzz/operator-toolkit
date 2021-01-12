package v1

import (
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconciler defines a sync reconciler.
type Reconciler struct {
	Name          string
	Ctrlr         Controller
	Prototype     client.Object
	PrototypeList client.ObjectList
	Client        client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	SyncFuncs     []SyncFunc
}

// ReconcilerOptions is used to configure Reconciler.
type ReconcilerOptions func(*Reconciler)

// WithName sets the name of the Reconciler.
func WithName(name string) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Name = name
	}
}

// WithClient sets the k8s client in the reconciler.
func WithClient(cli client.Client) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Client = cli
	}
}

// WithPrototype sets a prototype of the object that's reconciled.
func WithPrototype(obj client.Object) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Prototype = obj
	}
}

// WithLogger sets the Logger in a Reconciler.
func WithLogger(log logr.Logger) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Log = log
	}
}

// WithController sets the Controller in a Reconciler.
func WithController(ctrlr Controller) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Ctrlr = ctrlr
	}
}

// WithScheme sets the runtime Scheme of the Reconciler.
func WithScheme(scheme *runtime.Scheme) ReconcilerOptions {
	return func(s *Reconciler) {
		s.Scheme = scheme
	}
}

// WithSyncFuncs sets the syncFuncs of the Reconciler.
func WithSyncFuncs(sf []SyncFunc) ReconcilerOptions {
	return func(s *Reconciler) {
		s.SyncFuncs = sf
	}
}

// Init initializes the Reconciler for a given Object with the given
// options.
func (s *Reconciler) Init(mgr ctrl.Manager, prototype client.Object, prototypeList client.ObjectList, opts ...ReconcilerOptions) error {
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
		return fmt.Errorf("must provide a Controller to the Reconciler")
	}

	// Run the sync functions.
	s.RunSyncFuncs()

	return nil
}

// RunSyncFuncs runs all the SyncFuncs in go routines.
func (s *Reconciler) RunSyncFuncs() {
	for _, sf := range s.SyncFuncs {
		go sf.Run()
	}
}
