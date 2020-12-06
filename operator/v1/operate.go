package v1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// Operator is the operator interface that can be implemented by an operator to
// be used in a controller control loop.
type Operator interface {
	// IsSuspended tells if an operator is suspended and should not run any
	// operation.
	IsSuspended() bool

	// Ensure runs all the operands' Ensure method in order defined by their
	// dependencies.
	Ensure() (result ctrl.Result, err error)

	// Cleanup runs all the operands' Delete method in reverse order defined by
	// their dependencies.
	Cleanup() (result ctrl.Result, err error)
}

// defaultIsSuspended always returns false.
func defaultIsSuspended() bool {
	return false
}
