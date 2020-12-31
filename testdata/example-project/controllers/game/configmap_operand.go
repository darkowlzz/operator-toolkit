package game

import (
	"context"
	"fmt"
	"html/template"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative/pkg/applier"

	"github.com/darkowlzz/composite-reconciler/declarative/kustomize"
	"github.com/darkowlzz/composite-reconciler/declarative/loader"
	"github.com/darkowlzz/composite-reconciler/declarative/transform"
	eventv1 "github.com/darkowlzz/composite-reconciler/event/v1"
	"github.com/darkowlzz/composite-reconciler/operator/v1/operand"
)

// configmapTemplateParams is used to store the data used to populate the
// kustomization template.
type configmapTemplateParams struct {
	Namespace string
}

const configmapTemplate = `
namespace: {{.Namespace}}

resources:
  - configmap/configmap.yaml
`

// ConfigmapOperand implements an operand for ConfigMap.
type ConfigmapOperand struct {
	name            string
	client          client.Client
	requires        []string
	requeueStrategy operand.RequeueStrategy
	fs              *loader.ManifestFileSystem
}

var _ operand.Operand = &ConfigmapOperand{}

func (c *ConfigmapOperand) Name() string                             { return c.name }
func (c *ConfigmapOperand) Requires() []string                       { return c.requires }
func (c *ConfigmapOperand) RequeueStrategy() operand.RequeueStrategy { return c.requeueStrategy }
func (c *ConfigmapOperand) ReadyCheck(ctx context.Context, obj client.Object) (bool, error) {
	return true, nil
}

func (c *ConfigmapOperand) Ensure(ctx context.Context, obj client.Object, ownerRef metav1.OwnerReference) (eventv1.ReconcilerEvent, error) {
	// Populate the configmap template params.
	templateParams := configmapTemplateParams{Namespace: obj.GetNamespace()}

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

	// TODO: Move this template rendering into a helper function.
	// Render the kustomization template.
	var kResult strings.Builder
	tmpl, err := template.New("configmap").Parse(configmapTemplate)
	if err != nil {
		return nil, fmt.Errorf("error parsing template: %w", err)
	}
	if err := tmpl.Execute(&kResult, templateParams); err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	// Run kustomization with the template to obtain the final manifest.
	m, err := kustomize.Kustomize(c.fs, []byte(kResult.String()))
	if err != nil {
		return nil, fmt.Errorf("error kustomizing: %w", err)
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
	fs *loader.ManifestFileSystem,
) *ConfigmapOperand {
	return &ConfigmapOperand{
		name:            name,
		client:          client,
		requires:        requires,
		requeueStrategy: requeueStrategy,
		fs:              fs,
	}
}
