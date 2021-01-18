package object

import (
	"fmt"
	"reflect"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OwnerReferenceFromObject creates an owner reference with the given object.
func OwnerReferenceFromObject(obj client.Object) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: obj.GetObjectKind().GroupVersionKind().GroupVersion().String(),
		Kind:       obj.GetObjectKind().GroupVersionKind().Kind,
		Name:       obj.GetName(),
		UID:        obj.GetUID(),
	}
}

// GetUnstructuredObject converts the given Object into Unstructured type.
func GetUnstructuredObject(scheme *runtime.Scheme, obj runtime.Object) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}
	if err := scheme.Convert(obj, u, nil); err != nil {
		return nil, fmt.Errorf("failed to convert Object to Unstructured: %v", err)
	}
	return u, nil
}

// IsInitialized checks if an object is initialized by checking if there's
// any status condition.
func IsInitialized(scheme *runtime.Scheme, obj runtime.Object) (bool, error) {
	u, err := GetUnstructuredObject(scheme, obj)
	if err != nil {
		return false, fmt.Errorf("failed to convert Object to Unstructured: %v", err)
	}
	status, err := GetObjectStatus(u.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get Object status: %v", err)
	}

	_, ok := status["conditions"].([]interface{})
	if ok {
		return true, nil
	}
	return false, nil
}

// GetObjectStatus returns the status of a given object, if any.
func GetObjectStatus(obj map[string]interface{}) (map[string]interface{}, error) {
	status, found, err := NestedFieldNoCopy(obj, "status")
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

// StatusChanged gets the status of the given objects and compares them. It
// returns true if there's a change in the object status.
func StatusChanged(scheme *runtime.Scheme, oldo runtime.Object, newo runtime.Object) (bool, error) {
	// Get the old status value.
	ou, err := GetUnstructuredObject(scheme, oldo)
	if err != nil {
		return false, fmt.Errorf("failed to convert old Object to Unstructured: %v", err)
	}
	oldStatus, err := GetObjectStatus(ou.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get old Object status: %v", err)
	}

	// Get the new status value.
	nu, err := GetUnstructuredObject(scheme, newo)
	if err != nil {
		return false, fmt.Errorf("failed to convert new Object to Unstructured: %v", err)
	}
	newStatus, err := GetObjectStatus(nu.Object)
	if err != nil {
		return false, fmt.Errorf("failed to get new Object status: %v", err)
	}

	// Compare the status values.
	if !reflect.DeepEqual(oldStatus, newStatus) {
		return true, nil
	}
	return false, nil
}

// NestedFieldNoCopy returns the nested field from a given Object. The second
// returned value is true if the field is found, else false.
//
// Taken from kubebuilder-declarative-pattern's manifest package:
// https://github.com/kubernetes-sigs/kubebuilder-declarative-pattern/blob/b731a62175207a3d8343d318e72ddc13896bcb3f/pkg/patterns/declarative/pkg/manifest/objects.go#L96
func NestedFieldNoCopy(obj map[string]interface{}, fields ...string) (interface{}, bool, error) {
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
