package v1

//go:generate mockgen -destination=mocks/mock_controller.go -package=mocks github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1 Controller

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action"
)

// Controller is an interface that must be implemented by a stateless-action
// controller.
type Controller interface {
	// GetObject fetches an instance of an object being reconciled. It can be
	// fetched from any client or cache. To keep it generic, it returns an
	// empty interface.
	GetObject(context.Context, client.ObjectKey) (interface{}, error)

	// RequireAction evaluates the target object to find out of the action must
	// be executed on it. This is where all the checks must be performed to
	// decide if an action is needed or not.
	RequireAction(context.Context, interface{}) (bool, error)

	// BuildActionManager builds an action manager that manages the actions to
	// be executed.
	BuildActionManager(interface{}) (action.Manager, error)
}
