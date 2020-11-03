package v1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile implements the composite controller reconciliation.
func (c CompositeReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	controller := c.C

	// Initialize the reconciler.
	controller.InitReconcile(ctx, req)
	if err := controller.FetchInstance(); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Initialize the primary object if uninitialized.
	init := controller.IsUninitialized()
	if init {
		c.Log.Info("initializing", "instance", controller.GetObjectMetadata().Name)
		if err := controller.Initialize(c.InitCondition); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check finalizers for cleanup.
	if err := DeletionCheck(controller, c.FinalizerName); err != nil {
		return ctrl.Result{}, err
	}

	// Check for drift and run operation.
	drifted, err := controller.CheckForDrift()
	if err != nil {
		return ctrl.Result{}, err
	}
	if drifted {
		c.Log.Info("drift detected")
		if runErr := controller.RunOperation(); runErr != nil {
			return ctrl.Result{}, runErr
		}
	}

	// Requeue if we just initialized everything to reconcile on the new object
	// version after the initialization updates above.
	if init {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}
