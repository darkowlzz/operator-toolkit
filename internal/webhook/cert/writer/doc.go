// Package writer provides method to provision and persist the certificates.
// It will create the certificates if they don't exist.
// It will ensure the certificates are valid and not expiring. If not, it will
// recreate them.

// NOTE: This package originates from controller-runtime v0.1. The later
// versions of controller-runtime removed support for self signed certificate
// generation for webhook configuration. The package has been updated to
// support the new kubernetes APIs and controller-runtime.
package writer

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("admission").WithName("cert").WithName("writer")
