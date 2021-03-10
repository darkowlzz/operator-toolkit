package declarative

import (
	"context"
	"os"
	"strings"

	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/konfig"

	"github.com/darkowlzz/operator-toolkit/declarative/kubectl"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/loader"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	"github.com/pkg/errors"
)

// Builder helps build resources declaratively and apply or delete them.
type Builder struct {
	// fs is the filesystem containing all the manifest packages.
	fs filesys.FileSystem
	// packageName is the name of the package in the filesystem that contains
	// all the manifests.
	packageName string
	// kubectl is the kubectl client used for applying and deleting the
	// resources.
	kubectl kubectl.KubectlClient
	// manifestTransform is the manifest transforms to perform on the
	// manifests.
	manifestTransform transform.ManifestTransform
	// commonTransforms is a list of transforms to perform on all the manifests
	// in a package.
	commonTransforms []transform.TransformFunc
	// kMutateFuncs are kustomization mutation functions.
	kMutateFuncs []kustomize.MutateFunc
	// manifest is the resource manifest built by the builder.
	manifest string
}

// BuilderOption is used to configure Builder.
type BuilderOption func(*Builder)

// WithKubectlClient sets the kubectl client used by the builder.
func WithKubectlClient(kubectl kubectl.KubectlClient) BuilderOption {
	return func(b *Builder) {
		b.kubectl = kubectl
	}
}

// WithManifestTransform sets the ManifestTransform of the builder.
func WithManifestTransform(manifestTransform transform.ManifestTransform) BuilderOption {
	return func(b *Builder) {
		b.manifestTransform = manifestTransform
	}
}

// WithCommonTransforms sets the common manifest transforms to be used by the
// builder.
func WithCommonTransforms(commonTransforms []transform.TransformFunc) BuilderOption {
	return func(b *Builder) {
		b.commonTransforms = commonTransforms
	}
}

// WithKustomizeMutationFunc sets the kustomization mutation functions.
func WithKustomizeMutationFunc(kMutateFuncs []kustomize.MutateFunc) BuilderOption {
	return func(b *Builder) {
		b.kMutateFuncs = kMutateFuncs
	}
}

// NewBuilder builds a package, given a filesystem and build options and
// returns a builder which can be used to apply or delete the built resource
// manifests.
func NewBuilder(packageName string, fs filesys.FileSystem, opts ...BuilderOption) (*Builder, error) {
	// Create a copy of the filesystem.
	fsCopy := loader.ManifestFileSystem{FileSystem: filesys.MakeFsInMemory()}
	if err := loader.DeepCopy(fs, fsCopy); err != nil {
		return nil, errors.Wrap(err, "failed to create a copy of the filesystem")
	}

	builder := &Builder{
		kubectl:     kubectl.New(),
		fs:          fsCopy,
		packageName: packageName,
	}

	// Apply all the options.
	for _, opt := range opts {
		opt(builder)
	}

	// Apply manifest transforms.
	if builder.manifestTransform != nil && len(builder.manifestTransform) > 0 {
		if err := transform.Transform(builder.fs, builder.manifestTransform); err != nil {
			return nil, errors.Wrapf(err, "failed to transform package %q", builder.packageName)
		}
	}

	// Apply common transforms.
	if len(builder.commonTransforms) > 0 {
		// Load all the non-kustomization files and transform them all with the
		// common transforms.
		mt, err := ManifestTransformForPackage(builder.fs, builder.packageName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get ManifestTransform for package %q", builder.packageName)
		}
		if err := transform.Transform(builder.fs, mt, builder.commonTransforms...); err != nil {
			return nil, errors.Wrapf(err, "failed to transform package %q", builder.packageName)
		}
	}

	// Run mutation and kustomization to obtain the final manifest.
	m, err := kustomize.MutateAndKustomize(builder.fs, builder.kMutateFuncs, builder.packageName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to mutate and kustomization package %q", builder.packageName)
	}
	builder.manifest = string(m)

	return builder, nil
}

// Apply applies the built manifest.
func (b *Builder) Apply(ctx context.Context) error {
	// Skip when the manifest is empty.
	if b.manifest == "" {
		return nil
	}
	return b.kubectl.Apply(ctx, "", b.manifest, true)
}

// Delete deletes the built manifest.
func (b *Builder) Delete(ctx context.Context) error {
	// Skip when the manifest is empty.
	if b.manifest == "" {
		return nil
	}
	return b.kubectl.Delete(ctx, "", b.manifest, true)
}

// Manifest returns the built manifest.
func (b *Builder) Manifest() string {
	return b.manifest
}

// ManifestTransformForPackage returns a ManifestTransform of all the manifests
// in a package.
func ManifestTransformForPackage(fs filesys.FileSystem, packageName string) (transform.ManifestTransform, error) {
	mt := transform.ManifestTransform{}

	err := fs.Walk(packageName, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "failed accessing path %q", path)
		}

		if info.IsDir() {
			return nil
		}

		// Only pick non-kustomization files.
		if !IsKustomization(path) {
			mt[path] = []transform.TransformFunc{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return mt, nil
}

// isKustomization takes a path and returns true if the path is a kustomization
// file.
func IsKustomization(path string) bool {
	for _, name := range konfig.RecognizedKustomizationFileNames() {
		if strings.HasSuffix(path, name) {
			return true
		}
	}
	return false
}
