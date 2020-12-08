package operand

//go:generate mockgen -destination=../../../mocks/mock_operand.go -package=mocks github.com/darkowlzz/composite-reconciler/operator/v1/operand Operand

import (
	"fmt"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
)

// RequeueStrategy defines the requeue strategy of an operand.
type RequeueStrategy int

const (
	// RequeueOnError is used for requeue on error only.
	RequeueOnError RequeueStrategy = iota

	// RequeueAlways is used to requeue result after every applied change.
	RequeueAlways
)

// Operand defines a single operation that's part of a composite operator. It
// contains implementation details about how an action is performed, maybe for
// creating a resource, and how to reverse/undo the action, maybe for cleanup
// purposes. It also contains relationship information about the operand with
// other operands and details about checking the ready status of target
// objects.
type Operand interface {
	// Name of the operand.
	Name() string

	// Requires defines the relationship between the operands of an operator.
	Requires() []string

	// Ensure creates, or updates a target object with the wanted
	// configurations. It also returns an event that can be posted on the
	// parent object's event list.
	Ensure() (eventv1.ReconcilerEvent, error)

	// Delete deletes a target object. It also returns an event that can be
	// posted on the parent object's event list.
	Delete() (eventv1.ReconcilerEvent, error)

	// Requeue is the requeue strategy for this operand.
	RequeueStrategy() RequeueStrategy

	// ReadyCheck allows writing custom logic for checking if an object is
	// ready.
	ReadyCheck() (bool, error)
}

// OperandRunCall defines a function type used to define a function that
// returns an operand execute call. This is used for passing the operand
// execute function (Ensure or Delete) in a generic way.
type OperandRunCall func(op Operand) func() (eventv1.ReconcilerEvent, error)

// callEnsure is an OperandRunCall type function that calls the Ensure function
// and the ReadyCheck of a given operand. The Ensure function ensures that the
// desired change is applied to the world and ReadyCheck helps proceed only
// when the desired state of the world is reached. This helps run dependent
// operands only after a successful operand execution.
func CallEnsure(op Operand) func() (eventv1.ReconcilerEvent, error) {
	return func() (event eventv1.ReconcilerEvent, err error) {
		event, err = op.Ensure()
		if err != nil {
			return
		}

		ready, readyErr := op.ReadyCheck()
		if readyErr != nil {
			err = readyErr
			return
		}

		if !ready {
			err = fmt.Errorf("operand %q readiness check failed: not in the desired state yet", op.Name())
		}

		return
	}
}

// callCleanup is an OperandRunCall type function that calls the Cleanup
// function of a given operand.
func CallCleanup(op Operand) func() (eventv1.ReconcilerEvent, error) {
	return op.Delete
}
