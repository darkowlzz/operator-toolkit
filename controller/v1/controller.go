package v1

import (
	"context"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Controller is the controller interface that must be implemented by a
// composite controller. It provides methods required for reconciling a
// composite controller.
type Controller interface {
	// InitReconciler initializes the reconciler with a given request and
	// context. Logger setup and initialization of an instance of the primary
	// object can also be done here.
	InitReconcile(context.Context, ctrl.Request)

	// FetchInstance queries the latest version of the primary object the
	// controller is responsible for. If the object is not found, a "not found"
	// error is expected in return.
	FetchInstance() error

	// IsUninitialized checks the primary object instance fetched in
	// FetchInstance() and determines if the object has not been initialized.
	// This is usually done by checking the status of the object. This is
	// checked before running Initialize().
	IsUninitialized() bool

	// Initialize runs the initialization operation for the primary object and
	// updates the object status with the given condition to mark the object as
	// initialized. Depending on the controller, initialize can internally call
	// RunOperation() or have an entirely different set of operations to
	// initialize the primary object.
	Initialize(conditionsv1.Condition) error

	// EmitEvent emits event on the primary object.
	EmitEvent(eventType, reason, message string)

	// RunOperation runs the core operation of the controller that ensures that
	// the child objects or the other objects and configurations in the
	// environment are in the desired state. It should be able to update any
	// existing resources if there's a configuration drift appropriately, based
	// on the type of objects. It should also be able to perform the operations
	// that took place ini Initialize() to create resources, in case they don't
	// exist anymore but the desired state expects them to exist.
	RunOperation() error

	// StatusUpdate checks the child objects or the other objects and
	// configurations in the environment to determine the status of the primary
	// object and updates its status. This usually sets the status conditions
	// and phase, depending on the situation.
	StatusUpdate() error

	// CheckForDrift looks for any drift in the configuration of child objects
	// or the environment compared to the desired state expressed in the
	// primary object's configuration. This is usually run before
	// RunOperation().
	CheckForDrift() (bool, error)

	// GetObjectMetadata returns the resource metadata of the primary object.
	// This is usually used to check if a resource is marked for deletion.
	GetObjectMetadata() metav1.ObjectMeta

	// AddFinalizer adds a finalizer to the primary object's metadata. This is
	// used when the controller has a custom cleanup operation, not based on
	// garbage collection using owner reference of the resources.
	AddFinalizer(string) error

	// Cleanup runs the custom cleanup operation to delete or undo the changes
	// made by the controller. This can be empty for controllers that use owner
	// reference based garbage collection for cleanup. For controllers with
	// custom cleanup requirement, the cleanup logic can be defined here.
	Cleanup() error

	// UpdateConditions updates the status condition of the primary object with
	// the given conditions.
	UpdateConditions([]conditionsv1.Condition) error
}
