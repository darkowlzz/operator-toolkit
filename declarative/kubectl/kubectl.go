package kubectl

import (
	"context"

	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/applier"

	"github.com/darkowlzz/operator-toolkit/declarative/deleter"
)

// KubectlClient defines an interface for a kubernetes client that can be used
// for applying and deleting resources declaratively.
type KubectlClient interface {
	Apply(ctx context.Context, namespace string, manifest string, validate bool, extraArgs ...string) error
	Delete(ctx context.Context, namespace string, manifest string, validate bool, extraArgs ...string) error
}

// DefaultKubectl is the default implementation of the KubectlClient using
// direct applier and deleter.
type DefaultKubectl struct {
	*applier.DirectApplier
	*deleter.DirectDeleter
}

// New returns a new KubectlClient based on direct applier and deleter.
func New() *DefaultKubectl {
	return &DefaultKubectl{
		DirectApplier: applier.NewDirectApplier(),
		DirectDeleter: deleter.NewDirectDeleter(),
	}
}
