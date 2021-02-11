package builder

import (
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/darkowlzz/operator-toolkit/controller/external/source"
)

// Builder builds a Controller.
type Builder struct {
	evntSrc     <-chan event.GenericEvent
	hdler       handler.EventHandler
	mgr         manager.Manager
	ctrl        controller.Controller
	ctrlOptions controller.Options
	name        string
}

// ControllerManagedBy returns a new controller builder that will be started by
// the provided Manager.
func ControllerManagedBy(m manager.Manager) *Builder {
	return &Builder{mgr: m}
}

// WithSource sets the generic event source of the controller.
func (blder *Builder) WithSource(src <-chan event.GenericEvent) *Builder {
	blder.evntSrc = src
	return blder
}

// WithEventHandler sets the source event handler.
func (blder *Builder) WithEventHandler(h handler.EventHandler) *Builder {
	blder.hdler = h
	return blder
}

// WithOptions overrides the controller options use in doController. Defaults
// to empty.
func (blder *Builder) WithOptions(options controller.Options) *Builder {
	blder.ctrlOptions = options
	return blder
}

// WithLogger overrides the controller options's logger used.
func (blder *Builder) WithLogger(log logr.Logger) *Builder {
	blder.ctrlOptions.Log = log
	return blder
}

// Named sets the name of the controller to the given name. The name shows up
// in metrics, among other things, and thus should be a prometheus compatible name
// (underscores and alphanumeric characters only).
//
// By default, controllers are named using the lowercase version of their kind.
func (blder *Builder) Named(name string) *Builder {
	blder.name = name
	return blder
}

// Complete builds the Application ControllerManagedBy.
func (blder *Builder) Complete(r reconcile.Reconciler) error {
	_, err := blder.Build(r)
	return err
}

// Build builds the complete controller by validating the configuration,
// setting up the controller and starting the event source watcher.
func (blder *Builder) Build(r reconcile.Reconciler) (controller.Controller, error) {
	if r == nil {
		return nil, fmt.Errorf("must provide a non-nil Reconciler")
	}
	if blder.mgr == nil {
		return nil, fmt.Errorf("must provide a non-nil Manager")
	}

	// Set the ControllerManagedBy.
	if err := blder.doController(r); err != nil {
		return nil, err
	}

	// Set the Watch.
	if err := blder.doWatch(); err != nil {
		return nil, err
	}

	return blder.ctrl, nil
}

// doWatch sets up Watcher for the event source.
func (blder *Builder) doWatch() error {
	src := source.NewChannel(blder.evntSrc)

	if err := blder.ctrl.Watch(src, blder.hdler); err != nil {
		return err
	}

	return nil
}

// doController sets up a new Controller.
func (blder *Builder) doController(r reconcile.Reconciler) error {
	ctrlOptions := blder.ctrlOptions
	if ctrlOptions.Reconciler == nil {
		ctrlOptions.Reconciler = r
	}

	// Setup the logger.
	if ctrlOptions.Log == nil {
		ctrlOptions.Log = blder.mgr.GetLogger()
	}

	var err error
	blder.ctrl, err = controller.New(blder.name, blder.mgr, ctrlOptions)
	return err
}
