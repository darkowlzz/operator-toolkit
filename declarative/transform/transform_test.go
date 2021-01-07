package transform

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/darkowlzz/operator-toolkit/declarative/loader"
)

func TestSetOwnerReference(t *testing.T) {
	// Create an in-memory filesystem and load the packages in it.
	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	targetFile := "registry/db.yaml"

	ownerRefs := []metav1.OwnerReference{
		metav1.OwnerReference{
			APIVersion: "someapi/v1",
			Kind:       "Somekind",
			Name:       "somename",
			UID:        "17d16671-513f-4026-9302-904fe90601cf",
		},
		metav1.OwnerReference{
			APIVersion: "someotherapi/v1",
			Kind:       "SomekindX",
			Name:       "somenameX",
			UID:        "58e31192-513f-4026-9302-904fe90601ca",
		},
	}

	wantManifest := `apiVersion: example.com/v1
kind: DB
metadata:
  name: test-db
  ownerReferences:
    - apiVersion: someapi/v1
      blockOwnerDeletion: false
      controller: false
      kind: Somekind
      name: somename
      uid: 17d16671-513f-4026-9302-904fe90601cf
    - apiVersion: someotherapi/v1
      blockOwnerDeletion: false
      controller: false
      kind: SomekindX
      name: somenameX
      uid: 58e31192-513f-4026-9302-904fe90601ca
`

	manifestTransform := ManifestTransform{
		targetFile: []TransformFunc{
			SetOwnerReference(ownerRefs),
		},
	}

	err = Transform(fs, manifestTransform)
	assert.Nil(t, err)

	// Read the manifest and verify the result.
	b, err := fs.ReadFile(targetFile)
	assert.Nil(t, err)
	assert.Equal(t, wantManifest, string(b))

}

func TestReplicaTransform(t *testing.T) {
	// Create an in-memory filesystem and load the packages in it.
	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	targetFile := "registry/db.yaml"

	wantManifest := `apiVersion: example.com/v1
kind: DB
metadata:
  name: test-db
spec:
  replicas: 3
`

	manifestTransform := ManifestTransform{
		targetFile: []TransformFunc{
			SetReplicaFunc(3),
		},
	}

	err = Transform(fs, manifestTransform)
	assert.Nil(t, err)

	// Read the manifest and verify the result.
	b, err := fs.ReadFile(targetFile)
	assert.Nil(t, err)
	assert.Equal(t, wantManifest, string(b))
}

func TestTransform(t *testing.T) {
	// Create an in-memory filesystem and load the packages in it.
	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	// Labels to apply.
	labels := map[string]string{
		"mylabel1": "val1",
		"mylabel2": "val2",
	}

	annotations := map[string]string{
		"annot1": "anot-val1",
		"annot2": "anot-val2",
	}

	wantOwnerRefs := `
    - apiVersion: some-api-version
      blockOwnerDeletion: false
      controller: false
      kind: some-kind
      name: some-name
      uid: 58e31192-513f-4026-9302-904fe90601ca
`

	targetFileA := "guestbook/role.yaml"
	targetFileB := "registry/db.yaml"

	// Create a manifest transform.
	manifestTransform := ManifestTransform{
		targetFileA: []TransformFunc{
			AddLabelsFunc(labels),
			AddAnnotationsFunc(annotations),
		},
		targetFileB: []TransformFunc{
			AddLabelsFunc(labels),
		},
	}

	// Create owner refs for using SetOwnerReference as a common transform.
	ownerRefs := []metav1.OwnerReference{
		{
			APIVersion: "some-api-version",
			Kind:       "some-kind",
			Name:       "some-name",
			UID:        "58e31192-513f-4026-9302-904fe90601ca",
		},
	}

	// Run transform with common transform.
	err = Transform(fs, manifestTransform, SetOwnerReference(ownerRefs))
	assert.Nil(t, err)

	checkLabelsAnnotationsAndOwnerRefs(t, fs, targetFileA, labels, annotations, wantOwnerRefs)
	checkLabelsAnnotationsAndOwnerRefs(t, fs, targetFileB, labels, nil, wantOwnerRefs)
}

func checkLabelsAnnotationsAndOwnerRefs(t *testing.T, fs *loader.ManifestFileSystem, file string, labels, annotations map[string]string, ownerRefs string) {
	// Read the file and check the results.
	b, err := fs.ReadFile(file)
	assert.Nil(t, err)
	obj, err := yaml.Parse(string(b))
	assert.Nil(t, err)

	// Check if the labels exist in the obtained object.
	l, err := obj.GetLabels()
	assert.Nil(t, err)
	for k, v := range labels {
		assert.Equal(t, v, l[k])
	}

	// Check if the annotations exist in the obtained object.
	a, err := obj.GetAnnotations()
	assert.Nil(t, err)
	for k, v := range annotations {
		assert.Equal(t, v, a[k])
	}

	// NOTE: kyaml RNode objects don't have a method to get the owner
	// reference. Perform string comparion for now. It may be possible to do a
	// lookup of the metadata field and get owner reference for comparison.
	assert.True(t, strings.Contains(string(b), ownerRefs))
}
