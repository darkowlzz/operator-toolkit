// Package webhook providers tooling to generate self signed certificate and
// use it with a webhook server. It also helps inject the webhook configuration
// with the generated certificate.

// NOTE: This package originates from controller-runtime v0.1. The later
// versions of controller-runtime removed support for self signed certificate
// generation for webhook configuration. The package has been updated to
// support the new kubernetes APIs and controller-runtime.
package webhook
