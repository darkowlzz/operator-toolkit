package operand

//go:generate mockgen -destination=mocks/mock_operand.go -package=mocks github.com/darkowlzz/operator-toolkit/operator/v1/operand Operand

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
)

// RequeueStrategy defines the requeue strategy of an operand.
type RequeueStrategy int

const (
	// RequeueOnError is used for requeue on error only.
	RequeueOnError RequeueStrategy = iota

	// RequeueAlways is used to requeue result after every applied change.
	RequeueAlways
)

// ErrNotReady is returned by operand when the ready check fails.
var ErrNotReady = errors.New("operand not ready")

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
	Ensure(context.Context, client.Object, metav1.OwnerReference) (eventv1.ReconcilerEvent, error)

	// Delete deletes a target object. It also returns an event that can be
	// posted on the parent object's event list.
	Delete(context.Context, client.Object) (eventv1.ReconcilerEvent, error)

	// Requeue is the requeue strategy for this operand.
	RequeueStrategy() RequeueStrategy

	// ReadyCheck allows writing custom logic for checking if an object is
	// ready.
	ReadyCheck(context.Context, client.Object) (bool, error)

	// PostReady allows performing actions once the target object of the
	// operand is ready.
	PostReady(context.Context, client.Object) error
}

// OperandRunCall defines a function type used to define a function that
// returns an operand execute call. This is used for passing the operand
// execute function (Ensure or Delete) in a generic way.
type OperandRunCall func(op Operand) func(context.Context, client.Object, metav1.OwnerReference) (eventv1.ReconcilerEvent, error)

// CallEnsure is an OperandRunCall type function that calls the Ensure function
// and the ReadyCheck of a given operand. The Ensure function ensures that the
// desired change is applied to the world and ReadyCheck helps proceed only
// when the desired state of the world is reached. This helps run dependent
// operands only after a successful operand execution.
func CallEnsure(op Operand) func(context.Context, client.Object, metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	return func(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
		event, err := op.Ensure(ctx, obj, ownerRef)
		if err != nil {
			return nil, err
		}

		ready, readyErr := op.ReadyCheck(ctx, obj)
		if readyErr != nil {
			return nil, readyErr
		}

		if !ready {
			return nil, fmt.Errorf("operand %q readiness check failed: not in the desired state yet: %w", op.Name(), ErrNotReady)
		}

		if err := op.PostReady(ctx, obj); err != nil {
			return nil, err
		}

		return event, nil
	}
}

// CallCleanup is an OperandRunCall type function that calls the Cleanup
// function of a given operand.
func CallCleanup(op Operand) func(context.Context, client.Object, metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	// Wrap Delete with OperandRunCall, ignoring the arguments that aren't
	// required, to have the ability to call both Ensure and Delete with
	// OperandRunCall.
	return func(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (event eventv1.ReconcilerEvent, err error) {
		event, err = op.Delete(ctx, obj)
		return
	}
}
