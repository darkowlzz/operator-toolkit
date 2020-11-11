package v1

import (
	"k8s.io/client-go/tools/record"
)

const (
	K8sEventTypeNormal  = "Normal"
	K8sEventTypeWarning = "Warning"
)

// ReconcilerEvent represents the action of the controller having actually done
// anything. Any meaningful change should have an associated event.
type ReconcilerEvent interface {

	// Record this into an event recorder as a Kubernetes API event
	Record(recorder record.EventRecorder)
}
