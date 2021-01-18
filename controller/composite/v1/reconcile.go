package v1

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/darkowlzz/operator-toolkit/object"
)

// Reconcile implements the composite controller reconciliation.
func (c *CompositeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	tr := otel.Tracer("Reconcile")
	ctx, span := tr.Start(ctx, "reconcile")
	defer span.End()

	controller := c.ctrlr

	// Get an instance of the target object.
	instance := c.prototype.DeepCopyObject().(client.Object)
	if getErr := c.client.Get(ctx, req.NamespacedName, instance); getErr != nil {
		reterr = client.IgnoreNotFound(getErr)
		return
	}

	// Add defaults to the primary object instance.
	span.AddEvent("Populate defaults")
	controller.Default(ctx, instance)

	// Validate the instance spec.
	span.AddEvent("Validate")
	if valErr := controller.Validate(ctx, instance); valErr != nil {
		reterr = valErr
		c.log.Info("object validation failed", "error", valErr)
		return
	}

	// Save the instance before operating on it in memory.
	oldInstance := instance.DeepCopyObject()

	init, initErr := object.IsInitialized(c.scheme, instance)
	if initErr != nil {
		reterr = initErr
		return
	}

	// Initialize the instance if not initialized.
	if !init {
		span.AddEvent("Initialize instance")
		c.log.Info("initializing", "instance", instance.GetName())
		if initErr := controller.Initialize(ctx, instance, c.initCondition); initErr != nil {
			c.log.Info("initialization failed", "error", initErr)
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
		if updateErr := controller.UpdateStatus(ctx, instance); updateErr != nil {
			result = ctrl.Result{Requeue: true}
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while updating status: %v", updateErr)})
			return
		}

		span.AddEvent("Patch status")

		// Compare the old instance status with the updated instance status
		// and patch the status if there's a diff.
		changed, statusChngErr := object.StatusChanged(c.scheme, oldInstance, instance)
		if statusChngErr != nil {
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while checking for changed status: %v", statusChngErr)})
		}

		if changed {
			// ?: Should patch status only if reterr is nil?
			if statusErr := c.client.Status().Update(ctx, instance); statusErr != nil {
				reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while patching status: %v", statusErr)})
			}
		}
	}()

	// If the cleanup strategy is finalizer based, call the cleanup handler.
	if c.cleanupStrategy == FinalizerCleanup {
		span.AddEvent("Trigger cleanup")
		delEnabled, cResult, cErr := c.cleanupHandler(ctx, instance)
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
	result, reterr = controller.Operate(ctx, instance)
	if reterr != nil {
		c.log.Info("failed to finish Operation", "error", reterr)
	}

	return
}

// cleanupHandler checks if the target object is marked for deletion. If not,
// it ensures that a finalizer is added to the target object. If an object is
// marked for deletion, it runs the custom cleanup functions and returns the
// result and error of cleanup. It also returns delEnabled to help the caller
// of this function know that the cleanup process has started.
func (c *CompositeReconciler) cleanupHandler(ctx context.Context, obj client.Object) (delEnabled bool, result ctrl.Result, reterr error) {
	if obj.GetDeletionTimestamp().IsZero() {
		controllerutil.AddFinalizer(obj, c.finalizerName)
	} else {
		delEnabled = true
		if contains(obj.GetFinalizers(), c.finalizerName) {
			result, reterr = c.ctrlr.Cleanup(ctx, obj)
			if reterr != nil {
				c.log.Info("failed to cleanup", "error", reterr)
			}
		}
	}
	return
}

func contains(slice []string, s string) bool {
	for _, element := range slice {
		if element == s {
			return true
		}
	}
	return false
}
