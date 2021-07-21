package v1

import (
	"context"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	tkctrl "github.com/darkowlzz/operator-toolkit/controller"
	"github.com/darkowlzz/operator-toolkit/object"
)

// Reconcile implements the composite controller reconciliation.
func (c *CompositeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	ctx, span, _, log := c.inst.Start(ctx, "Reconcile")
	defer span.End()

	start := time.Now()
	defer tkctrl.LogReconcileFinish(log, "reconciliation finished", start, &result, &reterr)

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
		log.Error(valErr, "object validation failed")
		return
	}

	// Save the instance before operating on it in memory.
	oldInstance := instance.DeepCopyObject()

	init, initErr := object.IsInitialized(c.scheme, instance)
	if initErr != nil {
		reterr = initErr
		return
	}

	// NOTE: The init and finalizer blocks below return with `Requeue: true`
	// to keep the main reconciliation action separate from initial setup
	// steps. This helps ensure that the status and finalizers of the object
	// have the correct data while the main reconciliation actions are in
	// progress.

	// Initialize the instance if not initialized and update.
	if !init {
		log.Info("initializing", "instance", instance.GetName())
		if initErr := controller.Initialize(ctx, instance, c.initCondition); initErr != nil {
			log.Error(initErr, "initialization failed")
			reterr = initErr
			return
		}

		// Update the object status in the API.
		if updateErr := c.client.Status().Update(ctx, instance); updateErr != nil {
			log.Error(updateErr, "failed to update initialized object")
		}
		span.AddEvent("Updated object status")
		result = ctrl.Result{Requeue: true}
		return
	}

	// skipStatusUpdate is used to skip the deferred status update when it's
	// known that another reconciliation will take place. An example usage of
	// this is when the cleanupHandler() below adds a finalizer to the target
	// object, the existing instance of the object becomes old. Fetching a new
	// instance in UpdateStatus() in a very short time sometimes results in
	// fetching the cached old version of the object. Attempting an update with
	// this object results in object modified error from the API. Since adding
	// finalizer updated the target object, it's known that another
	// reconciliation will take place and it's okay to skip status update.
	skipStatusUpdate := false

	// Attempt to patch the status after each reconciliation.
	defer func() {
		if skipStatusUpdate {
			span.AddEvent("Skipping status update")
			return
		}

		// Update the local copy of the target object status based on the state of
		// the world.
		// NOTE: The actual target object gets updated in the API server at the end
		// of the control loop with the deferred PatchStatus.
		span.AddEvent("Get status updates")
		if updateErr := controller.UpdateStatus(ctx, instance); updateErr != nil {
			span.RecordError(updateErr)
			result = ctrl.Result{Requeue: true}
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while updating status: %v", updateErr)})
			return
		}

		span.AddEvent("Checking for status change")

		// Compare the old instance status with the updated instance status
		// and patch the status if there's a diff.
		changed, statusChngErr := object.StatusChanged(c.scheme, oldInstance, instance)
		if statusChngErr != nil {
			reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while checking for changed status: %v", statusChngErr)})
		}

		if changed {
			span.AddEvent("Found status change, updating object")
			// ?: Should patch status only if reterr is nil?
			if statusErr := c.client.Status().Update(ctx, instance); statusErr != nil {
				reterr = kerrors.NewAggregate([]error{reterr, fmt.Errorf("error while patching status: %v", statusErr)})
			}
		} else {
			span.AddEvent("No status change found")
		}
	}()

	// If the cleanup strategy is finalizer based, call the cleanup handler.
	if c.cleanupStrategy == FinalizerCleanup {
		span.AddEvent("Handle finalizers")
		delEnabled, updated, cResult, cErr := c.cleanupHandler(ctx, instance)
		if updated {
			log.Info("Finalizers updated")
			// Object updated, skip deferred status update and let the
			// subsequent reconciliation handle the status udpate.
			skipStatusUpdate = true
		}
		// If the deletion of the target object has started, return with the
		// result and error. Also, return if an update took place.
		if updated || delEnabled || cErr != nil {
			result = cResult
			reterr = cErr
			return
		}
	}

	// Run the operation.
	span.AddEvent("Run Operate")
	result, reterr = controller.Operate(ctx, instance)
	if reterr != nil {
		log.Error(reterr, "failed to finish Operation")
	}

	return
}

// cleanupHandler checks if the target object is marked for deletion. If not,
// it ensures that a finalizer is added to the target object. If an object is
// marked for deletion, it runs the custom cleanup functions and returns the
// result and error of cleanup. It returns delEnabled to help the caller of
// this function know that the cleanup process has started. It also returns
// updated which tells the caller about an API update, usually update to the
// finalizers in the object.
func (c *CompositeReconciler) cleanupHandler(ctx context.Context, obj client.Object) (delEnabled bool, updated bool, result ctrl.Result, reterr error) {
	ctx, span, _, log := c.inst.Start(ctx, "cleanupHandler")
	defer span.End()

	if obj.GetDeletionTimestamp().IsZero() {
		span.AddEvent("No delete timestamp")
		// If the object does not contain finalizer, add it.
		if !controllerutil.ContainsFinalizer(obj, c.finalizerName) {
			span.AddEvent("Finalizer not found, updating object to add finalizer")
			controllerutil.AddFinalizer(obj, c.finalizerName)
			if updateErr := c.client.Update(ctx, obj); updateErr != nil {
				log.Error(updateErr, "failed to add finalizer")
			}
			// Mark API object update.
			updated = true
		} else {
			span.AddEvent("Finalizer exists, no-op")
		}
	} else {
		span.AddEvent("Delete timestamp found")
		delEnabled = true

		// Perform cleanup if finalizer is found.
		if contains(obj.GetFinalizers(), c.finalizerName) {
			span.AddEvent("Finalizer found, run cleanup")
			result, reterr = c.ctrlr.Cleanup(ctx, obj)
			if reterr != nil {
				log.Error(reterr, "failed to cleanup")
			} else {
				// Cleanup successful, remove the finalizer.
				span.AddEvent("Cleanup completed, remove finalizer")
				controllerutil.RemoveFinalizer(obj, c.finalizerName)
				if updateErr := c.client.Update(ctx, obj); updateErr != nil {
					log.Error(updateErr, "failed to remove finalizer")
				}
				// Mark API object update.
				updated = true
			}
		} else {
			span.AddEvent("Finalizer not found, no-op")
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
