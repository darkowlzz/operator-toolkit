package controller

import (
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// LogReconcileFinish is used to log the reconcile function execution
// information. The start time is the start time of the reconcile function, it
// is used to calculate the execution time of the function.
func LogReconcileFinish(log logr.Logger, msg string, start time.Time, result *ctrl.Result, e *error) {
	log.V(4).Info(msg, "execution-time", time.Since(start).String(), "result", *result, "error", e)
}
