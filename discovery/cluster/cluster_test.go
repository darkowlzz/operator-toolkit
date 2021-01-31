package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclient "k8s.io/client-go/kubernetes/fake"
)

func TestClusterVersionCompare(t *testing.T) {
	cases := []struct {
		name           string
		currentVersion string
		targetVersion  string
		wantResult     int
	}{
		{
			name:           "equal",
			currentVersion: "v1.20.0",
			targetVersion:  "v1.20.0",
			wantResult:     0,
		},
		{
			name:           "greater than",
			currentVersion: "v1.21.0",
			targetVersion:  "v1.20.0",
			wantResult:     1,
		},
		{
			name:           "less than",
			currentVersion: "v1.20.0",
			targetVersion:  "v1.21.0",
			wantResult:     -1,
		},
		{
			name:           "versions with suffix",
			currentVersion: "v1.20.0-gke-xyz",
			targetVersion:  "v1.20.0-foo",
			wantResult:     0,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Setup fake client.
			clientset := fakeclient.NewSimpleClientset()
			fdiscovery, ok := clientset.Discovery().(*fakediscovery.FakeDiscovery)
			assert.True(t, ok)

			// Set cluster version.
			fdiscovery.FakedServerVersion = &version.Info{GitVersion: tc.currentVersion}
			dc := NewFromDiscoveryClient(fdiscovery)

			// Compare and check result.
			res, err := dc.ClusterVersionCompare(tc.targetVersion)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantResult, res)
		})
	}
}

func TestHasResource(t *testing.T) {
	testKind := "Foo"
	testKindGroupVersion := "v1"

	cases := []struct {
		name            string
		apiResourceList []*metav1.APIResourceList
		wantResult      bool
	}{
		{
			name:            "empty api resource list",
			apiResourceList: []*metav1.APIResourceList{},
			wantResult:      false,
		},
		{
			name: "have resource",
			apiResourceList: []*metav1.APIResourceList{
				{
					GroupVersion: "v1",
					APIResources: []metav1.APIResource{
						{Kind: "Foo"},
						{Kind: "Bar"},
					},
				},
			},
			wantResult: true,
		},
		{
			name: "have resource but different version",
			apiResourceList: []*metav1.APIResourceList{
				{
					GroupVersion: "v2",
					APIResources: []metav1.APIResource{
						{Kind: "Foo"},
						{Kind: "Bar"},
					},
				},
			},
			wantResult: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Setup fake client with the API resource list.
			client := fakeclient.NewSimpleClientset()
			client.Resources = tc.apiResourceList

			// Create discovery client and check if resource exists.
			dc := NewFromDiscoveryClient(client.Discovery())
			exists, err := dc.HasResource(testKindGroupVersion, testKind)
			assert.Nil(t, err)
			assert.Equal(t, tc.wantResult, exists)
		})
	}
}
