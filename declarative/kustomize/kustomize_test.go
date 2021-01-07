package kustomize

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/darkowlzz/operator-toolkit/declarative/loader"
)

func TestKustomize(t *testing.T) {
	wantManifest := `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  annotations:
    foo1: bar1
  labels:
    foo: bar
  name: app-role
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-sa
`
	// Create an in-memory filesystem and load the packages in it.
	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	m, err := Kustomize(fs, "guestbook")
	assert.Nil(t, err)
	assert.Equal(t, wantManifest, string(m))

	// Check if the kustomization file still exists. Reading should fail.
	_, err = fs.ReadFile("kustomization.yaml")
	assert.Error(t, err)
}
