package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/constant"
)

// Name of the instrumentation.
const instrumentationName = constant.LibraryName + "/operator"

// Operator is the operator interface that can be implemented by an operator to
// be used in a controller control loop.
type Operator interface {
	// IsSuspended tells if an operator is suspended and should not run any
	// operation.
	IsSuspended(context.Context, client.Object) bool

	// Ensure runs all the operands' Ensure method in order defined by their
	// dependencies.
	Ensure(context.Context, client.Object, metav1.OwnerReference) (result ctrl.Result, err error)

	// Cleanup runs all the operands' Delete method in reverse order defined by
	// their dependencies.
	Cleanup(context.Context, client.Object) (result ctrl.Result, err error)
}

// defaultIsSuspended always returns false.
func defaultIsSuspended(ctx context.Context, obj client.Object) bool {
	return false
}
