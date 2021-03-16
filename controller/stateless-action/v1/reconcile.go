package v1

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action"
)

// Reconciler is the StatelessAction reconciler.
type Reconciler struct {
	name   string
	ctrlr  Controller
	actmgr action.Manager
	log    logr.Logger
	client client.Client
	scheme *runtime.Scheme

	actionRetryPeriod time.Duration
	actionTimeout     time.Duration
}

// ReconcilerOption is used to configure Reconciler.
type ReconcilerOption func(*Reconciler)

// WithName sets the name of the Reconciler.
func WithName(name string) ReconcilerOption {
	return func(r *Reconciler) {
		r.name = name
	}
}

// WithActionRetryPeriod sets the action retry period when it fails.
func WithActionRetryPeriod(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.actionRetryPeriod = duration
	}
}

func WithActionTimeout(duration time.Duration) ReconcilerOption {
	return func(r *Reconciler) {
		r.actionTimeout = duration
	}
}

// WithLogger sets the Logger in a Reconciler.
func WithLogger(log logr.Logger) ReconcilerOption {
	return func(r *Reconciler) {
		r.log = log
	}
}

// WithScheme sets the runtime Scheme of the Reconciler.
func WithScheme(scheme *runtime.Scheme) ReconcilerOption {
	return func(r *Reconciler) {
		r.scheme = scheme
	}
}

func (r *Reconciler) Init(mgr ctrl.Manager, ctrlr Controller, actmgr action.Manager, opts ...ReconcilerOption) {
	r.ctrlr = ctrlr
	r.actmgr = actmgr

	// Use manager if provided. This is helpful in tests to provide explicit
	// client and scheme without a manager.
	if mgr != nil {
		r.client = mgr.GetClient()
		r.scheme = mgr.GetScheme()
	}

	// Add defaults.
	r.log = ctrl.Log

	// Run the options to override the defaults.
	for _, opt := range opts {
		opt(r)
	}

	// If a name is set, log it as the reconciler name.
	if r.name != "" {
		r.log = r.log.WithValues("reconciler", r.name)
	}
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	tr := otel.Tracer("Reconcile")
	_, span := tr.Start(ctx, "reconcile")
	defer span.End()

	controller := r.ctrlr

	// Get an instance of the target object.
	// NOTE: Since the object can be fetched from any backend, we don't know
	// about the error code to be able to perform a proper not found error
	// check. If it's a k8s apimachinery "not found" error, ignore it. Any
	// other error will result in returning error. In order to ignore not found
	// from other backend, return a nil object.
	obj, err := controller.GetObject(req.NamespacedName)
	if err != nil {
		reterr = client.IgnoreNotFound(err)
		return
	}
	// Return if the object is nil.
	if obj == nil {
		return
	}

	// Check if an action is required.
	requireAction, err := controller.RequireAction(obj)
	if err != nil {
		reterr = err
		return
	}

	// If an action is required, run an action manager for the target object.
	if requireAction {
		if err := r.RunActionManager(obj); err != nil {
			reterr = err
			return
		}
	}

	return
}

// RunActionManager runs the actions in the action manager based on the given
// object.
func (r *Reconciler) RunActionManager(o interface{}) error {
	// Get the objects to run action on.
	objects, err := r.actmgr.GetObjects()
	if err != nil {
		r.log.Info("failed to get objects from action manager", "error", err)
		return err
	}

	// Run the action in a goroutine.
	for _, obj := range objects {
		go r.RunAction(obj)
	}

	return nil
}

// RunAction checks if an action needs to be run before running it. It also
// runs a deferred function at the end.
func (r *Reconciler) RunAction(o interface{}) {
	log := r.log.WithValues("object", o)

	// Create a context with timeout.
	ctx, cancel := context.WithTimeout(context.Background(), r.actionTimeout)
	defer cancel()

	// Defer the Defer() function.
	defer func() {
		r.actmgr.Defer(ctx, o)
	}()

	// First run.
	r.actmgr.Run(ctx, o)

	// Check and run the action periodically if the check fails.
	for {
		select {
		case <-time.After(r.actionRetryPeriod):
			log.Info("checking action status")
			if r.actmgr.Check(ctx, o) {
				log.Info("retrying")
				r.actmgr.Run(ctx, o)
			} else {
				// Action successful, end the action.
				log.Info("action successful", "object", o)
				return
			}
		case <-ctx.Done():
			log.Info("context done, terminating action")
			return
		}
	}
}
