package singleton

import "fmt"

// multipleObjectsFound implements multipleInstances error interface to be used
// with IsMultipleInstancesFound().
type multipleObjectsFound struct {
	kind      string
	instances int
}

func (mo *multipleObjectsFound) Error() string {
	return fmt.Sprintf("multiple instances (%d) of %q found", mo.instances, mo.kind)
}

func (mo *multipleObjectsFound) MultipleInstances() (bool, int) {
	return true, mo.instances
}

func newMultipleObjectsFound(kind string, count int) *multipleObjectsFound {
	return &multipleObjectsFound{kind: kind, instances: count}
}
