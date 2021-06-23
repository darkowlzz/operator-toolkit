package v1

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/operator/v1/dag"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"github.com/darkowlzz/operator-toolkit/telemetry"
	"github.com/go-logr/logr"
)

// defaultRetryPeriod is used for waiting before a retry.
const defaultRetryPeriod = 5 * time.Second

// CompositeOperator contains all the operands and the relationship between
// them. It implements the Operator interface.
type CompositeOperator struct {
	Operands          []operand.Operand
	DAG               *dag.OperandDAG
	isSuspended       func(context.Context, client.Object) bool
	order             operand.OperandOrder
	executionStrategy executor.ExecutionStrategy
	recorder          record.EventRecorder
	executor          *executor.Executor
	inst              *telemetry.Instrumentation
	retryPeriod       time.Duration
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
func WithOperands(operands ...operand.Operand) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.Operands = operands
	}
}

// SetIsSuspended can be used to set the operator suspension check.
func WithSuspensionCheck(f func(context.Context, client.Object) bool) CompositeOperatorOption {
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

// WithRetryPeriod sets the wait period of the operator before performing a
// retry in the event of a failure.
func WithRetryPeriod(duration time.Duration) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.retryPeriod = duration
	}
}

// WithInstrumentation configures the instrumentation of the CompositeOperator.
func WithInstrumentation(tp trace.TracerProvider, mp metric.MeterProvider, log logr.Logger) CompositeOperatorOption {
	return func(c *CompositeOperator) {
		c.inst = telemetry.NewInstrumentationWithProviders(instrumentationName, tp, mp, log)
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
		retryPeriod:       defaultRetryPeriod,
	}

	// Loop through each option.
	for _, opt := range opts {
		opt(c)
	}

	// Ensure a recorder is provided.
	if c.recorder == nil {
		return nil, fmt.Errorf("EventRecorder must be provided to the CompositeOperator")
	}

	// If instrumentation is nil, create a new instrumentation with default
	// providers.
	if c.inst == nil {
		WithInstrumentation(nil, nil, nil)(c)
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
func (co *CompositeOperator) IsSuspended(ctx context.Context, obj client.Object) bool {
	ctx, span, _, _ := co.inst.Start(ctx, "IsSuspended")
	defer span.End()

	return co.isSuspended(ctx, obj)
}

// Ensure implements the Operator interface. It runs all the operands, in the
// order of their dependencies, to ensure all the operations the individual
// operands perform.
func (co *CompositeOperator) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (ctrl.Result, error) {
	ctx, span, _, log := co.inst.Start(ctx, "Ensure")
	defer span.End()

	result := ctrl.Result{}

	if !co.IsSuspended(ctx, obj) {
		res, err := co.executor.ExecuteOperands(co.order, operand.CallEnsure, ctx, obj, ownerRef)
		if err != nil {
			// Not ready error shouldn't be propagated to the caller. Handle
			// the error gracefully by returning a requeue result with a wait
			// period. Set explicit requeue regardless of the returned result
			// because an error was found.
			if errors.Is(err, operand.ErrNotReady) {
				log.Info("components not ready, retrying in a few seconds...", "waitPeriod", co.retryPeriod, "failure", err)
				return ctrl.Result{Requeue: true, RequeueAfter: co.retryPeriod}, nil
			}
			return ctrl.Result{Requeue: true}, err
		}
		result = res
		span.AddEvent("CompositeOperator Ensure executed successfully")
	} else {
		span.AddEvent("CompositeOperator Ensure skipped because it's suspended")
	}
	return result, nil
}

// Cleanup implements the Operator interface.
func (co *CompositeOperator) Cleanup(ctx context.Context, obj client.Object) (result ctrl.Result, rerr error) {
	ctx, span, _, _ := co.inst.Start(ctx, "Cleanup")
	defer span.End()

	if !co.IsSuspended(ctx, obj) {
		return co.executor.ExecuteOperands(co.order.Reverse(), operand.CallCleanup, ctx, obj, metav1.OwnerReference{})
	}
	return
}
