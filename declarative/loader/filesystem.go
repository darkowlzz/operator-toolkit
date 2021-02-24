package loader

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/kustomize/api/filesys"
)

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

// DeepCopy performs a deep copy of the FileSystem from fs1 to fs2.
func DeepCopy(fs1, fs2 filesys.FileSystem) error {
	walkErr := fs1.Walk("/", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			// Create the directory in copy target.
			if err := fs2.MkdirAll(path); err != nil {
				return err
			}
		} else {
			// Read the file from source and write it in the destination.
			content, err := fs1.ReadFile(path)
			if err != nil {
				return err
			}
			if err := fs2.WriteFile(path, content); err != nil {
				return err
			}
		}

		return nil
	})
	if walkErr != nil {
		return walkErr
	}
	return nil
}
