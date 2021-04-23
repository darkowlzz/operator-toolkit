// Package composite provides a k8s client composed of a cached and an uncached
// client.
// For Get operations, the cache is used, in case the target object isn't
// available in the cache, the client will perform a GET call using the
// uncached client to get the object directly from the k8s api server.
// For List operations, the client can be configured to use the cache or
// directly list from the k8s api server. Unlike Get, List does not return
// error when objects are not found. It returns an empty list. The decision to
// retry without cache can't be made for List operations.
package composite
