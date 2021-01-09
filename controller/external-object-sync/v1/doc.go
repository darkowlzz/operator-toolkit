// Package v1 provides helpers to implement an external object sync controller.
// It's based on the sync controller and adds a garbage collector sync function
// for the purpose of syncing objects between a kubernetes cluster and an
// external system. The garbage collector deletes orphan objects in the
// external system.
package v1
