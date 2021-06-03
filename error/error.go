package error

// multipleInstances defines an interface for errors to implement when an error
// is caused by multiple instances of something.
type multipleInstances interface {
	MultipleInstances() (bool, int)
}

// IsMultipleInstancesFound checks if the given error is due to multiple
// instances of a target object type.
func IsMultipleInstancesFound(err error) (bool, int) {
	if e, ok := err.(multipleInstances); ok {
		return e.MultipleInstances()
	}
	return false, 0
}
