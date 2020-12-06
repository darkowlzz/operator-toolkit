package v1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/executor"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
)

// fooCreatedEvent is a ReconcilerEvent type used for testing event
// broadcasting.
type fooCreatedEvent struct {
	Object  runtime.Object
	FooName string
}

func (c *fooCreatedEvent) Record(recorder record.EventRecorder) {
	recorder.Event(c.Object,
		eventv1.K8sEventTypeNormal,
		"FooReady",
		fmt.Sprintf("Created foo with name %s", c.FooName),
	)
}

// operandCalls keeps a count of all the calls in an operand.
type operandCalls struct {
	Ensure int
	Delete int
	Ready  int
}

// Reset resets the counters in operandCalls.
func (oc *operandCalls) Reset() {
	oc.Ensure = 0
	oc.Delete = 0
	oc.Ready = 0
}

// isSuspendedAlwaysTrue always return true.
func isSuspendedAlwaysTrue() bool {
	return true
}

func TestCompositeOperatorOrder(t *testing.T) {
	secretName := "secret"
	configmapName := "configmap"
	daemonsetName := "daemonset"
	deploymentAName := "deploymentA"
	deploymentBName := "deploymentB"

	operands := []*operand.Operand{
		&operand.Operand{
			Name: secretName,
		},
		&operand.Operand{
			Name: configmapName,
		},
		&operand.Operand{
			Name:     daemonsetName,
			Requires: []string{secretName, configmapName},
		},
		&operand.Operand{
			Name:     deploymentAName,
			Requires: []string{secretName},
		},
		&operand.Operand{
			Name:     deploymentBName,
			Requires: []string{deploymentAName},
		},
	}

	wantOrder := `[
  0: [ configmap secret ]
  1: [ daemonset deploymentA ]
  2: [ deploymentB ]
]`
	wantDeleteOrder := `[
  0: [ deploymentB ]
  1: [ daemonset deploymentA ]
  2: [ configmap secret ]
]`

	rec := record.NewFakeRecorder(1)
	co, err := NewCompositeOperator(
		WithOperands(operands...),
		WithEventRecorder(rec),
	)

	if err != nil {
		t.Errorf("unexpected error while creating a composite operator: %v", err)
	}
	od := co.Order()
	assert.Equal(t, wantOrder, od.String(), "DAG ensure order")

	or := od.Reverse()
	assert.Equal(t, wantDeleteOrder, or.String(), "DAG delete order")
}

func TestCompositeOperatorEnsure(t *testing.T) {
	var operandACalls, operandBCalls, operandCCalls operandCalls

	operandA := &operand.Operand{
		Name: "opA",
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			operandACalls.Ensure++
			// Return event only on first execution. This simulates the
			// behavior where a configuration is applied in the first loop and
			// the desired state is reached in the consecutive loop. No change.
			if operandACalls.Ensure == 1 {
				pod := &corev1.Pod{}
				evnt := &fooCreatedEvent{Object: pod, FooName: "foo foo"}
				return evnt, nil
			}
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			operandACalls.Ready++
			return true, nil
		},
	}

	// operandA with RequeueAlways requeue strategy.
	operandAReq := &operand.Operand{
		Name: "opA",
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			operandACalls.Ensure++
			if operandACalls.Ensure == 1 {
				pod := &corev1.Pod{}
				evnt := &fooCreatedEvent{Object: pod, FooName: "foo foo"}
				return evnt, nil
			}
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			operandACalls.Ready++
			return true, nil
		},
		// Setting RequeueAlways results in early return from ExecuteOperands
		// and operandC doesn't get executed. It's intended. In a real
		// reconciler, the operands will be re-executed via requeue.
		Requeue: operand.RequeueAlways,
	}

	operandB := &operand.Operand{
		Name: "opB",
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			operandBCalls.Ensure++
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			operandBCalls.Ready++
			return true, nil
		},
	}

	operandC := &operand.Operand{
		Name:     "opC",
		Requires: []string{operandA.Name},
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			operandCCalls.Ensure++
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			operandCCalls.Ready++
			return true, nil
		},
	}

	tests := []struct {
		name          string
		opts          []CompositeOperatorOption
		alwaysRequeue bool
		wantRequeue   bool // wantRequeue is the requeue value after times of Ensure.
		times         int  // times is the number of times Ensure is run.
		operandACalls operandCalls
		operandBCalls operandCalls
		operandCCalls operandCalls
	}{
		{
			name: "serial execution",
			opts: []CompositeOperatorOption{
				WithOperands(operandA, operandB, operandC),
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue:   false,
			operandACalls: operandCalls{1, 0, 1},
			operandBCalls: operandCalls{1, 0, 1},
			operandCCalls: operandCalls{1, 0, 1},
			times:         1,
		},
		{
			name: "serial execution - requeue always - 1x",
			opts: []CompositeOperatorOption{
				WithOperands(operandAReq, operandB, operandC),
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue:   true,
			operandACalls: operandCalls{1, 0, 1},
			operandBCalls: operandCalls{1, 0, 1},
			operandCCalls: operandCalls{0, 0, 0},
			times:         1,
		},
		{
			name: "serial execution - requeue always - 2x",
			opts: []CompositeOperatorOption{
				WithOperands(operandAReq, operandB, operandC),
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue:   false,
			operandACalls: operandCalls{2, 0, 2},
			operandBCalls: operandCalls{2, 0, 2},
			operandCCalls: operandCalls{1, 0, 1},
			times:         2,
		},
		{
			name: "parallel execution",
			opts: []CompositeOperatorOption{
				WithOperands(operandA, operandB, operandC),
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue:   false,
			operandACalls: operandCalls{1, 0, 1},
			operandBCalls: operandCalls{1, 0, 1},
			operandCCalls: operandCalls{1, 0, 1},
			times:         1,
		},
		{
			name: "parallel execution - requeue always - 1x",
			opts: []CompositeOperatorOption{
				WithOperands(operandAReq, operandB, operandC),
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue:   true,
			operandACalls: operandCalls{1, 0, 1},
			operandBCalls: operandCalls{1, 0, 1},
			operandCCalls: operandCalls{0, 0, 0},
			times:         1,
		},
		{
			name: "parallel execution - requeue always - 2x",
			opts: []CompositeOperatorOption{
				WithOperands(operandAReq, operandB, operandC),
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue:   false,
			operandACalls: operandCalls{2, 0, 2},
			operandBCalls: operandCalls{2, 0, 2},
			operandCCalls: operandCalls{1, 0, 1},
			times:         2,
		},
		{
			name: "suspended",
			opts: []CompositeOperatorOption{
				WithOperands(operandA, operandB, operandC),
				WithSuspensionCheck(isSuspendedAlwaysTrue),
			},
			wantRequeue: false,
			times:       1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Reset all the counters.
			operandACalls.Reset()
			operandBCalls.Reset()
			operandCCalls.Reset()

			// Append a fake recorder to the list of options.
			rec := record.NewFakeRecorder(1)
			tc.opts = append(tc.opts, WithEventRecorder(rec))

			// Create a new CompositeOperator.
			co, err := NewCompositeOperator(tc.opts...)
			assert.Nil(t, err)

			var res ctrl.Result
			var eerr error

			// Run ensure for the given number of times.
			i := 0
			for i < tc.times {
				res, eerr = co.Ensure()
				assert.Nil(t, eerr)
				i++
			}

			assert.Equal(t, tc.wantRequeue, res.Requeue, "requeue value")
			assert.Equal(t, tc.operandACalls, operandACalls, "A calls")
			assert.Equal(t, tc.operandBCalls, operandBCalls, "B calls")
			assert.Equal(t, tc.operandCCalls, operandCCalls, "C calls")
		})
	}
}

// TODO: Add TestCompositeOperatorCleanup.
