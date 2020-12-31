// Package v1 contains helpers to build a composite controller reconciler.
// It defines a Controller interface which provides methods required for
// reconciling a composite controller. A composite controller manages a set of
// child objects based on the desired state specified in a parent object. The
// package also provides an implementation of the Reconcile method that can be
// embedded in a controller to satisfy the controller-runtime's Reconciler
// interface.
package v1
