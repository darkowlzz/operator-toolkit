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

// SpaceInformer2Reconciler reconciles external object from space.
type SpaceInformer2Reconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Instrumentation *telemetry.Instrumentation
}

func (r *SpaceInformer2Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_, span, _, log := r.Instrumentation.Start(ctx, "spaceInformer2.Reconcile")
	defer span.End()

	log = log.WithValues("spaceinformer2", req.NamespacedName)

	log.Info("reconciling game", "req", req)

	return ctrl.Result{}, nil
}

func (r *SpaceInformer2Reconciler) SetupWithManager(mgr ctrl.Manager, spaceCache cache.Cache) error {
	// Create a new controller with the reconciler.
	c, err := controller.New("spaceinformer2-controller", mgr, controller.Options{
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
