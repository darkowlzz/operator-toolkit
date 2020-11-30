package operate

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestCompositeOperator(t *testing.T) {
	operandA := &Operand{
		Name: "OperandA",
		Obj: &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "podA",
				Namespace: "default",
			},
		},
		DependsOn: []string{},
	}

	co := CompositeOperator{Operands: []Operands{}}
	_ = co.Order()
}
