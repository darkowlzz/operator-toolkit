package admission

import (
	"context"
	"encoding/json"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// DefaultFunc is a function in the defaulting function chain, which forms a
// defaulter pipeline.
type DefaultFunc func(ctx context.Context, obj client.Object)

// Defaulter defines functions for setting defaults on resource.
type Defaulter interface {
	// ObjectGetter returns a new instance of the target object type of the
	// defaulter.
	ObjectGetter
	// Default returns a list of default functions that form the defaulting
	// pipeline.
	Default() []DefaultFunc
	// RequireDefaulting can be used to perform a check before processing the
	// request object and decide if defaulting is required or the object can be
	// ignore. Default() will not be called if this returns false. In case of
	// any error, true can be returned and a similar check can be performed in
	// a defaulting function that can return the proper error message that'll
	// be propagated to the user.
	RequireDefaulting(obj client.Object) bool
}

// DefaultingWebhookFor creates a new webhook for Defaulting the provided
// object type.
func DefaultingWebhookFor(defaulter Defaulter) *admission.Webhook {
	return &admission.Webhook{
		Handler: &mutatingHandler{defaulter: defaulter},
	}
}

type mutatingHandler struct {
	defaulter Defaulter
	decoder   *admission.Decoder
}

var _ admission.DecoderInjector = &mutatingHandler{}

// InjectDecoder injects the decoder into a mutatingHandler.
func (h *mutatingHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

// Handle handles admission requests.
func (h *mutatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	tr := otel.Tracer(tracerName)
	ctx, span := tr.Start(ctx, "mutating-handle")
	defer span.End()

	if h.defaulter == nil {
		panic("defaulter should never be nil")
	}

	// Obtain a new object of the target type to decode the request object.
	obj := h.defaulter.GetNewObject()

	// Add namespace info into the object. The webhook payload only contains
	// runtime.Object without any metadata info.
	obj.SetNamespace(req.AdmissionRequest.Namespace)

	addRequestInfoIntoSpan(span, req.AdmissionRequest)

	// Get the object in the request.
	span.AddEvent("Decode request object")
	err := h.decoder.Decode(req, obj)
	if err != nil {
		span.RecordError(err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Run the defaulters only if defaulting is required.
	if h.defaulter.RequireDefaulting(obj) {
		span.AddEvent("Run defaulting functions")
		span.SetAttributes(attribute.Int("default-func-count", len(h.defaulter.Default())))
		// Process the object through the defaulting pipeline.
		for _, m := range h.defaulter.Default() {
			m(ctx, obj)
		}
	}

	span.AddEvent("Marshal object")
	marshalled, err := json.Marshal(obj)
	if err != nil {
		span.RecordError(err)
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Create the patch
	span.AddEvent("Create patch response")
	return admission.PatchResponseFromRaw(req.Object.Raw, marshalled)
}
