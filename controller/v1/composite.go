package v1

import (
	"github.com/go-logr/logr"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
)

// CompositeReconciler defines a composite reconciler.
type CompositeReconciler struct {
	Log           logr.Logger
	C             Controller
	InitCondition conditionsv1.Condition
	FinalizerName string
}
