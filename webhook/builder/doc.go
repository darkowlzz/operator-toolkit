// Package builder is based on the controller-runtime webhook builder. It's
// modified to support building webhooks for the unified admission controller
// that supports creating webhooks for both native and custom resources.
package builder

import ctrl "sigs.k8s.io/controller-runtime"

var log = ctrl.Log.WithName("webhook").WithName("builder")
