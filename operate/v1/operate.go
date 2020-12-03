package v1

import (
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
)

// Operator is the operator interface that can be implemented by an operator to
// be used with the composite operator lifecycle.
type Operator interface {
	// IsSuspended tells if an operator is suspended and should not run any
	// operation.
	IsSuspended() bool

	// Ensure runs all the operands' Ensure method in order defined by their
	// dependencies.
	Ensure() (result ctrl.Result, events []eventv1.ReconcilerEvent, err error)

	// Cleanup runs all the operands' Delete method in reverse order defined by
	// their dependencies.
	Cleanup() (result ctrl.Result, events []eventv1.ReconcilerEvent, err error)
}

// defaultIsSuspended always returns false.
func defaultIsSuspended() bool {
	return false
}
