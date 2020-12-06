package operand

import (
	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
)

// ChangeApplyStrategy is the strategy used to apply a change using the
// operator.
// type ChangeApplyStrategy int
// TODO: Move to Operand.
type RequeueStrategy int

const (
	// AllAtOnce is a ChangeApplyStrategy that applies all the changes at once.
	// This strategy is fast and applies all the changes at once in a single
	// reconciliation.

	// Only requeue on error.
	OnError RequeueStrategy = iota

	// OneAtATime is a ChangeApplyStrategy that applies one change at a time.
	// This strategy is slow and results in reconciliation requeue for every
	// applied change.
	// OneAtATime ChangeApplyStrategy = iota

	// Always requeue result after executing.
	Always
)

// Operand defines a single operation that's part of an composite operator. It
// contains implementation details about how an action is performed, maybe for
// creating a resource, and how to reverse/undo the action, maybe for cleanup
// purposes. It also contains relationship information about the operand with
// other operands and details about checking the ready status of target
// objects.
type Operand struct {
	// Name of the operand.
	Name string

	// Objs is the metadata of the target objects an operator updates.
	// Objs []runtime.Object

	// Resources are the objects that the operand creates, updates or deletes.
	// These objects are checked for readiness based on the ReadyConditions.
	// Resources []runtime.Object

	// Requires defines the relationship between the operands of an operator.
	Requires []string

	// Ensure creates, or updates a target object with the wanted
	// configurations. It also returns an event that can be posted on the
	// parent object's event list.
	Ensure func() (eventv1.ReconcilerEvent, error)

	// Delete deletes a target object. It also returns an event that can be
	// posted on the parent object's event list.
	Delete func() (eventv1.ReconcilerEvent, error)

	// ReadyConditions are the set of conditions that indicate that the target
	// object is ready and available.
	// ReadyConditions []map[conditionsv1.ConditionType]corev1.ConditionStatus

	// Requeue is the requeue strategy for this operand.
	Requeue RequeueStrategy

	// CheckReady allows writing custom logic for checking if an object is
	// ready. This should be used when status conditions are not enough for
	// knowing the readiness.
	CheckReady func() (bool, error)
}

type OperandOption func(*Operand)

func WithRequires(requires []string) OperandOption {
	return func(o *Operand) {
		o.Requires = append(o.Requires, requires...)
	}
}

func WithEnsure(f func() (eventv1.ReconcilerEvent, error)) OperandOption {
	return func(o *Operand) {
		o.Ensure = f
	}
}

func WithDelete(f func() (eventv1.ReconcilerEvent, error)) OperandOption {
	return func(o *Operand) {
		o.Delete = f
	}
}

func WithCheckReady(f func() (bool, error)) OperandOption {
	return func(o *Operand) {
		o.CheckReady = f
	}
}

func WithRequeueStrategy(requeue RequeueStrategy) OperandOption {
	return func(o *Operand) {
		o.Requeue = requeue
	}
}

// NewOperand creates and returns a new Operand with the given name and
// options.
func NewOperand(name string, opts ...OperandOption) *Operand {
	o := &Operand{
		Name:    name,
		Requeue: OnError,
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// OperandRunCall defines a function type used to define a function that
// returns an operand execute call. This is used for passing the operand
// execute function (Ensure or Delete) in a generic way.
type OperandRunCall func(op *Operand) func() (eventv1.ReconcilerEvent, error)

// callEnsure is an OperandRunCall type function that calls the Ensure function
// of a given operand.
func CallEnsure(op *Operand) func() (eventv1.ReconcilerEvent, error) {
	// TODO: Perform the readiness check before returning. This will ensure
	// that the operands that depend on this are executed only after this
	// operation is successful.
	return op.Ensure
}

// callCleanup is an OperandRunCall type function that calls the Cleanup
// function of a given operand.
func CallCleanup(op *Operand) func() (eventv1.ReconcilerEvent, error) {
	return op.Delete
}
