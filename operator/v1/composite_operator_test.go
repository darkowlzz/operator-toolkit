package v1

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/executor"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand/mocks"
)

// fooCreatedEvent is a ReconcilerEvent type used for testing event
// broadcasting.
type fooCreatedEvent struct {
	Object  client.Object
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
func isSuspendedAlwaysTrue(ctx context.Context, obj client.Object) bool {
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
	conditionalEnsure := func(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opA.EXPECT().RequeueStrategy()
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opB.EXPECT().RequeueStrategy()
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opC.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opC.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(conditionalEnsure)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways)
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(conditionalEnsure).Times(2)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways).Times(2)
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil).Times(2)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil).Times(2)
				opC.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opC.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opA.EXPECT().RequeueStrategy()
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opB.EXPECT().RequeueStrategy()
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opC.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opC.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(conditionalEnsure)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways)
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				opA.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(conditionalEnsure).Times(2)
				opA.EXPECT().RequeueStrategy().Return(operand.RequeueAlways).AnyTimes()
				opA.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
				opA.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil).Times(2)
				opB.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
				opB.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil).Times(2)
				opB.EXPECT().RequeueStrategy().AnyTimes()
				opB.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil).Times(2)
				opC.EXPECT().Ensure(gomock.Any(), gomock.Any(), gomock.Any())
				opC.EXPECT().RequeueStrategy()
				opC.EXPECT().ReadyCheck(gomock.Any(), gomock.Any()).Return(true, nil)
				opC.EXPECT().PostReady(gomock.Any(), gomock.Any()).Return(nil)
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
				res, eerr = co.Ensure(context.Background(), pod, metav1.OwnerReference{})
				assert.Nil(t, eerr)
				i++
			}

			assert.Equal(t, tc.wantRequeue, res.Requeue, "requeue value")
		})
	}
}

// TODO: Add TestCompositeOperatorCleanup.
