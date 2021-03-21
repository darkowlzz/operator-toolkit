// Package source is a modification of the controller-runtime pkg/source
// package to use the same Kind based source but without the hard requirement
// to register them with the k8s API server. This source is meant to be used
// when the event source is not k8s and the event object cache is based on
// client-go informers.
package source
