package admission

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	admissionv1 "k8s.io/api/admission/v1"

	"github.com/darkowlzz/operator-toolkit/constant"
)

// Name of the tracer.
const tracerName = constant.LibraryName + "/webhook/admission"

// addRequestInfoIntoSpan adds the admission request information into a trace
// span.
func addRequestInfoIntoSpan(s trace.Span, req admissionv1.AdmissionRequest) {
	s.SetAttributes(attribute.String("namespace", req.Namespace))
	s.SetAttributes(attribute.String("name", req.Name))
	s.SetAttributes(attribute.Any("kind", req.Kind))
	// RequestKind is found to be nil in tests where a minimal admission
	// request is created, causing a panic. Other unset fields aren't nil.
	if req.RequestKind != nil {
		s.SetAttributes(attribute.Any("requestKind", req.RequestKind))
	}
	s.SetAttributes(attribute.Any("resource", req.Resource))
	s.SetAttributes(attribute.Any("uid", req.UID))
	s.SetAttributes(attribute.Any("userInfo", req.UserInfo))
}
