package v1

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/mocks"
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

// isSuspendedAlwaysTrue always return true.
func isSuspendedAlwaysTrue() bool {
	return true
}

func TestCompositeOperatorEnsure(t *testing.T) {
	pod := &corev1.Pod{}
	evnt := &fooCreatedEvent{Object: pod, FooName: "foo foo"}
	count := 0

	// This anonymous function is used to return Ensure result based on a
	// counter, count. It tries to simulate the behavior in which the first
	// time Ensure is run, it returns an event, but the following times, it
	// returns nil.
	conditionalEnsure := func() (eventv1.ReconcilerEvent, error) {
		count++
		if count == 1 {
			return evnt, nil
		}
		return nil, nil
	}

	tests := []struct {
		name          string
		opts          []CompositeOperatorOption
		alwaysRequeue bool
		wantRequeue   bool // wantRequeue is the requeue value after times of Ensure.
		times         int  // times is the number of times Ensure is run.
		expectations  func(a, b, c *mocks.MockOperand)
	}{
		{
			name: "serial execution",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue: false,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				opA.EXPECT().Ensure()
				opA.EXPECT().RequeueStrategy()
				opA.EXPECT().ReadyCheck().Return(true, nil)
				opB.EXPECT().Ensure()
				opB.EXPECT().RequeueStrategy()
				opB.EXPECT().ReadyCheck().Return(true, nil)
				opC.EXPECT().Ensure()
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck().Return(true, nil)
			},
			times: 1,
		},
		{
			name: "serial execution - requeue always - 1x",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue: true,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				// Reset counter used by conditionalEnsure.
				count = 0
				opA.EXPECT().Ensure().DoAndReturn(conditionalEnsure)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways)
				opA.EXPECT().ReadyCheck().Return(true, nil)
				opB.EXPECT().Ensure()
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck().Return(true, nil)
				// No execution of opC.
			},
			times: 1,
		},
		{
			name: "serial execution - requeue always - 2x",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Serial),
			},
			wantRequeue: false,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				// Reset counter used by conditionalEnsure.
				count = 0
				opA.EXPECT().Ensure().DoAndReturn(conditionalEnsure).Times(2)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways).Times(2)
				opA.EXPECT().ReadyCheck().Return(true, nil).Times(2)
				opB.EXPECT().Ensure().Times(2)
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck().Return(true, nil).Times(2)
				opC.EXPECT().Ensure()
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck().Return(true, nil)
			},
			times: 2,
		},
		{
			name: "parallel execution",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue: false,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				opA.EXPECT().Ensure()
				opA.EXPECT().RequeueStrategy()
				opA.EXPECT().ReadyCheck().Return(true, nil)
				opB.EXPECT().Ensure()
				opB.EXPECT().RequeueStrategy()
				opB.EXPECT().ReadyCheck().Return(true, nil)
				opC.EXPECT().Ensure()
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck().Return(true, nil)
			},
			times: 1,
		},
		{
			name: "parallel execution - requeue always - 1x",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue: true,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				// Reset counter used by conditionalEnsure.
				count = 0
				opA.EXPECT().Ensure().DoAndReturn(conditionalEnsure)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways)
				opA.EXPECT().ReadyCheck().Return(true, nil)
				opB.EXPECT().Ensure()
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck().Return(true, nil)
				// No execution of opC.
			},
			times: 1,
		},
		{
			name: "parallel execution - requeue always - 2x",
			opts: []CompositeOperatorOption{
				WithExecutionStrategy(executor.Parallel),
			},
			wantRequeue: false,
			expectations: func(opA, opB, opC *mocks.MockOperand) {
				// Reset counter used by conditionalEnsure.
				count = 0
				opA.EXPECT().Ensure().DoAndReturn(conditionalEnsure).Times(2)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways).AnyTimes()
				opA.EXPECT().ReadyCheck().Return(true, nil).Times(2)
				opB.EXPECT().Ensure().Times(2)
				opB.EXPECT().ReadyCheck().Return(true, nil).Times(2)
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opC.EXPECT().Ensure()
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck().Return(true, nil)
			},
			times: 2,
		},
		{
			name: "suspended",
			opts: []CompositeOperatorOption{
				WithSuspensionCheck(isSuspendedAlwaysTrue),
			},
			wantRequeue:  false,
			expectations: func(opA, opB, opC *mocks.MockOperand) {},
			times:        1,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock operands.
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			mA := mocks.NewMockOperand(mctrl)
			mB := mocks.NewMockOperand(mctrl)
			mC := mocks.NewMockOperand(mctrl)

			// A, B, C and C requires A.
			mA.EXPECT().Name().Return("opA").AnyTimes()
			mA.EXPECT().Requires().Return([]string{})
			mB.EXPECT().Name().Return("opB").AnyTimes()
			mB.EXPECT().Requires().Return([]string{})
			mC.EXPECT().Name().Return("opC").AnyTimes()
			mC.EXPECT().Requires().Return([]string{"opA"})

			// Set the expectations on the mocked operands.
			tc.expectations(mA, mB, mC)

			// Append a fake recorder to the list of options.
			rec := record.NewFakeRecorder(1)
			tc.opts = append(tc.opts, WithEventRecorder(rec), WithOperands(mA, mB, mC))

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
		})
	}
}

// TODO: Add TestCompositeOperatorCleanup.
