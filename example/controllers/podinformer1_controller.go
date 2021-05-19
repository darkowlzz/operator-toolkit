package controllers

import (
	"context"

	"github.com/darkowlzz/operator-toolkit/source"
	"github.com/darkowlzz/operator-toolkit/telemetry"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
)

// PodInformer1Reconciler reconciles external object from space.
type PodInformer1Reconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Instrumentation *telemetry.Instrumentation
}

func (r *PodInformer1Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_, _, _, log := r.Instrumentation.Start(ctx, "podInformer1.Reconcile")
	log = log.WithValues("podinformer1", req.NamespacedName)

	log.Info("reconciling pod", "req", req)

	return ctrl.Result{}, nil
}

func (r *PodInformer1Reconciler) SetupWithManager(mgr ctrl.Manager, spaceCache cache.Cache) error {
	// Create a new controller with the reconciler.
	c, err := controller.New("podinformer1-controller", mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return err
	}

	// Watch Pods and enqueue Pod object key.
	if err := c.Watch(source.NewKindWithCache(&corev1.Pod{}, spaceCache), &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	return nil
}
