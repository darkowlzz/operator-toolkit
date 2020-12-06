package v1

import (
	"fmt"

	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/darkowlzz/composite-reconciler/operator/v1/dag"
	"github.com/darkowlzz/composite-reconciler/operator/v1/executor"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
)

// CompositeOperator contains all the operands and the relationship between
// them. It implements the Operator interface.
type CompositeOperator struct {
	Operands          []*operand.Operand
	DAG               *dag.OperandDAG
	isSuspended       func() bool
	order             operand.OperandOrder
	executionStrategy executor.ExecutionStrategy
	recorder          record.EventRecorder
	executor          *executor.Executor

	// TODO: Add a k8s client to be used by the operands.
}

// CompositeOperatorOption is used to configure CompositeOperator.
type CompositeOperatorOption func(*CompositeOperator)

// WithExecutionStrategy sets the execution strategy of a CompositeOperator.
func WithExecutionStrategy(strategy executor.ExecutionStrategy) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.executionStrategy = strategy
	}
}

// WithOperands sets the set of operands in a CompositeOperator.
func WithOperands(operands ...*operand.Operand) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.Operands = operands
	}
}

// SetIsSuspended can be used to set the operator suspension check.
func WithSuspensionCheck(f func() bool) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.isSuspended = f
	}
}

// WithEventRecorder sets the EventRecorder of a CompositeOperator.
func WithEventRecorder(recorder record.EventRecorder) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.recorder = recorder
	}
}

// NewCompositeOperator creates a composite operator with a list of operands.
// It evaluates the operands for valid, creating a relationship model between
// the model and returns a CompositeOperator. The relationship model between
// the operands is made using a directed acyclic graph(DAG). If the
// relationships do not form a proper DAG, an error is returned.
func NewCompositeOperator(opts ...CompositeOperatorOption) (*CompositeOperator, error) {
	// Set all the default configurations.
	c := &CompositeOperator{
		isSuspended:       defaultIsSuspended,
		executionStrategy: executor.Parallel,
	}

	// Loop through each option.
	for _, opt := range opts {
		opt(c)
	}

	// Ensure a recorder is provided.
	if c.recorder == nil {
		return nil, fmt.Errorf("EventRecorder must be provided to the CompositeOperator")
	}

	// Initialize the operator DAG.
	od, err := dag.NewOperandDAG(c.Operands)
	if err != nil {
		return nil, err
	}
	c.DAG = od

	// Compute traversal order in the DAG.
	order, err := od.Order()
	if err != nil {
		return nil, err
	}
	c.order = order

	// Create an executor.
	c.executor = executor.NewExecutor(c.executionStrategy, c.recorder)

	return c, nil
}

// Order returns the order at which the operands depends on each other. This
// can be used for creation and deletion of all the resource, if used in
// reverse order.
func (co *CompositeOperator) Order() operand.OperandOrder {
	return co.order
}

// IsSuspend implements the Operator interface. It checks if the operator can
// run or if it's suspended and shouldn't run.
func (co *CompositeOperator) IsSuspended() bool {
	return co.isSuspended()
}

// Ensure implements the Operator interface. It runs all the operands, in the
// order of their dependencies, to ensure all the operations the individual
// operands perform.
func (co *CompositeOperator) Ensure() (result ctrl.Result, rerr error) {
	return co.executor.ExecuteOperands(co.order, operand.CallEnsure)
}

// Cleanup implements the Operator interface.
func (co *CompositeOperator) Cleanup() (result ctrl.Result, rerr error) {
	return co.executor.ExecuteOperands(co.order.Reverse(), operand.CallCleanup)
}
