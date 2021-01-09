package v1

//go:generate mockgen -destination=mocks/mock_controller.go -package=mocks github.com/darkowlzz/operator-toolkit/controller/sync/v1 Controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Controller is an external object sync controller interface that must be
// implemented by an external object sync controller. It provides methods
// required for reconciling and syncing external objects.
type Controller interface {
	// Ensure receives a k8s object and calls an external system's API to
	// ensure an associated object exists in the external system. Based on the
	// use case, it may also update the object in the external system to have
	// the same the desired configuration.
	Ensure(context.Context, client.Object) error

	// Delete receives a k8s object that's deleted and calls an external
	// system's API to delete the associated object in the external system.
	Delete(context.Context, client.Object) error

	// List lists all the objects in the external system. It returns a list of
	// NamespacedName of the external objects. This is used for garbage
	// collection and can be expensive. The garbage collector is run in a
	// separate goroutine periodically, not affecting the main reconciliation
	// control-loop. If the external system has no concept of namespace, the
	// namespace value can be empty.
	List(context.Context) ([]types.NamespacedName, error)
}
