package deleter

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/darkowlzz/operator-toolkit/declarative/applier"
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
	ioStreams := genericclioptions.IOStreams{
		Out:    ioutil.Discard,
		ErrOut: ioutil.Discard,
	}

	// Use the applier to create the namespace.
	a := applier.NewDirectApplier().IOStreams(ioStreams)
	err := a.Apply(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)

	// Delete the namespace using the deleter.
	d := NewDirectDeleter().IOStreams(ioStreams)
	err = d.Delete(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)
}
