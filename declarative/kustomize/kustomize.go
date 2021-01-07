package kustomize

import (
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/ifc"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/loader"
	apitypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

const kustomizationFile string = "kustomization.yaml"

// Kustomize takes a filesystem and a kustomization configuration and
// runs the kustomization on the filesystem. It returns the result of
// kustomization.
func Kustomize(fs filesys.FileSystem, path string) (result []byte, err error) {
	// Run kustomization.
	opt := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(fs, opt)
	m, rErr := k.Run(path)
	if rErr != nil {
		err = rErr
		return
	}

	return m.AsYaml()
}

// LoadKustomizationFromPackage loads and returns Kustomization object from a
// given package path.
func LoadKustomizationFromPackage(fs filesys.FileSystem, path string) (*apitypes.Kustomization, error) {
	// Check if the path exists.
	if !fs.Exists(path) {
		return nil, fmt.Errorf("%q path not found", path)
	}

	// Create a file loader at the given path.
	ldr, err := loader.NewFileLoaderAtRoot(fs).New(path)
	if err != nil {
		return nil, err
	}

	// Load the kustomization file.
	kbytes, err := loadKustFile(ldr)
	if err != nil {
		return nil, err
	}

	return LoadKustomization(kbytes)
}

// loadKustFile takes a loader and tries to load kustomization file from it.
// Taken from kustomize internal package:
// https://github.com/kubernetes-sigs/kustomize/blob/90f45651d146bb8f8f703d7c06b828e389d8b2c6/api/internal/target/kusttarget.go#L82
func loadKustFile(ldr ifc.Loader) ([]byte, error) {
	var content []byte
	match := 0
	for _, kf := range konfig.RecognizedKustomizationFileNames() {
		c, err := ldr.Load(kf)
		if err == nil {
			match += 1
			content = c
		}
	}
	switch match {
	case 0:
		return nil, fmt.Errorf("unable to find kustomization file in %q", ldr.Root())
	case 1:
		return content, nil
	default:
		return nil, fmt.Errorf(
			"found multiple kustomization files under: %s\n", ldr.Root())
	}
}

// LoadKustomization loads the byte content into a Kustomization object and
// returns it. It also performs some basic validation.
func LoadKustomization(content []byte) (*apitypes.Kustomization, error) {
	k := &apitypes.Kustomization{}
	if err := k.Unmarshal(content); err != nil {
		return nil, err
	}

	k.FixKustomizationPostUnmarshalling()
	errs := k.EnforceFields()
	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to read kustomization: %s", strings.Join(errs, ","))
	}

	return k, nil
}

// WriteKustomizationInPackage takes a filesystem, a kustomization object and a
// package path and writes the kustomization file in the package.
func WriteKustomizationInPackage(fs filesys.FileSystem, k *apitypes.Kustomization, path string) error {
	y, err := yaml.Marshal(k)
	if err != nil {
		return err
	}
	return fs.WriteFile(filepath.Join(path, kustomizationFile), y)
}

// MutateAndKustomize applies all the mutations to a kustomization in a
// package, runs kustomize and returns the result.
func MutateAndKustomize(fs filesys.FileSystem, mutateFuncs []MutateFunc, path string) ([]byte, error) {
	k, err := LoadKustomizationFromPackage(fs, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load kustomization: %w", err)
	}

	// Mutate kustomization file and write in the same package.
	Mutate(k, mutateFuncs)
	if err := WriteKustomizationInPackage(fs, k, path); err != nil {
		return nil, fmt.Errorf("failed to write kustomization: %w", err)
	}

	// Run kustomization in the given package.
	return Kustomize(fs, path)
}
