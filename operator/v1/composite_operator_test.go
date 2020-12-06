package v1

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"

	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
)

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

	rec := record.NewFakeRecorder(1)
	co, err := NewCompositeOperator(
		WithOperands(operands...),
		WithEventRecorder(rec),
	)

	if err != nil {
		t.Errorf("unexpected error while creating a composite operator: %v", err)
	}
	od := co.Order()
	if od.String() != wantOrder {
		t.Errorf("unexpected operator order:\n\t(WNT) %q\n\t(GOT) %q", wantOrder, od.String())
	}
}

func TestCompositeOperatorEnsure(t *testing.T) {
	operandA := &operand.Operand{
		Name: "opA",
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("RUNNING opA")
			time.Sleep(2 * time.Second)
			fmt.Println("ENDING opA")

			pod := &corev1.Pod{}
			evnt := &fooCreatedEvent{Object: pod, FooName: "foo foo"}
			return evnt, nil
			// return nil, nil
			// return errors.New("some error for opA")
		},
		Delete: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("DELETING opA")
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			return true, nil
		},
		// Setting RequeueAlways results in early return from ExecuteOperands
		// and operandC doesn't get executed. It's intended. In a real
		// reconciler, the operands will be re-executed via requeue.
		// Requeue: operand.RequeueAlways,
	}

	operandB := &operand.Operand{
		Name: "opB",
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("RUNNING opB")
			time.Sleep(2 * time.Second)
			fmt.Println("ENDING opB")
			return nil, nil
			// return errors.New("some error for opB")
		},
		Delete: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("DELETING opB")
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			return true, nil
		},
	}

	operandC := &operand.Operand{
		Name:     "opC",
		Requires: []string{operandA.Name},
		Ensure: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("RUNNING opC")
			return nil, nil
		},
		Delete: func() (eventv1.ReconcilerEvent, error) {
			fmt.Println("DELETING opC")
			return nil, nil
		},
		ReadyCheck: func() (bool, error) {
			return true, nil
		},
	}

	// tests := []struct {
	//     name string
	//     operands []*operand.Operand
	//     wantRequeue bool
	// }{
	//     {
	//         name: ""
	//     },
	// }

	rec := record.NewFakeRecorder(1)
	co, err := NewCompositeOperator(
		WithOperands(operandA, operandB, operandC),
		WithEventRecorder(rec),
	)
	if err != nil {
		t.Errorf("unexpected error while creating a composite operator: %v", err)
	}

	res, eerr := co.Ensure()
	fmt.Println("EERR:", eerr)
	fmt.Println("RES:", res)

	cres, ceerr := co.Cleanup()
	fmt.Println("EERR:", ceerr)
	fmt.Println("RES:", cres)

}
