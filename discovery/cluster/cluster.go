package cluster

import (
	"fmt"

	"github.com/blang/semver/v4"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

// DiscoveryClient is a cluster discovery client.
type DiscoveryClient struct {
	discovery.DiscoveryInterface
}

// New returns a DiscoveryClient given a rest config.
func New(c *rest.Config) (*DiscoveryClient, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(c)
	if err != nil {
		return nil, err
	}
	return &DiscoveryClient{
		DiscoveryInterface: dc,
	}, nil
}

// NewFromDiscoveryClient returns a DiscoveryClient given an implementation of
// the DiscoveryInterface.
func NewFromDiscoveryClient(discoveryClient discovery.DiscoveryInterface) *DiscoveryClient {
	return &DiscoveryClient{
		DiscoveryInterface: discoveryClient,
	}
}

// GetClusterVersion returns the base version of the cluster, without any extra
// information.
func (d *DiscoveryClient) GetClusterVersion() (string, error) {
	version, err := d.ServerVersion()
	if err != nil {
		return "", err
	}

	return basicVersion(version.String())
}

// ClusterVersionCompare compares the cluster version with a given target
// version.
// 0  = cluster version equal to target version
// -1 = cluster version less than target version
// 1  = cluster version greater than target version
func (d *DiscoveryClient) ClusterVersionCompare(targetVersion string) (int, error) {
	// Fetch cluster version and parse it.
	currentVersion, err := d.GetClusterVersion()
	if err != nil {
		return 0, err
	}
	cv, err := semver.Parse(currentVersion)
	if err != nil {
		return 0, err
	}

	// Simplify target version and parse it.
	tvSimple, err := basicVersion(targetVersion)
	if err != nil {
		return 0, err
	}
	tv, err := semver.Parse(tvSimple)
	if err != nil {
		return 0, err
	}

	return cv.Compare(tv), nil
}

// basicVersion parses a given version string and returns only the basic
// version info (Major.Minor.Patch).
func basicVersion(version string) (string, error) {
	ver, err := semver.ParseTolerant(version)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d.%d.%d", ver.Major, ver.Minor, ver.Patch), nil
}

// HasResource takes an API version and a kind of a resource and checks if the
// resource is supported by the API server.
func (d *DiscoveryClient) HasResource(apiVersion, kind string) (bool, error) {
	// Get supported groups and resources API list.
	_, apiLists, err := d.ServerGroupsAndResources()
	if err != nil {
		return false, err
	}
	// Compare the API list with the target resource API version and kind.
	for _, apiList := range apiLists {
		if apiList.GroupVersion == apiVersion {
			for _, r := range apiList.APIResources {
				if r.Kind == kind {
					return true, nil
				}
			}
		}
	}
	return false, nil
}
