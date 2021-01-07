package game

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/applier"
	"sigs.k8s.io/kustomize/api/filesys"

	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
)

// ConfigmapOperand implements an operand for ConfigMap.
type ConfigmapOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              filesys.FileSystem
}

var _ operand.Operand = &ConfigmapOperand{}

func (c *ConfigmapOperand) Name() string                             { return c.name }
func (c *ConfigmapOperand) Requires() []string                       { return c.requires }
func (c *ConfigmapOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *ConfigmapOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *ConfigmapOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	game, ok := obj.(*appv1alpha1.Game)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to Game", obj)
	}

	// Create a ManifestTransform with all the transformations and run
	// transforms.
	manifestTransform := transform.ManifestTransform{
		"configmap/configmap.yaml": []transform.TransformFunc{
			transform.AddLabelsFunc(map[string]string{"labelkey1": "labelval1"}),
			transform.SetOwnerReference([]metav1.OwnerReference{ownerRef}),
		},
	}
	if err := transform.Transform(c.fs, manifestTransform); err != nil {
		return nil, fmt.Errorf("error while transforming: %w", err)
	}

	// Mutate kustomization file with object attributes.
	kMutate := []kustomize.MutateFunc{kustomize.AddNamespace(game.GetNamespace())}

	// Run mutation and kustomization to obtain the final manifest.
	m, err := kustomize.MutateAndKustomize(c.fs, kMutate, "configmap")
	if err != nil {
		return nil, fmt.Errorf("error mutating and kustomizing: %w", err)
	}

	// Apply the manifest.
	kubectl := applier.NewDirectApplier()
	if err := kubectl.Apply(ctx, obj.GetNamespace(), string(m), false); err != nil {
		return nil, fmt.Errorf("error applying manifests: %w", err)
	}

	return nil, nil
}

func (c *ConfigmapOperand) Delete(ctx context.Context, obj client.Object) (eventv1.ReconcilerEvent, error) {
	return nil, nil
}

func NewConfigmapOperand(
	name string,
	client client.Client,
	requires []string,
	requeueStrategy operand.RequeueStrategy,
	fs filesys.FileSystem,
) *ConfigmapOperand {
	return &ConfigmapOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
