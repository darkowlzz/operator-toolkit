// Package client provides an implementation of the controller-runtime's
// generic client with RBAC recording action recording. It wraps an actual
// client, records the API call object and verb, and forwards the call to the
// actual embedded client. The package also provides tooling to convert the
// recorded RBAC metadata into RBAC Role
package client
