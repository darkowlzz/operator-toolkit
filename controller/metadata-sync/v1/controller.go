package v1

//go:generate mockgen -destination=mocks/mock_controller.go -package=mocks github.com/darkowlzz/operator-toolkit/controller/metadata-sync/v1 Controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
)

// Controller is a metadata sync controller interface that must be implemented
// by an external sync controller. It provides methods required for reconciling
// and syncing a k8s object's metadata to external objects.
type Controller interface {
	syncv1.Controller

	// List lists all the objects in the external system. It returns a list of
	// NamespacedName of the external objects. This is used for garbage
	// collection and can be expensive. The garbage collector is run in a
	// separate goroutine periodically, not affecting the main reconciliation
	// control-loop. If the external system has no concept of namespace, the
	// namespace value can be empty.
	List(context.Context) ([]types.NamespacedName, error)

	// Diff receives a list of k8s objects and should return a subset of the
	// same list to indicate the objects that are not in sync with the external
	// system.  If resync has been enabled, then each object in the list will be
	// re-applied with Ensure().
	Diff(context.Context, []client.Object) ([]client.Object, error)
}
