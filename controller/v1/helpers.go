package v1

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
)

// DefaultIsUninitialized performs uninitialized check on an object based on
// the status conditions.
func DefaultIsUninitialized(conditions []conditionsv1.Condition) bool {
	if conditions == nil {
		return true
	}
	return false
}

// DeletionCheck checks if the main resource has been marked for deletion and
// runs cleanup. If the resource is not marked for deletion, it ensures that
// the metadata contains finalizer.
func DeletionCheck(c Controller, finalizerName string) error {
	metadata := c.GetObjectMetadata()
	if metadata.DeletionTimestamp.IsZero() {
		if !contains(metadata.Finalizers, finalizerName) {
			if err := c.AddFinalizer(finalizerName); err != nil {
				return err
			}
		}
	} else {
		if contains(metadata.Finalizers, finalizerName) {
			return c.Cleanup()
		}
	}
	return nil
}

func contains(slice []string, s string) bool {
	for _, element := range slice {
		if element == s {
			return true
		}
	}
	return false
}
