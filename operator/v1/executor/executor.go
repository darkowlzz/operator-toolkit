package executor

import (
	"context"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
)

// ExecutionStrategy is the operands execution strategy of an operator.
type ExecutionStrategy int

const (
	Parallel ExecutionStrategy = iota
	Serial
)

// Executor is an operand executor. It is used to configure how the operands
// are executed. The event recorder is used to broadcast an event right after
// executing an operand.
type Executor struct {
	execStrategy ExecutionStrategy
	recorder     record.EventRecorder
}

// NewExecutor initializes and returns an Executor.
func NewExecutor(e ExecutionStrategy, r record.EventRecorder) *Executor {
	return &Executor{
		execStrategy: e,
		recorder:     r,
	}
}

// ExecuteOperands executes operands in a given OperandOrder by calling a given
// OperandRunCall function on each of the operands. The OperandRunCall can be a
// call to Ensure or Delete.
func (exe *Executor) ExecuteOperands(
	order operand.OperandOrder,
	call operand.OperandRunCall,
	ctx context.Context,
	obj client.Object,
	ownerRef metav1.OwnerReference,
) (result ctrl.Result, rerr error) {
	// Iterate through the order steps and run the operands in the steps as per
	// the execution strategy.
	for _, ops := range order {
		// Error in the current execution step.
		var execErr error

		// res is the Result of the step.
		var res *ctrl.Result

		requeueStrategy := operand.StepRequeueStrategy(ops)

		switch exe.execStrategy {
		case Serial:
			// Run the operands serially.
			res, execErr = exe.serialExec(ops, call, ctx, obj, ownerRef)
		case Parallel:
			// Run the operands concurrently.
			res, execErr = exe.concurrentExec(ops, call, ctx, obj, ownerRef)
		default:
			rerr = fmt.Errorf("unknown operands execution strategy: %v", exe.execStrategy)
			return
		}

		if execErr != nil {
			result = ctrl.Result{Requeue: true}
			rerr = execErr
			break
		}

		// If a change was made with a Result received after the execution and
		// the RequeueStrategy is RequeueAlways, set a requeued result.
		if res != nil && requeueStrategy == operand.RequeueAlways {
			result = ctrl.Result{Requeue: true}
			break
		}
	}

	return
}

// serialExec runs the given set of operands serially with the given call
// function. An event is used to know if a change was applied. When an event is
// found, a result object is returned, else nil.
func (exe *Executor) serialExec(
	ops []operand.Operand,
	call operand.OperandRunCall,
	ctx context.Context,
	obj client.Object,
	ownerRef metav1.OwnerReference,
) (result *ctrl.Result, rerr error) {
	result = nil

	for _, op := range ops {
		// Call the run call function. Since this is serial execution, return
		// if an error occurs.
		event, err := call(op)(ctx, obj, ownerRef)
		if err != nil {
			rerr = kerrors.NewAggregate([]error{rerr, err})
			return
		}
		if event != nil {
			event.Record(exe.recorder)
			result = &ctrl.Result{}
		}
	}

	return
}

// concurrentExec runs the operands concurrently, collecting the errors from
// the operand executions and returns them.
func (exe *Executor) concurrentExec(
	ops []operand.Operand,
	call operand.OperandRunCall,
	ctx context.Context,
	obj client.Object,
	ownerRef metav1.OwnerReference,
) (result *ctrl.Result, rerr error) {
	result = nil

	// Wait group to synchronize the go routines.
	var wg sync.WaitGroup

	totalOperands := len(ops)

	// resultChan is used to collect the result returned from the concurrent
	// execution of the operands.
	var resultChan chan ctrl.Result = make(chan ctrl.Result, totalOperands)

	// Error buffered channel to collect all the errors from the go routines.
	var errChan chan error = make(chan error, totalOperands)

	wg.Add(totalOperands)
	for _, op := range ops {
		go exe.operateWithWaitGroup(&wg, resultChan, errChan, call(op), ctx, obj, ownerRef)
	}
	wg.Wait()
	close(errChan)

	// Check if any errors were encountere.
	for err := range errChan {
		rerr = kerrors.NewAggregate([]error{rerr, err})
	}

	// Check the result channel, if it contains any result, return a result
	// object.
	foundResult := false
	if len(resultChan) > 0 {
		foundResult = true
	}
	if foundResult {
		result = &ctrl.Result{}
	}

	return
}

// operateWithWaitGroup runs the given function f and calls done on the wait
// group at the end. This is a goroutine function used for running the operands
// concurrently. The result from events and errors from the execution are
// communicated via the respective channels.
func (exe *Executor) operateWithWaitGroup(
	wg *sync.WaitGroup,
	resultChan chan ctrl.Result,
	errChan chan error,
	f func(context.Context, client.Object, metav1.OwnerReference) (eventv1.ReconcilerEvent, error),
	ctx context.Context,
	obj client.Object,
	ownerRef metav1.OwnerReference,
) {
	defer wg.Done()

	event, err := f(ctx, obj, ownerRef)
	if err != nil {
		errChan <- err
	}

	// Event is used to determine if a change tool place. Send a result to the
	// result channel when an event is received.
	if event != nil {
		event.Record(exe.recorder)
		resultChan <- ctrl.Result{}
	}
}
