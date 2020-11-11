package v1

// Object is an interface for an API object to implement.
type Object interface {
	// Default sets default values for optional object fields. This can be used
	// in webhooks and in the Reconciler to ensure the object contains default
	// values for optionalonal fields.
	Default()

	// ValidateCreate validates that all the required fields are present and
	// valid.
	ValidateCreate() error

	// ValidateUpdate validates that only supported fields are changed.
	ValidateUpdate() error

	// ValidateDelete validates that deletion is allowed.
	ValidateDelete() error
}
