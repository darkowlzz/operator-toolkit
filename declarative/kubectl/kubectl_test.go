package kubectl

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestKubectl(t *testing.T) {
	// TODO: Move this into an envtest and enable it.
	t.Skip("skip this for unit tests, it shows an example of KubectlClient in presence of a k8s cluster")

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

	k := New().IOStreams(ioStreams)

	// Use the applier to create the namespace.
	err := k.Apply(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)

	// Delete the namespace using the deleter.
	err = k.Delete(context.Background(), "", nsManifest, false)
	assert.Nil(t, err)
}
