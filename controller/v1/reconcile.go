package v1

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile implements the composite controller reconciliation.
func (c CompositeReconciler) Reconcile(req ctrl.Request) (_ ctrl.Result, reterr error) {
	ctx := context.Background()
	controller := c.C

	// Initialize the reconciler.
	controller.InitReconcile(ctx, req)
	if err := controller.FetchInstance(); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Add defaults to the primary object instance.
	controller.Default()

	// Validate the instance spec.
	if err := controller.Validate(); err != nil {
		c.Log.Info("object validation failed", "error", err)
		return ctrl.Result{}, err
	}

	// Save the instance before operating on it in memory.
	controller.SaveClone()

	// Initialize the primary object if uninitialized.
	init := controller.IsUninitialized()
	if init {
		c.Log.Info("initializing", "instance", controller.GetObjectMetadata().Name)
		if err := controller.Initialize(c.InitCondition); err != nil {
			c.Log.Info("initialization failed", "error", err)
			return ctrl.Result{}, err
		}
	}

	// Check finalizers for cleanup.
	if err := DeletionCheck(controller, c.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	// Attempt to patch the status after each reconciliation.
	defer func() {
		if err := controller.UpdateStatus(); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while patching status: %s", err)})
		}
	}()

	if fetchErr := controller.FetchStatus(); fetchErr != nil {
		return ctrl.Result{}, fetchErr
	}

	// Run the operation.
	result, event, err := controller.Operate()
	if err != nil {
		c.Log.Info("failed to finish Operation", "error", err)
	}

	// Record an event if the operation returned one.
	if event != nil {
		event.Record(c.Recorder)
	}

	return result, err
}
