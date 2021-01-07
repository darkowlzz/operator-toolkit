package declarative

import (
	"context"
	"fmt"

	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/applier"
	"sigs.k8s.io/kustomize/api/filesys"

	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
)

// BuildAndApply runs manifest transform, kustomization build and applies
// the result.
func BuildAndApply(
	ctx context.Context,
	fs filesys.FileSystem,
	path string,
	namespace string,
	manifestTransform transform.ManifestTransform,
	kMutateFuncs []kustomize.MutateFunc) error {
	// Run manifest transforms.
	if err := transform.Transform(fs, manifestTransform); err != nil {
		return fmt.Errorf("error while transforming: %w", err)
	}

	// Run mutation and kustomization to obtain the final manifest.
	m, err := kustomize.MutateAndKustomize(fs, kMutateFuncs, path)
	if err != nil {
		return fmt.Errorf("error mutating and kustomizing: %w", err)
	}

	// Apply the manifest.
	kubectl := applier.NewDirectApplier()
	return kubectl.Apply(ctx, namespace, string(m), false)
}
