package operand

import (
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Operand struct {
	// Name of the operand.
	Name string

	// Objs is the metadata of the target objects an operator updates.
	// Objs []runtime.Object

	// Resources are the objects that the operand creates, updates or deletes.
	// These objects are checked for readiness based on the ReadyConditions.
	Resources []runtime.Object

	// DependsOn defines the relationship between the operands of an
	// operator. This is used to create an order of the operation
	// based on the operands it depends on.
	// DependsOn []runtime.Object

	// DependsOn defines the relationship between the operands of an operator.
	// "Requires"
	DependsOn []string

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
