package function

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/singleton"
	"github.com/darkowlzz/operator-toolkit/webhook/admission"
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

// ValidateSingletonCreate allows singleton of an object type. If an instance
// of an object already exists, this will prevent creation of another object.
func ValidateSingletonCreate(sf singleton.GetInstanceFunc, c client.Client) admission.ValidateCreateFunc {
	return func(ctx context.Context, obj client.Object) error {
		// Get singleton.
		o, err := sf(ctx, c)
		if err != nil {
			return err
		}
		// If the returned object isn't nil, an instance already exists.
		if o != nil {
			return fmt.Errorf("an instance of %q - %s already exists", o.GetObjectKind().GroupVersionKind().Kind, client.ObjectKeyFromObject(o))
		}
		return nil
	}
}
