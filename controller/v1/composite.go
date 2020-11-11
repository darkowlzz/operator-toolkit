package v1

import (
	"github.com/go-logr/logr"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"k8s.io/client-go/tools/record"
)

// CompositeReconciler defines a composite reconciler.
type CompositeReconciler struct {
	Log           logr.Logger
	C             Controller
	InitCondition conditionsv1.Condition
	FinalizerName string
	Recorder      record.EventRecorder
}
