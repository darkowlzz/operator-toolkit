// Package external provides tools to build an external controller with the
// same core components that a k8s controller uses. Instead of using client-go
// based informers for fetching the events from the cluster, it uses a Channel
// source which receives generic events. The generic event contain
// client.Object type object which can be used store the external object
// details in the object metadata, maybe in annotations or labels, or in the
// body of an unstructured object. These generic events are processed by
// a cache based event handler which enqueues the object based on the cache
// data. The reconcile loop receives NamespacedName because it's based on k8s
// object model, but the namespace and name can be set to anything that can be
// used to save unique objects key in the cache. The cache can store extra
// information about the external object. It can be queried by the reconciler
// to get full information about the desired state.
package external
