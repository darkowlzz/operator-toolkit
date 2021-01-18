package game

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	compositev1 "github.com/darkowlzz/operator-toolkit/controller/composite/v1"
	"github.com/darkowlzz/operator-toolkit/object"
	operatorv1 "github.com/darkowlzz/operator-toolkit/operator/v1"
)

// GameController is a controller that implements the CompositeReconciler
// contoller interface. It watches the Game CRD.
type GameController struct {
	Operator operatorv1.Operator
}

var _ compositev1.Controller = &GameController{}

func (gc *GameController) Default(context.Context, client.Object) {}

func (gc *GameController) Validate(context.Context, client.Object) error { return nil }

func (gc *GameController) Initialize(context.Context, client.Object, metav1.Condition) error {
	return nil
}

func (gc *GameController) Operate(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return gc.Operator.Ensure(ctx, obj, object.OwnerReferenceFromObject(obj))
}

func (gc *GameController) Cleanup(context.Context, client.Object) (result ctrl.Result, err error) {
	return ctrl.Result{}, nil
}

func (gc *GameController) UpdateStatus(context.Context, client.Object) error {
	return nil
}
