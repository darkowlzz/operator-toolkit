package declarative

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
)

func TestNewBuilder(t *testing.T) {
	labels := map[string]string{"testkey": "testval"}
	labels2 := map[string]string{"testkey2": "testval2"}
	annotations := map[string]string{"annokey": "annoval"}

	cases := []struct {
		name          string
		builderOption []BuilderOption
		wantManifest  string
	}{
		{
			name: "no transformation",
			wantManifest: `apiVersion: rbac.authorization.k8s.io/v1
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
`,
		},
		{
			name: "manifest transform",
			builderOption: []BuilderOption{
				WithManifestTransform(transform.ManifestTransform{
					"guestbook/role.yaml": []transform.TransformFunc{transform.AddLabelsFunc(labels)},
				}),
			},
			wantManifest: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  annotations:
    foo1: bar1
  labels:
    foo: bar
    testkey: testval
  name: app-role
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-sa
`,
		},
		{
			name: "common transform",
			builderOption: []BuilderOption{
				WithCommonTransforms([]transform.TransformFunc{
					transform.AddAnnotationsFunc(annotations),
				}),
			},
			wantManifest: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  annotations:
    annokey: annoval
    foo1: bar1
  labels:
    foo: bar
  name: app-role
---
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    annokey: annoval
  name: test-sa
`,
		},
		{
			name: "kustomization transform",
			builderOption: []BuilderOption{
				WithKustomizeMutationFunc([]kustomize.MutateFunc{
					kustomize.AddCommonLabels(labels),
				}),
			},
			wantManifest: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  annotations:
    foo1: bar1
  labels:
    foo: bar
    testkey: testval
  name: app-role
---
apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    testkey: testval
  name: test-sa
`,
		},
		{
			name: "manifest, common and kustomization transform",
			builderOption: []BuilderOption{
				WithManifestTransform(transform.ManifestTransform{
					"guestbook/role.yaml": []transform.TransformFunc{transform.AddLabelsFunc(labels)},
				}),
				WithCommonTransforms([]transform.TransformFunc{
					transform.AddAnnotationsFunc(annotations),
				}),
				WithKustomizeMutationFunc([]kustomize.MutateFunc{
					kustomize.AddCommonLabels(labels2),
				}),
			},
			wantManifest: `apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  annotations:
    annokey: annoval
    foo1: bar1
  labels:
    foo: bar
    testkey: testval
    testkey2: testval2
  name: app-role
---
apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    annokey: annoval
  labels:
    testkey2: testval2
  name: test-sa
`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			fs, err := loader.NewLoadedManifestFileSystem("testdata/channels", "")
			assert.Nil(t, err)

			b, err := NewBuilder("guestbook", fs, tc.builderOption...)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantManifest, b.Manifest())
		})
	}
}

func TestManifestTransformForPackage(t *testing.T) {
	fs, err := loader.NewLoadedManifestFileSystem("testdata/channels", "")
	assert.Nil(t, err)

	wantManifests := []string{"/registry/db.yaml", "/registry/frontend.yaml"}

	mt, err := ManifestTransformForPackage(fs, "registry")
	assert.Nil(t, err)

	for _, manifest := range wantManifests {
		assert.True(t, manifestFound(manifest, mt))
	}
}

func manifestFound(fileName string, mt transform.ManifestTransform) bool {
	if _, exists := mt[fileName]; exists {
		return true
	}
	return false
}
