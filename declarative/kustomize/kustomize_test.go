package kustomize

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/darkowlzz/operator-toolkit/declarative/loader"
)

func TestKustomize(t *testing.T) {
	kustomization := `resources:
  - guestbook/role.yaml
  - guestbook/service_account.yaml
  - registry/db.yaml
  - registry/frontend.yaml
`

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
---
apiVersion: example.com/v1
kind: DB
metadata:
  name: test-db
---
apiVersion: example.com/v1
kind: Frontend
metadata:
  name: test-frontend
`
	// Create an in-memory filesystem and load the packages in it.
	fs, err := loader.NewLoadedManifestFileSystem("../testdata/channels", "")
	assert.Nil(t, err)

	m, err := Kustomize(fs, []byte(kustomization), nil)
	assert.Nil(t, err)
	assert.Equal(t, wantManifest, string(m))

	// Check if the kustomization file still exists. Reading should fail.
	_, err = fs.ReadFile("kustomization.yaml")
	assert.Error(t, err)
}

func TestAddCommonLabels(t *testing.T) {
	kustomization := `resources:
  - guestbook/role.yaml

commonLabels:
  app: lala
`

	wantCommonLabels := `commonLabels:
  kkk: vvv
  qqqq: iiii
`

	labels := map[string]string{"kkk": "vvv", "qqqq": "iiii"}
	r, err := AddCommonLabels([]byte(kustomization), labels)
	assert.Nil(t, err)
	assert.Contains(t, string(r), wantCommonLabels)
}
