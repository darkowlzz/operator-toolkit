package executor

import (
	"fmt"
	"sync"

	multierror "github.com/hashicorp/go-multierror"
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operate/v1/operand"
)

// ExecutionStrategy is the operands execution strategy of an operator.
type ExecutionStrategy int

const (
	Parallel ExecutionStrategy = iota
	Serial
)

// ExecuteOperands executes operands in a given OperandOrder by calling a given
// OperandRunCall function on each of the operands. The OperandRunCall can be a
// call to Ensure or Delete.
func ExecuteOperands(
	order operand.OperandOrder,
	call operand.OperandRunCall,
	execStrat ExecutionStrategy,
) (result ctrl.Result, events []eventv1.ReconcilerEvent, rerr error) {
	// Iterate through the order steps and run the operands in the steps as per
	// the execution strategy.
	for _, ops := range order {
		// Store the collected events and errors in the current execution step.
		var evnts []eventv1.ReconcilerEvent
		var execErr error

		switch execStrat {
		case Serial:
			// Run the operands serially.
			evnts, execErr = SerialExec(ops, call)
		case Parallel:
			// Run the operands concurrently.
			evnts, execErr = ConcurrentExec(ops, call)
		default:
			rerr = fmt.Errorf("unknown operands execution strategy: %v", execStrat)
			return
		}

		// Append all the events received from the execution and check the
		// error.
		events = append(events, evnts...)
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
func SerialExec(ops []*operand.Operand, call operand.OperandRunCall) (events []eventv1.ReconcilerEvent, rerr error) {
	for _, op := range ops {
		// Call the run call function and collect the event and error. Since
		// this is serial execution, return if an error occurs.
		event, err := call(op)()
		events = append(events, event)
		if err != nil {
			rerr = multierror.Append(rerr, err)
			return
		}
	}

	return
}

// ConcurrentExec runs the operands concurrently, collecting the events and
// errors from the operand executions and returns them.
func ConcurrentExec(ops []*operand.Operand, call operand.OperandRunCall) (events []eventv1.ReconcilerEvent, rerr error) {
	// Wait group to synchronize the go routines.
	var wg sync.WaitGroup

	totalOperands := len(ops)

	// Event buffered channel to collect all the events from the go routines.
	var eventChan chan eventv1.ReconcilerEvent = make(chan eventv1.ReconcilerEvent, totalOperands)

	// Error buffered channel to collect all the errors from the go routines.
	var errChan chan error = make(chan error, totalOperands)

	wg.Add(totalOperands)
	for _, op := range ops {
		go operateWithWaitGroup(&wg, eventChan, errChan, call(op))
	}
	wg.Wait()
	close(eventChan)
	close(errChan)

	// Aggregate all the events.
	for event := range eventChan {
		events = append(events, event)
	}

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
func operateWithWaitGroup(wg *sync.WaitGroup, eventChan chan eventv1.ReconcilerEvent, errChan chan error, f func() (eventv1.ReconcilerEvent, error)) {
	defer wg.Done()

	event, err := f()
	if err != nil {
		errChan <- err
	}
	if event != nil {
		eventChan <- event
	}
}
