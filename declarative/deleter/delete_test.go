package deleter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/applier"
)

func TestDelete(t *testing.T) {
	// TODO: Move this into an envtest and enable it.
	t.Skip("skip this for unit tests, it shows an example of DirectDeleter in presence of a k8s cluster")

	nsManifest := `
apiVersion: v1
kind: Namespace
metadata:
  name: toolkit-test-ns
`
	// Use the applier to create the namespace.
	a := applier.NewDirectApplier()
	err := a.Apply(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)

	// Delete the namespace using the deleter.
	d := NewDirectDeleter()
	err = d.Delete(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)
}
