// Package v1 provides helpers to build a sync controller. It defines a
// Controller interface which provides methods required for reconciling a sync
// controller. The package also provides an implementation of the Reconcile
// method that can be embedded in a controller to satisfy the
// controller-runtime's Reconciler interface. It supports plugging in sync
// functions that run as go routines and help with keeping the systems in sync.
package v1
