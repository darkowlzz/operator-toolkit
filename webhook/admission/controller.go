package admission

import "sigs.k8s.io/controller-runtime/pkg/client"

// Controller defines an interface for a webhook admission controller.
type Controller interface {
	// Name returns the name of the controller.
	Name() string
	Defaulter
	Validator
}

// ObjectGetter defines an interface for getting an object of any type,
// depending on the controller's target object type, in a generic form.
type ObjectGetter interface {
	// GetNewObject returns a new initialized object of the target object type.
	GetNewObject() client.Object
}
