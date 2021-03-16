// Package v1 provides helpers for building a stateless-action controller. It
// can be used to trigger an action on an observed event just once without
// keeping any state for retry on controller restart. The event could be from
// k8s or any other source. This package defines a Controller interface which
// provides methods required for executing an action based on an event. The
// action is defined using an action manager which allows targetting the action
// on any object, not just the event source object.
package v1
