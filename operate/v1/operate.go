package v1

import (
	"sync"

	multierror "github.com/hashicorp/go-multierror"
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operate/v1/dag"
	"github.com/darkowlzz/composite-reconciler/operate/v1/operand"
)

// Operator is the operator interface that can be implemented by an operator to
// use the composite operator lifecycle.
type Operator interface {
	// IsSuspended tells if an operator is suspended and should not run any
	// operation.
	IsSuspended() bool

	// Ensure runs all the operands' ensure method in order defined by their
	// dependencies.
	Ensure() (result ctrl.Result, event eventv1.ReconcilerEvent, err error)

	// Cleanup runs all the operands' delete method in reverse order defined by
	// their dependencies.
	Cleanup() (result ctrl.Result, event eventv1.ReconcilerEvent, err error)
}

// CompositeOperator contains all the operators and the relationship between
// them.
type CompositeOperator struct {
	Operands    []*operand.Operand
	DAG         *dag.OperandDAG
	isSuspended func() bool
	order       operand.OperandOrder
	// TODO: Add a k8s client to be used by the operands.
}

// defaultIsSuspended always returns false. It is used with in a new
// CompositeOperator by default.
func defaultIsSuspended() bool {
	return false
}

// NewCompositeOperator creates a composite operator with a list of operands.
// It evaluates the operands for valid, creating a relationship model between
// the model and returns a CompositeOperator. The relationship model between
// the operands is made using a directed acyclic graph(DAG). If the
// relationships do not form a proper DAG, an error is returned.
func NewCompositeOperator(operands ...*operand.Operand) (*CompositeOperator, error) {
	od, err := dag.NewOperandDAG(operands)
	if err != nil {
		return nil, err
	}
	order, err := od.Order()
	if err != nil {
		return nil, err
	}
	return &CompositeOperator{
		Operands:    operands,
		DAG:         od,
		isSuspended: defaultIsSuspended,
		order:       order,
	}, nil
}

// Order returns the order at which the operands depends on each other. This
// can be used for creation and deletion of all the resource, if used in
// reverse order.
func (co *CompositeOperator) Order() operand.OperandOrder {
	return co.order
}

// SetIsSuspended can be used to set or override the operator suspension check.
func (co *CompositeOperator) SetIsSuspended(checkSuspend func() bool) {
	co.isSuspended = checkSuspend
}

// IsSuspend implements the Operator interface. It checks if the operator can
// run or if it's suspended and shouldn't run.
func (co *CompositeOperator) IsSuspended() bool {
	return co.isSuspended()
}

// Ensure implements the Operator interface. It runs all the operands, in the
// order of their dependencies, to ensure all the operations the individual
// operands perform.
func (co *CompositeOperator) Ensure() (result ctrl.Result, event eventv1.ReconcilerEvent, rerr error) {
	// TODO: Get the right event from the operand's target objects.

	for _, ops := range co.order {
		// Run the operands in the same order concurrently.
		var wg sync.WaitGroup
		var e chan error = make(chan error, len(ops))

		wg.Add(len(ops))
		for _, op := range ops {
			go operateWithWaitGroup(&wg, e, op.Ensure)
		}
		wg.Wait()
		close(e)

		// TODO: Use a multierror package to better collect the multiple
		// errors.
		gotErrs := false
		for err := range e {
			gotErrs = true
			rerr = multierror.Append(rerr, err)
		}

		// If an error is encountered, set requeue in the result and break out
		// of the operation execution loop.
		if gotErrs {
			result = ctrl.Result{Requeue: true}
			break
		}
	}

	return
}

// Cleanup implements the Operator interface.
func (co *CompositeOperator) Cleanup() (result ctrl.Result, event eventv1.ReconcilerEvent, rerr error) {
	return
}

func operateWithWaitGroup(wg *sync.WaitGroup, e chan error, f func() error) {
	defer wg.Done()

	if err := f(); err != nil {
		e <- err
	}
}
