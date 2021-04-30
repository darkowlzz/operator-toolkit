package function

import (
	"context"
	"fmt"

	"github.com/darkowlzz/operator-toolkit/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ValidateLabels takes an object and a set of labels and validates the
// object's labels based on the given set of labels. Unknown labels results in
// failure.
func ValidateLabels(obj client.Object, labels map[string]string) error {
	for k := range obj.GetLabels() {
		// Valid only if the key if found in the given set of labels.
		if _, exists := labels[k]; !exists {
			return fmt.Errorf("found unexpected label key %q", k)
		}
	}
	return nil
}

func ValidateLabelsCreate(labels map[string]string) admission.ValidateCreateFunc {
	return func(ctx context.Context, obj client.Object) error {
		return ValidateLabels(obj, labels)
	}
}

func ValidateLabelsUpdate(labels map[string]string) admission.ValidateUpdateFunc {
	return func(ctx context.Context, obj client.Object, oldobj client.Object) error {
		return ValidateLabels(obj, labels)
	}
}
