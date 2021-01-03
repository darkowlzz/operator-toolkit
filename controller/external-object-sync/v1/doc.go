// Package v1 helpers to build an external object sync controller reconciler.
// It defines a Controller interface which provides methods required for
// reconciling an external object sync controller. An external object sync
// controller syncs objects between a kubernetes cluster and an external
// system. The package also provides an implementation of the Reconcile method
// that can be embedded in a controller to satisfy the controller-runtime's
// Reconciler interface. The reconciler also implements a garbage collector to
// delete orphaned objects in the external system.
package v1
