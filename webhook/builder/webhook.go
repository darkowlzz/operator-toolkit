package builder

import (
	"net/http"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	tkAdmission "github.com/darkowlzz/operator-toolkit/webhook/admission"
)

// Builder builds a Webhook.
type Builder struct {
	c            tkAdmission.Controller
	mgr          manager.Manager
	mutatePath   string
	validatePath string
}

// WebhookManagedBy adds the manager to the builder.
func WebhookManagedBy(m manager.Manager) *Builder {
	return &Builder{mgr: m}
}

// MutatePath is the mutation webhook endpoint path to use when registering the
// mutating webhook.
func (blder *Builder) MutatePath(path string) *Builder {
	blder.mutatePath = path
	return blder
}

// ValidatePath is the validating webhook endpoint path to use when registering
// the validating webhook.
func (blder *Builder) ValidatePath(path string) *Builder {
	blder.validatePath = path
	return blder
}

// Complete builds the webhook.
func (blder *Builder) Complete(c tkAdmission.Controller) error {
	blder.c = c
	return blder.registerWebhooks()
}

// registerWebhooks registers the defaulting and validating webhooks based on
// their endpoint paths.
func (blder *Builder) registerWebhooks() error {
	if blder.mutatePath != "" {
		blder.registerDefaultingWebhook()
	}

	if blder.validatePath != "" {
		blder.registerValidatingWebhook()
	}

	return nil
}

// registerDefaultingWebhook builds and registers the defaulting webhook.
func (blder *Builder) registerDefaultingWebhook() {
	mwh := tkAdmission.DefaultingWebhookFor(blder.c)
	if mwh != nil {
		path := blder.mutatePath

		// Checking if the path is already registered.
		// If so, just skip it.
		if !blder.isAlreadyHandled(path) {
			log.Info("Registering a mutating webhook",
				"controller", blder.c.Name(),
				"path", path)
			blder.mgr.GetWebhookServer().Register(path, mwh)
		} else {
			log.Info("Webhook path already registered, skipping registration",
				"controller", blder.c.Name(),
				"path", path)
		}
	}
}

// registerValidatingWebhook builds and registers the validating webhook.
func (blder *Builder) registerValidatingWebhook() {
	vwh := tkAdmission.ValidatingWebhookFor(blder.c)
	if vwh != nil {
		path := blder.validatePath

		// Checking if the path is already registered.
		// If so, just skip it.
		if !blder.isAlreadyHandled(path) {
			log.Info("Registering a validating webhook",
				"controller", blder.c.Name(),
				"path", path)
			blder.mgr.GetWebhookServer().Register(path, vwh)
		} else {
			log.Info("Webhook path already registered, skipping registration",
				"controller", blder.c.Name(),
				"path", path)
		}
	}
}

// isAlreadyHandled checks if a webhook endpoint path is already registered.
func (blder *Builder) isAlreadyHandled(path string) bool {
	if blder.mgr.GetWebhookServer().WebhookMux == nil {
		return false
	}
	h, p := blder.mgr.GetWebhookServer().WebhookMux.Handler(&http.Request{URL: &url.URL{Path: path}})
	if p == path && h != nil {
		return true
	}
	return false
}
