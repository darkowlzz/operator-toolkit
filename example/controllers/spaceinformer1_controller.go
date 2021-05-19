package controllers

import (
	"context"

	"github.com/darkowlzz/operator-toolkit/source"
	"github.com/darkowlzz/operator-toolkit/telemetry"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
)

// SpaceInformer1Reconciler reconciles external object from space.
type SpaceInformer1Reconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Instrumentation *telemetry.Instrumentation
}

func (r *SpaceInformer1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_, _, _, log := r.Instrumentation.Start(ctx, "spaceInformer1.Reconcile")
	log = log.WithValues("spaceinformer1", req.NamespacedName)

	log.Info("reconciling game", "req", req)

	return ctrl.Result{}, nil
}

func (r *SpaceInformer1Reconciler) SetupWithManager(mgr ctrl.Manager, spaceCache cache.Cache) error {
	// Create a new controller with the reconciler.
	c, err := controller.New("spaceinformer1-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch Games and enqueue Game object key.
	if err := c.Watch(source.NewKindWithCache(&appv1alpha1.Game{}, spaceCache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}
