package transform

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/darkowlzz/composite-reconciler/declarative/loader"
)

func TestReplicaTransform(t *testing.T) {
	// Create an in-memory filesystem and load the packages in it.
	fs := loader.ManifestFileSystem{filesys.MakeFsInMemory()}
	err := loader.LoadPackages(fs, "../testdata/channels", "")
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
	fs := loader.ManifestFileSystem{filesys.MakeFsInMemory()}
	err := loader.LoadPackages(fs, "../testdata/channels", "")
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

	// Run transform.
	err = Transform(fs, manifestTransform)
	assert.Nil(t, err)

	checkLabelsAndAnnotations(t, fs, targetFileA, labels, annotations)
	checkLabelsAndAnnotations(t, fs, targetFileB, labels, nil)
}

func checkLabelsAndAnnotations(t *testing.T, fs loader.ManifestFileSystem, file string, labels, annotations map[string]string) {
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
}
