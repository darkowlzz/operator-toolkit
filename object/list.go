package object

import (
	"errors"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespacedNames converts an ObjectList to a slice of NamespacedNames.
func NamespacedNames(instances client.ObjectList) ([]types.NamespacedName, error) {
	result := []types.NamespacedName{}

	items, err := apimeta.ExtractList(instances)
	if err != nil {
		return nil, fmt.Errorf("failed to extract objects from object list %v: %w", instances, err)
	}
	for _, item := range items {
		// Get meta object from the item and extract namespace/name info.
		obj, err := apimeta.Accessor(item)
		if err != nil {
			return nil, fmt.Errorf("failed to get accessor for %v: %w", item, err)
		}
		result = append(result, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()})
	}

	return result, nil
}

// NamespacedNamesDiff takes two slices of NamespacedNames and returns a
// slice of elements from the list a that are not present in list b.
func NamespacedNamesDiff(a, b []types.NamespacedName) []types.NamespacedName {
	result := []types.NamespacedName{}

	for _, aa := range a {
		found := false
		for _, bb := range b {
			if aa == bb {
				found = true
				break
			}
		}
		if !found {
			result = append(result, aa)
		}
	}

	return result
}

// ClientObjects converts a slice of runtime objects to a list of client.Object.
func ClientObjects(scheme *runtime.Scheme, objs []runtime.Object) ([]client.Object, error) {
	result := []client.Object{}
	for _, o := range objs {
		u := &unstructured.Unstructured{}
		if err := scheme.Convert(o, u, nil); err != nil {
			return nil, err
		}

		if u.IsList() {
			return nil, errors.New("object is a list")
		}
		result = append(result, u)
	}

	return result, nil
}
