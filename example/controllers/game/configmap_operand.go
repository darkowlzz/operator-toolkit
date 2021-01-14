package game

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kustomize/api/filesys"

	"github.com/darkowlzz/operator-toolkit/declarative"
	"github.com/darkowlzz/operator-toolkit/declarative/kustomize"
	"github.com/darkowlzz/operator-toolkit/declarative/transform"
	eventv1 "github.com/darkowlzz/operator-toolkit/event/v1"
	appv1alpha1 "github.com/darkowlzz/operator-toolkit/example/api/v1alpha1"
	"github.com/darkowlzz/operator-toolkit/operator/v1/operand"
)

// manifestPackage is the name of the package that contains manifests for the
// operand.
const manifestPackage = "configmap"

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
	// Setup a tracer and start a span.
	tr := otel.Tracer("ConfigmapOperand")
	ctx, span := tr.Start(ctx, "configmap")
	defer span.End()

	game, ok := obj.(*appv1alpha1.Game)
	if !ok {
		return nil, fmt.Errorf("failed to convert %v to Game", obj)
	}

	// Create a builder with all the transformations.
	b, err := declarative.NewBuilder(manifestPackage, c.fs,
		declarative.WithManifestTransform(transform.ManifestTransform{
			"configmap/configmap.yaml": []transform.TransformFunc{
				transform.AddLabelsFunc(map[string]string{"labelkey1": "labelval1"}),
			},
		}),
		declarative.WithCommonTransforms([]transform.TransformFunc{
			transform.SetOwnerReference([]metav1.OwnerReference{ownerRef}),
		}),
		declarative.WithKustomizeMutationFunc([]kustomize.MutateFunc{
			kustomize.AddNamespace(game.GetNamespace()),
		}),
	)
	if err != nil {
		return nil, err
	}

	return nil, b.Apply(ctx)
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
