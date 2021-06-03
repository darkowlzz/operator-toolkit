package singleton

import (
	"context"
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/darkowlzz/operator-toolkit/object"
)

// GetInstanceFunc is a function type for functions that return an instance of
// a specific kind.
type GetInstanceFunc func(ctx context.Context, cli client.Client) (client.Object, error)

// GetInstance returns a GetInstanceFunc for a given kind of object.
func GetInstance(prototypeList client.ObjectList, scheme *runtime.Scheme) (GetInstanceFunc, error) {
	if prototypeList == nil {
		return nil, fmt.Errorf("ObjectList must not be nil")
	}
	if scheme == nil {
		return nil, fmt.Errorf("scheme must not be nil")
	}

	return func(ctx context.Context, cli client.Client) (client.Object, error) {
		instances := prototypeList.DeepCopyObject().(client.ObjectList)
		// No list option namespace set to list across all the namespaces.
		if listErr := cli.List(ctx, instances); listErr != nil {
			return nil, listErr
		}
		items, err := apimeta.ExtractList(instances)
		if err != nil {
			return nil, err
		}

		switch count := len(items); {
		case count > 1:
			var objectKind string
			gvk, err := apiutil.GVKForObject(items[0], scheme)
			if err != nil {
				objectKind = "unknown"
			} else {
				objectKind = gvk.Kind
			}
			return nil, newMultipleObjectsFound(objectKind, len(items))
		case count == 1:
			cobjs, err := object.ClientObjects(scheme, items)
			if err != nil {
				return nil, err
			}
			return cobjs[0], nil
		default:
			return nil, nil
		}
	}, nil
}
