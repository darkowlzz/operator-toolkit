package v1

//go:generate mockgen -destination=mocks/mock_reconciler.go -package=mocks github.com/darkowlzz/operator-toolkit/controller/composite/v1 Controller

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Controller is the controller interface that must be implemented by a
// composite controller. It provides methods required for reconciling a
// composite controller.
type Controller interface {
	// Apply default values to the primary object spec. Use this in case a
	// defaulting webhook has not been deployed.
	Default(context.Context, client.Object)

	// Validate validates the primary object spec before it's created. It
	// ensures that all required fields are present and valid. Use this in case
	// a validating webhook has not been deployed.
	Validate(context.Context, client.Object) error

	// Initialize sets the provided initialization condition on the object
	// status. The object status need not be updated using a k8s client, only
	// the status value should be set. The controller handles updating the
	// object status in the API. Any additional status, other than the initial
	// conditions can be set here.
	Initialize(context.Context, client.Object, metav1.Condition) error

	// UpdateStatus queries the status of the child objects and based on them,
	// sets the status of the primary object instance. It need not save the
	// updated object in the API. API update is done by the controller after
	// collecting and comparing the new status with the old status. This is the
	// only place for updating the object. All other updates to the object are
	// discarded.
	UpdateStatus(context.Context, client.Object) error

	// Operate runs the core operation of the controller that ensures that
	// the child objects or the other objects and configurations in the
	// environment are in the desired state. It should be able to update any
	// existing resources or create one, if there's a configuration drift,
	// based on the type of objects.
	// The returned result is the returned reconcile result.
	Operate(context.Context, client.Object) (result ctrl.Result, err error)

	// Cleanup runs the custom cleanup operation to delete or undo the changes
	// made by the controller. This can be empty for controllers that use owner
	// reference based garbage collection for cleanup. For controllers with
	// custom cleanup requirement, the cleanup logic can be defined here.
	Cleanup(context.Context, client.Object) (result ctrl.Result, err error)
}
