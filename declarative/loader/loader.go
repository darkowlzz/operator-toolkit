package loader

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/loaders"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/yaml"
)

const (
	// DefaultChannelDir is the default name of the directory containing
	// channel and packages files.
	DefaultChannelDir = "channels"
	// DefaultChannelName is the default name of a channel.
	DefaultChannelName = "stable"
)

// NewLoadedManifestFileSystem creates and returns a new ManifestFileSystem,
// loaded with the manifests.
func NewLoadedManifestFileSystem(baseDir string, channel string) (*ManifestFileSystem, error) {
	fs := &ManifestFileSystem{FileSystem: filesys.MakeFsInMemory()}

	// Use default baseDir and channel if not provided.
	if baseDir == "" {
		baseDir = DefaultChannelDir
	}
	if channel == "" {
		channel = DefaultChannelName
	}
	err := LoadPackages(fs, baseDir, channel)
	if err != nil {
		return nil, fmt.Errorf("failed to load channel packages: %w", err)
	}

	return fs, nil
}

// LoadPackages takes a filesystem, a base directory path and a channel name,
// and loads the versioned packages specified in the channel into the
// filesystem. The channel is based on a Channel type defined in the
// kubebuilder-declarative-pattern addon loaders package.
// Example channel:
//
// manifests:
// - name: guestbook
//   version: 0.1.0
// - name: registry
//   version: 0.3.0
//
func LoadPackages(fs *ManifestFileSystem, baseDir string, channel string) error {

	if baseDir == "" {
		baseDir = DefaultChannelDir
	}

	if channel == "" {
		channel = DefaultChannelName
	}

	// Read the channel.
	p := filepath.Join(baseDir, channel)
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", p, err)
	}
	ch := &loaders.Channel{}
	if err := yaml.Unmarshal(b, ch); err != nil {
		return fmt.Errorf("failed to unmarshal channel %q: %w", p, err)
	}

	// Load the manifests in the channel into the filesystem.
	if err := LoadManifests(fs, baseDir, ch); err != nil {
		return err
	}

	return nil
}

// LoadManifests takes a filesystem, a base directory path that contains the
// manifests and a Channel, and loads the manifests into the filesystem.
// The copied package manifests are copied at the root of the filesystem in a
// directory named the same as the package name. These manifests are not
// versioned.
// Example filesystem structure once the manifests are loaded:
//
// /
// |-- guestbook
// |   |-- role.yaml
// |   |-- service_account.yaml
// |
// |-- registry
//     |-- db.yaml
//     |-- frontend.yaml
//
func LoadManifests(fs *ManifestFileSystem, baseDir string, channel *loaders.Channel) error {
	for _, manifest := range channel.Manifests {
		packagePath := filepath.Join(baseDir, "packages", manifest.Package, manifest.Version)
		if err := fs.CopyDirectory(packagePath, manifest.Package); err != nil {
			return fmt.Errorf("failed to load manifests at %q: %w", packagePath, err)
		}
	}
	return nil
}
