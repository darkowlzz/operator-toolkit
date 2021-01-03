package v1

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *SyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, reterr error) {
	tr := otel.Tracer("Reconcile")
	ctx, span := tr.Start(ctx, "reconcile")
	defer span.End()

	controller := s.ctrlr

	// Get an instance of the object.
	instance := s.prototype.DeepCopyObject().(client.Object)
	if getErr := s.client.Get(ctx, req.NamespacedName, instance); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// Not found means that it's a delete event. Delete the associated
			// object from the external system.

			// Populate the info about the deleted object into the instance
			// before calling Delete.
			instance.SetName(req.Name)
			instance.SetNamespace(req.Namespace)

			if delErr := controller.Delete(ctx, instance); delErr != nil {
				result = ctrl.Result{Requeue: true}
				reterr = fmt.Errorf("failed to delete %v from external system: %w", req.NamespacedName, delErr)
			}
		} else {
			reterr = getErr
		}
		return
	}

	// Ensure the object exists in the external system.
	if ensureErr := controller.Ensure(ctx, instance); ensureErr != nil {
		result = ctrl.Result{Requeue: true}
		reterr = fmt.Errorf("failed to ensure %v in the external system: %w", req.NamespacedName, ensureErr)
	}

	return
}
