package game

import (
	"context"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	controllerv1 "github.com/darkowlzz/composite-reconciler/controller/v1"
	operatorv1 "github.com/darkowlzz/composite-reconciler/operator/v1"
)

// GameController is a controller that implements the CompositeReconciler
// contoller interface. It watches the Game CRD.
type GameController struct {
	Operator operatorv1.Operator
}

var _ controllerv1.Controller = &GameController{}

func (gc *GameController) Default(context.Context, client.Object) {}

func (gc *GameController) Validate(context.Context, client.Object) error { return nil }

func (gc *GameController) Initialize(context.Context, client.Object, conditionsv1.Condition) error {
	return nil
}

func (gc *GameController) Operate(ctx context.Context, obj client.Object) (result ctrl.Result, err error) {
	return gc.Operator.Ensure(ctx, obj, controllerv1.OwnerReferenceFromObject(obj))
}

func (gc *GameController) Cleanup(context.Context, client.Object) (result ctrl.Result, err error) {
	return ctrl.Result{}, nil
}

func (gc *GameController) UpdateStatus(context.Context, client.Object) error {
	return nil
}
