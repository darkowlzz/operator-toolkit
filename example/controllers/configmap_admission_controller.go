package controllers

import (
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	tkadmission "github.com/darkowlzz/operator-toolkit/webhook/admission"
	"github.com/darkowlzz/operator-toolkit/webhook/builder"
	"github.com/darkowlzz/operator-toolkit/webhook/function"
)

type ConfigMapAdmissionController struct {
	CtrlName        string
	Log             logr.Logger
	Client          client.Client
	DiscoveryClient discovery.DiscoveryInterface
}

var _ tkadmission.Controller = &ConfigMapAdmissionController{}

var validLabels = map[string]string{
	"valid-key-1": "some-val1",
	"valid-key-2": "some-val2",
	"foo":         "baz",
}

func NewConfigMapAdmissionController(name string, cli client.Client, dc discovery.DiscoveryInterface, log logr.Logger) *ConfigMapAdmissionController {
	return &ConfigMapAdmissionController{
		CtrlName:        name,
		Log:             log,
		Client:          cli,
		DiscoveryClient: dc,
	}
}

func (cmac *ConfigMapAdmissionController) Name() string {
	return cmac.CtrlName
}

func (cmac *ConfigMapAdmissionController) GetNewObject() client.Object {
	return &corev1.ConfigMap{}
}

func (cmac *ConfigMapAdmissionController) RequireDefaulting(obj client.Object) bool {
	// Perform any relevant checks to determine if the object should be
	// defaulted or ignored.
	return true
}

func (cmac *ConfigMapAdmissionController) RequireValidating(obj client.Object) bool {
	// Perform any relevant checks to determine if the object should be
	// validated or ignored.
	return true
}

func (cmac *ConfigMapAdmissionController) Default() []tkadmission.DefaultFunc {
	return []tkadmission.DefaultFunc{
		function.AddLabels(cmac.Client, map[string]string{"foo": "bar"}),
		function.AddAnnotations(cmac.Client, map[string]string{"zzz": "qqqq"}),
		function.AddClusterVersionAnnotation(cmac.DiscoveryClient),
	}
}

func (cmac *ConfigMapAdmissionController) ValidateCreate() []tkadmission.ValidateCreateFunc {
	return []tkadmission.ValidateCreateFunc{
		function.ValidateLabelsCreate(validLabels),
	}
}

func (cmac *ConfigMapAdmissionController) ValidateUpdate() []tkadmission.ValidateUpdateFunc {
	return []tkadmission.ValidateUpdateFunc{
		function.ValidateLabelsUpdate(validLabels),
	}
}

func (cmac *ConfigMapAdmissionController) ValidateDelete() []tkadmission.ValidateDeleteFunc {
	return []tkadmission.ValidateDeleteFunc{}
}

func (cmac *ConfigMapAdmissionController) SetupWithManager(mgr manager.Manager) error {
	return builder.WebhookManagedBy(mgr).
		MutatePath("/mutate-configmap").
		ValidatePath("/validate-configmap").
		Complete(cmac)
}
