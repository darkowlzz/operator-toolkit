package v1

import (
	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// DefaultIsUninitialized performs uninitialized check on an object based on
// the status conditions.
func DefaultIsUninitialized(conditions []conditionsv1.Condition) bool {
	return conditions == nil
}

// DeletionCheck checks if the main resource has been marked for deletion and
// runs cleanup. If the resource is not marked for deletion, it ensures that
// the metadata contains finalizer.
func DeletionCheck(c Controller, finalizerName string) (result ctrl.Result, event []eventv1.ReconcilerEvent, rerr error) {
	metadata := c.GetObjectMetadata()
	if metadata.DeletionTimestamp.IsZero() {
		if !contains(metadata.Finalizers, finalizerName) {
			if err := c.AddFinalizer(finalizerName); err != nil {
				// Return for a retry and return the error.
				result = ctrl.Result{Requeue: true}
				rerr = err
				return
			}
		}
	} else {
		if contains(metadata.Finalizers, finalizerName) {
			return c.Cleanup()
		}
	}
	return nil
}

// HasFinalizer returns true if obj has the named finalizer.
func HasFinalizer(obj metav1.Object, name string) bool {
	for _, item := range obj.GetFinalizers() {
		if item == name {
			return true
		}
	}
	return false
}

// AddFinalizer adds the named finalizer to obj, if it isn't already present.
func AddFinalizer(obj metav1.Object, name string) {
	if HasFinalizer(obj, name) {
		// It's already present, so there's nothing to do.
		return
	}
	obj.SetFinalizers(append(obj.GetFinalizers(), name))
}

// RemoveFinalizer removes the named finalizer from obj, if it's present.
func RemoveFinalizer(obj metav1.Object, name string) {
	finalizers := obj.GetFinalizers()
	for i, item := range finalizers {
		if item == name {
			obj.SetFinalizers(append(finalizers[:i], finalizers[i+1:]...))
			return
		}
	}
	// We never found it, so it's already gone and there's nothing to do.
}

func contains(slice []string, s string) bool {
	for _, element := range slice {
		if element == s {
			return true
		}
	}
	return false
}
