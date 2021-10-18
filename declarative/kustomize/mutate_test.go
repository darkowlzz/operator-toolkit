package kustomize

import (
	"testing"

	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/stretchr/testify/assert"
	apitypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

func TestMutate(t *testing.T) {
	wantKustomize := `apiVersion: kustomize.config.k8s.io/v1beta1
commonAnnotations:
  kqkq: lele
  lala: iaia
commonLabels:
  haha: xaxa
  oqoq: pqpq
images:
- name: someAppA
  newName: example/AppA
  newTag: v0.5.0
- digest: sha256:25a0d4
  name: someAppB
  newName: example/AppB
- name: someAppX
  newName: foo/AppX
  newTag: v5
kind: Kustomization
namePrefix: ttt
nameSuffix: yyy
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

	images := []apitypes.Image{
		{
			Name:    "someAppA",
			NewName: "example/AppA",
			NewTag:  "v0.5.0",
		},
		{
			Name:    "someAppB",
			NewName: "example/AppB",
			Digest:  "sha256:25a0d4",
		},
		{
			Name:    "someAppX",
			NewName: "foo/AppX",
			NewTag:  "v5",
		},
	}

	// List of mutations to apply.
	mut := []MutateFunc{
		AddCommonLabels(labels),
		AddCommonAnnotations(annotations),
		AddNamespace("test-ns"),
		AddResources([]string{"uuuu.yaml", "lll.yaml"}),
		AddImages(images),
		AddNamePrefix("ttt"),
		AddNameSuffix("yyy"),
	}

	Mutate(k, mut)

	// Write the result into a file (converting to yaml).
	err = WriteKustomizationInPackage(fs, k, "")
	assert.Nil(t, err)

	result, err := fs.ReadFile(kustomizationFile)
	assert.Nil(t, err)
	assert.Equal(t, wantKustomize, string(result))
}

func TestAddImages(t *testing.T) {
	existingKustomizeContent := `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: someAppA
  newName: example/AppA
  newTag: v0.5.0
- digest: sha256:25a0d4
  name: someAppB
  newName: example/AppB
- name: someAppX
  newName: foo/AppX
  newTag: v5
kind: Kustomization
`

	kustomization, err := LoadKustomization([]byte(existingKustomizeContent))
	assert.Nil(t, err)

	cases := []struct {
		name          string
		images        []apitypes.Image
		wantKustomize string
	}{
		{
			name: "add new image",
			images: []apitypes.Image{
				{
					Name: "fooAppA", NewName: "fooapp/appA", NewTag: "33",
				},
			},
			wantKustomize: `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: someAppA
  newName: example/AppA
  newTag: v0.5.0
- digest: sha256:25a0d4
  name: someAppB
  newName: example/AppB
- name: someAppX
  newName: foo/AppX
  newTag: v5
- name: fooAppA
  newName: fooapp/appA
  newTag: "33"
kind: Kustomization
`,
		},
		{
			name: "add existing image",
			images: []apitypes.Image{
				{
					Name: "someAppA", NewName: "fooexample/AppY", NewTag: "v10",
				},
			},
			wantKustomize: `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- digest: sha256:25a0d4
  name: someAppB
  newName: example/AppB
- name: someAppX
  newName: foo/AppX
  newTag: v5
- name: fooAppA
  newName: fooapp/appA
  newTag: "33"
- name: someAppA
  newName: fooexample/AppY
  newTag: v10
kind: Kustomization
`,
		},
		{
			name: "add duplicate images",
			images: []apitypes.Image{
				{
					Name: "someAppA", NewName: "fooexample/AppY", NewTag: "v10",
				},
				{
					Name: "someAppA", NewName: "barexample/AppZ", NewTag: "v20",
				},
			},
			wantKustomize: `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- digest: sha256:25a0d4
  name: someAppB
  newName: example/AppB
- name: someAppX
  newName: foo/AppX
  newTag: v5
- name: fooAppA
  newName: fooapp/appA
  newTag: "33"
- name: someAppA
  newName: fooexample/AppY
  newTag: v10
kind: Kustomization
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mutate := AddImages(tc.images)
			mutate(kustomization)
			r, err := yaml.Marshal(kustomization)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantKustomize, string(r))
		})
	}
}
