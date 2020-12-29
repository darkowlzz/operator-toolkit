package kustomize

import (
	"sigs.k8s.io/kustomize/api/krusty"

	"github.com/darkowlzz/composite-reconciler/declarative/loader"
)

const kustomizationFile string = "kustomization.yaml"

// Kustomize takes a filesystem and a kustomization configuration and
// runs the kustomization on the filesystem. It returns the result of
// kustomization.
func Kustomize(fs loader.ManifestFileSystem, kustom []byte) (result []byte, err error) {
	// Create a kustomization file with the given content.
	if fErr := fs.WriteFile(kustomizationFile, kustom); fErr != nil {
		err = fErr
		return
	}

	// Remove the kustomization file at the end.
	defer func() {
		if rmErr := fs.RemoveAll(kustomizationFile); err != nil {
			err = rmErr
		}
	}()

	// Run kustomization.
	opt := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(fs, opt)
	m, rErr := k.Run("/")
	if rErr != nil {
		err = rErr
		return
	}

	return m.AsYaml()
}
