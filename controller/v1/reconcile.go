package v1

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Reconcile implements the composite controller reconciliation.
func (c *CompositeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	tr := otel.Tracer("Reconcile")
	ctx, span := tr.Start(ctx, "reconcile")
	defer span.End()

	result = ctrl.Result{}
	reterr = nil

	controller := c.Ctrlr

	// Initialize the reconciler.
	span.AddEvent("Initialize reconcile")
	controller.InitReconcile(ctx, req)
	if fetchErr := controller.FetchInstance(ctx); fetchErr != nil {
		reterr = client.IgnoreNotFound(fetchErr)
		return
	}

	// Add defaults to the primary object instance.
	span.AddEvent("Populate defaults")
	controller.Default()

	// Validate the instance spec.
	span.AddEvent("Validate")
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
		span.AddEvent("Initialize instance")
		c.Log.Info("initializing", "instance", controller.GetObjectMetadata().Name)
		if initErr := controller.Initialize(c.InitCondition); initErr != nil {
			c.Log.Info("initialization failed", "error", initErr)
			reterr = initErr
			return
		}
	}

	// Attempt to patch the status after each reconciliation.
	defer func() {
		// Update the local copy of the target object status based on the state of
		// the world.
		// NOTE: The actual target object gets updated in the API server at the end
		// of the control loop with the deferred PatchStatus.
		span.AddEvent("Get status updates")
		if updateErr := controller.UpdateStatus(ctx); updateErr != nil {
			result = ctrl.Result{Requeue: true}
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while updating status: %v", updateErr)})
			return
		}

		span.AddEvent("Patch status")
		// ?: Should patch status only if reterr is nil?
		if err := controller.PatchStatus(ctx); err != nil {
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while patching status: %v", err)})
		}
	}()

	// If the cleanup strategy is finalizer based, call the cleanup handler.
	if c.CleanupStrategy == FinalizerCleanup {
		span.AddEvent("Trigger cleanup")
		delEnabled, cResult, cErr := c.cleanupHandler(ctx)
		// If the deletion of the target object has started, return with the
		// result and error.
		if delEnabled || cErr != nil {
			result = cResult
			reterr = cErr
			return
		}
	}

	// Run the operation.
	span.AddEvent("Run Operate")
	result, reterr = controller.Operate(ctx)
	if reterr != nil {
		c.Log.Info("failed to finish Operation", "error", reterr)
	}

	return
}

// cleanupHandler checks if the target object is marked for deletion. If not,
// it ensures that a finalizer is added to the target object. If an object is
// marked for deletion, it runs the custom cleanup functions and returns the
// result and error of cleanup. It also returns delEnabled to help the caller
// of this function know that the cleanup process has started.
func (c *CompositeReconciler) cleanupHandler(ctx context.Context) (delEnabled bool, result ctrl.Result, reterr error) {
	metadata := c.Ctrlr.GetObjectMetadata()
	if metadata.DeletionTimestamp.IsZero() {
		if !contains(metadata.Finalizers, c.FinalizerName) {
			if ferr := c.Ctrlr.AddFinalizer(c.FinalizerName); ferr != nil {
				reterr = ferr
				result = ctrl.Result{Requeue: true}
			}
		}
	} else {
		delEnabled = true
		if contains(metadata.Finalizers, c.FinalizerName) {
			result, reterr = c.Ctrlr.Cleanup(ctx)
			if reterr != nil {
				c.Log.Info("failed to cleanup", "error", reterr)
			}
		}
	}
	return
}
