package v1

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile implements the composite controller reconciliation.
func (c CompositeReconciler) Reconcile(req ctrl.Request) (result ctrl.Result, reterr error) {
	result = ctrl.Result{}
	reterr = nil

	ctx := context.Background()
	controller := c.C

	// Initialize the reconciler.
	controller.InitReconcile(ctx, req)
	if fetchErr := controller.FetchInstance(); fetchErr != nil {
		reterr = client.IgnoreNotFound(fetchErr)
		return
	}

	// Add defaults to the primary object instance.
	controller.Default()

	// Validate the instance spec.
	if valErr := controller.Validate(); valErr != nil {
		reterr = valErr
		c.Log.Info("object validation failed", "error", valErr)
		return
	}

	// Save the instance before operating on it in memory.
	controller.SaveClone()

	// Initialize the primary object if uninitialized.
	init := controller.IsUninitialized()
	if init {
		c.Log.Info("initializing", "instance", controller.GetObjectMetadata().Name)
		if initErr := controller.Initialize(c.InitCondition); initErr != nil {
			c.Log.Info("initialization failed", "error", initErr)
			reterr = initErr
			return
		}
	}

	// If finalizer is set, check for deletion.
	if c.FinalizerName != "" {
		if delErr := DeletionCheck(controller, c.FinalizerName); delErr != nil {
			result = ctrl.Result{Requeue: true}
			reterr = delErr
			return
		}
	}

	// Attempt to patch the status after each reconciliation.
	defer func() {
		if err := controller.PatchStatus(); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while patching status: %s", err)})
		}
	}()

	if updateErr := controller.UpdateStatus(); updateErr != nil {
		result = ctrl.Result{Requeue: true}
		reterr = updateErr
		return
	}

	// Run the operation.
	result, event, opErr := controller.Operate()
	if opErr != nil {
		c.Log.Info("failed to finish Operation", "error", opErr)
		reterr = opErr
	}

	// Record an event if the operation returned one.
	if event != nil {
		event.Record(c.Recorder)
	}

	return
}
