package action

//go:generate mockgen -destination=mocks/mock_manager.go -package=mocks github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action Manager

import "context"

// Manager manages the actions to be executed on objects.
type Manager interface {
	// GetObjects returns all the objects on which action should be run.
	GetObjects() ([]interface{}, error)

	// Check checks if the action is needed anymore.
	Check(context.Context, interface{}) bool

	// Run runs the action on the given object.
	Run(context.Context, interface{})

	// Defer is executed at the end of run to execute once run ends.
	Defer(context.Context, interface{})
}
