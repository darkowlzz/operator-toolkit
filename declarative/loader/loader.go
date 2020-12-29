package loader

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
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
func LoadPackages(fs ManifestFileSystem, baseDir string, channel string) error {
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
func LoadManifests(fs ManifestFileSystem, baseDir string, channel *loaders.Channel) error {
	for _, manifest := range channel.Manifests {
		packagePath := filepath.Join(baseDir, "packages", manifest.Package, manifest.Version)
		if err := fs.CopyDirectory(packagePath, manifest.Package); err != nil {
			return fmt.Errorf("failed to load manifests at %q: %w", packagePath, err)
		}
	}
	return nil
}

// ManifestFileSystem is a wrapper around the kustomize filesys.Filesystem with
// methods to copy all the files and directories of a manifest directory. This
// can be backed by a disk or an in-memory filesystem.
type ManifestFileSystem struct {
	filesys.FileSystem
}

// CopyDirectory recursively copies directory content from disk filesystem to
// the manifest filesystem.
func (mfs *ManifestFileSystem) CopyDirectory(srcDir, dest string) error {
	entries, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		sourcePath := filepath.Join(srcDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := mfs.CreateIfNotExists(destPath); err != nil {
				return err
			}
			if err := mfs.CopyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			// TODO: Resolve and handle symlinks.
		default:
			if err := mfs.Copy(sourcePath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copy reads a file from disk and copies it into manifest filesystem.
func (mfs *ManifestFileSystem) Copy(srcFile, dstFile string) (err error) {
	out, cErr := mfs.Create(dstFile)
	if cErr != nil {
		err = cErr
		return
	}

	defer func() {
		if cErr := out.Close(); cErr != nil {
			err = cErr
		}
	}()

	in, oErr := os.Open(srcFile)
	defer func() {
		if cErr := in.Close(); cErr != nil {
			err = cErr
		}
	}()
	if oErr != nil {
		err = oErr
		return
	}

	_, cErr = io.Copy(out, in)
	if cErr != nil {
		err = cErr
		return
	}

	return
}

// CreateIfNotExists creates a path if not exists.
func (mfs *ManifestFileSystem) CreateIfNotExists(dir string) error {
	if mfs.Exists(dir) {
		return nil
	}

	if err := mfs.MkdirAll(dir); err != nil {
		return fmt.Errorf("failed to create directory: %q, error: %v", dir, err)
	}

	return nil
}
