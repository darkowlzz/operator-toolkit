package admission

import (
	"context"
	goerrors "errors"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Validate-Funcs are functions in validating function chain, which forms a
// validating pipeline for a type of operation.
type ValidateCreateFunc func(ctx context.Context, obj client.Object) error
type ValidateUpdateFunc func(ctx context.Context, obj client.Object, oldObj client.Object) error
type ValidateDeleteFunc func(ctx context.Context, oldObj client.Object) error

// Validator defines functions for validating an operation.
type Validator interface {
	// ObjectGetter returns a new instance of the target object type of the
	// defaulter.
	ObjectGetter
	// ValidateCreate returns a list of validate functions for create event.
	ValidateCreate() []ValidateCreateFunc
	// ValidateCreate returns a list of validate functions for update event.
	ValidateUpdate() []ValidateUpdateFunc
	// ValidateCreate returns a list of validate functions for delete event.
	ValidateDelete() []ValidateDeleteFunc
	// RequireValidating can be used to perform a check before processing the
	// request object and decide if validating is required or the object can be
	// ignored. None of the validating functions will be called if this returns
	// false. In case of any error, true can be returned and a similar check
	// can be performed in a validate function that can return the proper error
	// message that'll be propogated to the user.
	RequireValidating(obj client.Object) bool
}

// ValidatingWebhookFor creates a new Webhook for validating the provided
// object type.
func ValidatingWebhookFor(validator Validator) *admission.Webhook {
	return &admission.Webhook{
		Handler: &validatingHandler{validator: validator},
	}
}

type validatingHandler struct {
	validator Validator
	decoder   *admission.Decoder
}

var _ admission.DecoderInjector = &validatingHandler{}

// InjectDecoder injects the decoder into a validatingHandler.
func (h *validatingHandler) InjectDecoder(d *admission.Decoder) error {
	h.decoder = d
	return nil
}

// Handle handles admission requests.
func (h *validatingHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	tr := otel.Tracer(tracerName)
	ctx, span := tr.Start(ctx, "validating-handle")
	defer span.End()

	if h.validator == nil {
		panic("validator should never be nil")
	}

	// Obtain a new object of the target type to decode the request object.
	obj := h.validator.GetNewObject()

	// Add namespace info into the object. The webhook payload only contains
	// runtime.Object without any metadata info.
	obj.SetNamespace(req.AdmissionRequest.Namespace)

	addRequestInfoIntoSpan(span, req.AdmissionRequest)

	if req.Operation == v1.Create {
		span.SetAttributes(attribute.String("operation", "create"))

		// Get the object in the request.
		span.AddEvent("Decode request object")
		err := h.decoder.Decode(req, obj)
		if err != nil {
			span.RecordError(err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		// Run the validations only if validation is required.
		if h.validator.RequireValidating(obj) {
			span.AddEvent("Run validating functions")
			span.SetAttributes(attribute.Int("validatecreate-func-count", len(h.validator.ValidateCreate())))
			for _, m := range h.validator.ValidateCreate() {
				if err := m(ctx, obj); err != nil {
					span.RecordError(err)
					var apiStatus errors.APIStatus
					if goerrors.As(err, &apiStatus) {
						return validationResponseFromStatus(false, apiStatus.Status())
					}
					return admission.Denied(err.Error())
				}
			}
		}
	}

	if req.Operation == v1.Update {
		span.SetAttributes(attribute.String("operation", "update"))

		oldObj := h.validator.GetNewObject()

		span.AddEvent("Decode request objects")
		err := h.decoder.DecodeRaw(req.Object, obj)
		if err != nil {
			span.RecordError(err)
			return admission.Errored(http.StatusBadRequest, err)
		}
		err = h.decoder.DecodeRaw(req.OldObject, oldObj)
		if err != nil {
			span.RecordError(err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		// Run the validations only if validation is required.
		if h.validator.RequireValidating(obj) {
			span.AddEvent("Run validating")
			span.SetAttributes(attribute.Int("validateupdate-func-count", len(h.validator.ValidateUpdate())))
			for _, m := range h.validator.ValidateUpdate() {
				if err := m(ctx, obj, oldObj); err != nil {
					span.RecordError(err)
					var apiStatus errors.APIStatus
					if goerrors.As(err, &apiStatus) {
						return validationResponseFromStatus(false, apiStatus.Status())
					}
					return admission.Denied(err.Error())
				}
			}
		}
	}

	if req.Operation == v1.Delete {
		span.SetAttributes(attribute.String("operation", "delete"))

		// In reference to PR: https://github.com/kubernetes/kubernetes/pull/76346
		// OldObject contains the object being deleted
		span.AddEvent("Decode request object")
		err := h.decoder.DecodeRaw(req.OldObject, obj)
		if err != nil {
			span.RecordError(err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		// Run the validations only if validation is required.
		if h.validator.RequireValidating(obj) {
			span.AddEvent("Run validating")
			span.SetAttributes(attribute.Int("validatedelete-func-count", len(h.validator.ValidateDelete())))
			for _, m := range h.validator.ValidateDelete() {
				if err := m(ctx, obj); err != nil {
					span.RecordError(err)
					var apiStatus errors.APIStatus
					if goerrors.As(err, &apiStatus) {
						return validationResponseFromStatus(false, apiStatus.Status())
					}
					return admission.Denied(err.Error())
				}
			}
		}
	}

	span.SetAttributes(attribute.Bool("allowed", true))

	return admission.Allowed("")
}

// validationResponseFromStatus returns a response for admitting a request with provided Status object.
func validationResponseFromStatus(allowed bool, status metav1.Status) admission.Response {
	resp := admission.Response{
		AdmissionResponse: v1.AdmissionResponse{
			Allowed: allowed,
			Result:  &status,
		},
	}
	return resp
}
