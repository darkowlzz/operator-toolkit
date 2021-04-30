// Package admission provides interfaces for building admission controllers
// with chaining function with support for both k8s native and custom resources
// in a consistent way. It's based on the custom resource admission webhook
// from controller-runtime, modified to not be specific to custom resources
// only. The defaulter and validator can have multiple functions, chained
// together to form a processing pipeline. They also have the ability to
// perform checks in advance before passing the object to the processing
// pipeline to avoid repetitive checks in each of the functions for filtering
// the objects and ignoring if needed.
package admission
