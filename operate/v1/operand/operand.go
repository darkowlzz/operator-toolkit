package operand

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	Resources []runtime.Object

	// Requires defines the relationship between the operands of an operator.
	Requires []string

	// Ensure creates, or updates a target object with the wanted
	// configurations.
	Ensure func() error

	// Delete deletes a target object.
	Delete func() error

	// ReadyConditions are the set of conditions that indicate that the target
	// object is ready and available.
	ReadyConditions []map[conditionsv1.ConditionType]corev1.ConditionStatus

	// CheckReady allows writing custom logic for checking if an object is
	// ready. This should be used when status conditions are not enough for
	// knowing the readiness.
	CheckReady func() (bool, error)
}

func (c *Operand) Ready() (bool, error) {
	// Fetch dependent objects and check ReadyConditions or call c.CheckReady().
	ready := false
	return ready, nil
}

// OperandRunCall defines a function type used to define a function that
// returns an operand execute call. This is used for passing the operand
// execute function (Ensure or Delete) in a generic way.
type OperandRunCall func(op *Operand) func() error

// callEnsure is an OperandRunCall type function that calls the Ensure function
// of a given operand.
func CallEnsure(op *Operand) func() error {
	return op.Ensure
}

// callCleanup is an OperandRunCall type function that calls the Cleanup
// function of a given operand.
func CallCleanup(op *Operand) func() error {
	return op.Delete
}
