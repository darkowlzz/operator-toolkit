package executor

import (
	"fmt"
	"sync"

	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

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
) (result ctrl.Result, rerr error) {
	// Iterate through the order steps and run the operands in the steps as per
	// the execution strategy.
	for _, ops := range order {
		// Error in the current execution step.
		var execErr error

		switch exe.execStrategy {
		case Serial:
			// Run the operands serially.
			execErr = exe.SerialExec(ops, call)
		case Parallel:
			// Run the operands concurrently.
			execErr = exe.ConcurrentExec(ops, call)
		default:
			rerr = fmt.Errorf("unknown operands execution strategy: %v", exe.execStrategy)
			return
		}

		if execErr != nil {
			result = ctrl.Result{Requeue: true}
			// TODO: When the failure toleration option is added, aggregate all
			// the errors by appending to the main rerr and continue iterating.
			rerr = execErr
			break
		}
	}

	return
}

// SerialExec runs the given set of operands serially with the given call
// function.
func (exe *Executor) SerialExec(ops []*operand.Operand, call operand.OperandRunCall) (rerr error) {
	for _, op := range ops {
		// Call the run call function. Since this is serial execution, return
		// if an error occurs.
		event, err := call(op)()
		if err != nil {
			rerr = multierror.Append(rerr, err)
			return
		}
		if event != nil {
			event.Record(exe.recorder)
		}
	}

	return
}

// ConcurrentExec runs the operands concurrently, collecting the events and
// errors from the operand executions and returns them.
func (exe *Executor) ConcurrentExec(ops []*operand.Operand, call operand.OperandRunCall) (rerr error) {
	// Wait group to synchronize the go routines.
	var wg sync.WaitGroup

	totalOperands := len(ops)

	// Error buffered channel to collect all the errors from the go routines.
	var errChan chan error = make(chan error, totalOperands)

	wg.Add(totalOperands)
	for _, op := range ops {
		go exe.operateWithWaitGroup(&wg, errChan, call(op))
	}
	wg.Wait()
	close(errChan)

	// Check if any errors were encountere.
	for err := range errChan {
		rerr = multierror.Append(rerr, err)
	}

	return
}

// operateWithWaitGroup runs the given function f and calls done on the wait
// group at the end. This is a goroutine function used for running the operands
// concurrently. The events and errors from the execution are communicated via
// the respective channels.
func (exe *Executor) operateWithWaitGroup(wg *sync.WaitGroup, errChan chan error, f func() (eventv1.ReconcilerEvent, error)) {
	defer wg.Done()

	event, err := f()
	if err != nil {
		errChan <- err
	}

	if event != nil {
		event.Record(exe.recorder)
	}
}
