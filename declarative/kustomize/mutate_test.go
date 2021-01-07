package kustomize

import (
	"testing"

	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/stretchr/testify/assert"
)

func TestMutate(t *testing.T) {
	wantKustomize := `apiVersion: kustomize.config.k8s.io/v1beta1
commonAnnotations:
  kqkq: lele
  lala: iaia
commonLabels:
  haha: xaxa
  oqoq: pqpq
kind: Kustomization
namespace: test-ns
resources:
- role.yaml
- service_account.yaml
- uuuu.yaml
- lll.yaml
`

	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	k, err := LoadKustomizationFromPackage(fs, "guestbook")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(k.Resources))

	labels := map[string]string{
		"haha": "xaxa",
		"oqoq": "pqpq",
	}
	annotations := map[string]string{
		"lala": "iaia",
		"kqkq": "lele",
	}

	// List of mutations to apply.
	mut := []MutateFunc{
		AddCommonLabels(labels),
		AddCommonAnnotations(annotations),
		AddNamespace("test-ns"),
		AddResources([]string{"uuuu.yaml", "lll.yaml"}),
	}

	Mutate(k, mut)

	// Write the result into a file (converting to yaml).
	err = WriteKustomizationInPackage(fs, k, "")
	assert.Nil(t, err)

	result, err := fs.ReadFile(kustomizationFile)
	assert.Nil(t, err)
	assert.Equal(t, wantKustomize, string(result))
}
