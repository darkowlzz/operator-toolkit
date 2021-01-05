package v1

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.opentelemetry.io/otel"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile implements the composite controller reconciliation.
func (c *CompositeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	tr := otel.Tracer("Reconcile")
	ctx, span := tr.Start(ctx, "reconcile")
	defer span.End()

	controller := c.ctrlr

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
	clonedInstance := instance.DeepCopyObject()

	init, initErr := c.isInitialized(instance)
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

		// Compare the cloned instance status with the updated instance status
		// and patch the status if there's a diff.
		changed, statusChngErr := c.statusChanged(clonedInstance, instance)
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

// nestedFieldNoCopy returns the nested field from a given Object. The second
// returned value is true if the field is found, else false.
//
// Taken from kubebuilder-declarative-pattern's manifest package:
// https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/blob/b731a62175207a3d8343d318e72ddc13896bcb3f/pkg/patterns/declarative/pkg/manifest/objects.go#L96
func nestedFieldNoCopy(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj

	for i, field := range fields {
		if m, ok := val.(map[string]interface{}); ok {
			val, ok = m[field]
			if !ok {
				return nil, false, nil
			}
		} else {
			return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected map[string]interface{}", strings.Join(fields[:i+1], "."), val, val)
		}
	}
	return val, true, nil
}

// getObjectStatus returns the status of a given object, if any.
func getObjectStatus(obj map[string]interface{}) (map[string]interface{}, error) {
	status, found, err := nestedFieldNoCopy(obj, "status")
	if err != nil {
		return nil, fmt.Errorf("error reading object status: %v", err)
	}

	if !found {
		return nil, fmt.Errorf("object status not found")
	}

	objStatus, ok := status.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("status was not of type map[string]interface{}")
	}

	return objStatus, nil
}

// statusChanged gets the status of the given objects and compares them. It
// returns true if there's a change in the object status.
func (c *CompositeReconciler) statusChanged(oldo runtime.Object, newo runtime.Object) (bool, error) {
	// Get the old status value.
	ou, err := c.getUnstructuredObject(oldo)
	if err != nil {
		return false, fmt.Errorf("failed to convert old Object to Unstructured: %v", err)
	}
	oldStatus, err := getObjectStatus(ou.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get old Object status: %v", err)
	}

	// Get the new status value.
	nu, err := c.getUnstructuredObject(newo)
	if err != nil {
		return false, fmt.Errorf("failed to convert new Object to Unstructured: %v", err)
	}
	newStatus, err := getObjectStatus(nu.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get new Object status: %v", err)
	}

	// Compare the status values.
	if !reflect.DeepEqual(oldStatus, newStatus) {
		return true, nil
	}
	return false, nil
}

// getUnstructuredObject converts the given Object into Unstructured type.
func (c *CompositeReconciler) getUnstructuredObject(obj runtime.Object) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if err := c.scheme.Convert(obj, u, nil); err != nil {
		return nil, fmt.Errorf("failed to convert Object to Unstructured: %v", err)
	}
	return u, nil
}

// isUninitialized checks if an object is uninitialized by checking if there's
// any status condition.
func (c *CompositeReconciler) isInitialized(obj runtime.Object) (bool, error) {
	u, err := c.getUnstructuredObject(obj)
	if err != nil {
		return false, fmt.Errorf("failed to convert Object to Unstructured: %v", err)
	}
	status, err := getObjectStatus(u.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get Object status: %v", err)
	}

	_, ok := status["conditions"].([]interface{})
	if ok {
		return true, nil
	}
	return false, nil
}
