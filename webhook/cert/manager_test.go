package cert

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/darkowlzz/operator-toolkit/internal/pkiutil"
)

// getTestResources returns the basic objects required in cert manager tests.
func getTestResources() (
	*corev1.Secret,
	*admissionregistrationv1.MutatingWebhookConfiguration,
	*admissionregistrationv1.ValidatingWebhookConfiguration,
	*apix.CustomResourceDefinition) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-secret",
			Namespace: "default",
		},
	}

	mutatingWebhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-mutating-webhook-config",
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{
			{Name: "foo"},
		},
	}
	validatingWebhookConfig := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-validating-webhook-config",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{Name: "foo"},
		},
	}
	crd := &apix.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "some-custom-resource-definition",
		},
		Spec: apix.CustomResourceDefinitionSpec{
			Conversion: &apix.CustomResourceConversion{
				Webhook: &apix.WebhookConversion{
					ClientConfig: &apix.WebhookClientConfig{},
				},
			},
		},
	}

	return secret, mutatingWebhookConfig, validatingWebhookConfig, crd
}

func TestManager(t *testing.T) {
	// Use this secret when referring to the cert secret. Let the cert manager
	// create it.
	// Create webhook configurations, they must exist for the the cert manager
	// to work.
	secret, mutatingWebhookConfig, validatingWebhookConfig, crd := getTestResources()

	tscheme := scheme.Scheme
	assert.Nil(t, apix.AddToScheme(tscheme))

	// Create a fake client with the webhook configurations.
	cli := fake.NewClientBuilder().WithScheme(tscheme).WithObjects(mutatingWebhookConfig, validatingWebhookConfig, crd).Build()

	certDir, err := ioutil.TempDir("", "cert-test")
	assert.Nil(t, err)
	defer os.RemoveAll(certDir)

	// Configure the certificate manager options.
	certOpts := Options{
		CertDir: certDir,
		Service: &admissionregistrationv1.ServiceReference{
			Name:      "webhook-service",
			Namespace: "default",
		},
		Client:                      cli,
		SecretRef:                   &types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace},
		MutatingWebhookConfigRefs:   []types.NamespacedName{{Name: mutatingWebhookConfig.Name}},
		ValidatingWebhookConfigRefs: []types.NamespacedName{{Name: validatingWebhookConfig.Name}},
		CRDRefs:                     []types.NamespacedName{{Name: crd.Name}},
		CertValidity:                24 * time.Hour,
	}

	// Create a new cert manager.
	certMgr, err := newManager(certOpts)
	assert.Nil(t, err)

	// Start the cert manager.
	assert.Nil(t, certMgr.Start(context.TODO()))

	// Validate the generated cert on host.
	_, _, err = pkiutil.TryLoadCertAndKeyFromDisk(certDir, "tls")
	assert.Nil(t, err)

	// Test various recovery cases handled by cert refresh below.

	// Test case - 1
	// When cert on host does not exist, write the cert on host.

	// Delete the cert on host.
	assert.Nil(t, os.RemoveAll(certDir))
	assert.Nil(t, certMgr.run())
	// Check if cert was written on the host again.
	_, _, err = pkiutil.TryLoadCertAndKeyFromDisk(certDir, "tls")
	assert.Nil(t, err)

	// Test case - 2
	// When secret with cert gets deleted, generate new cert and
	// secret.

	// Delete the secret.
	assert.Nil(t, cli.Delete(context.TODO(), secret))
	assert.Nil(t, certMgr.run())
	// Check if the secret is recreated.
	assert.Nil(t, cli.Get(context.TODO(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, secret))

	// Test case - 3
	// When the CABundle in webhook configurations don't match with the secret
	// cert, update the webhook configuration with the proper CABundle.

	// Empty the CABundle from a webhook configuration.
	assert.Nil(t, cli.Get(context.TODO(), types.NamespacedName{Name: mutatingWebhookConfig.Name}, mutatingWebhookConfig))
	mutatingWebhookConfig.Webhooks[0].ClientConfig.CABundle = []byte{}
	assert.Nil(t, cli.Update(context.TODO(), mutatingWebhookConfig))
	assert.Nil(t, certMgr.run())
	// Check if the CABundle was re-populated.
	assert.Nil(t, cli.Get(context.TODO(), types.NamespacedName{Name: mutatingWebhookConfig.Name}, mutatingWebhookConfig))
	assert.NotEmpty(t, mutatingWebhookConfig.Webhooks[0].ClientConfig.CABundle)
}

func TestMultipleManagers(t *testing.T) {
	// Get the basic resources needed to run the cert manager.
	secret, mutatingWebhookConfig, validatingWebhookConfig, crd := getTestResources()

	tscheme := scheme.Scheme
	assert.Nil(t, apix.AddToScheme(tscheme))

	// Create a fake client with the webhook configurations.
	cli := fake.NewClientBuilder().WithScheme(tscheme).WithObjects(mutatingWebhookConfig, validatingWebhookConfig, crd).Build()

	// Create two cert dirs for the two managers.
	certDir1, err := ioutil.TempDir("", "cert-test")
	assert.Nil(t, err)
	defer os.RemoveAll(certDir1)

	certDir2, err := ioutil.TempDir("", "cert-test")
	assert.Nil(t, err)
	defer os.RemoveAll(certDir2)

	// Configure the certificate manager options.
	certOpts1 := Options{
		CertDir: certDir1,
		Service: &admissionregistrationv1.ServiceReference{
			Name:      "webhook-service",
			Namespace: "default",
		},
		Client:                      cli,
		SecretRef:                   &types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace},
		MutatingWebhookConfigRefs:   []types.NamespacedName{{Name: mutatingWebhookConfig.Name}},
		ValidatingWebhookConfigRefs: []types.NamespacedName{{Name: validatingWebhookConfig.Name}},
		CRDRefs:                     []types.NamespacedName{{Name: crd.Name}},
	}
	// Copy and set the cert dir.
	certOpts2 := certOpts1
	certOpts2.CertDir = certDir2

	// Create new cert managers.
	certMgr1, err := newManager(certOpts1)
	assert.Nil(t, err)
	certMgr2, err := newManager(certOpts2)
	assert.Nil(t, err)

	// Start the cert managers.
	assert.Nil(t, certMgr1.Start(context.TODO()))
	assert.Nil(t, certMgr2.Start(context.TODO()))

	// Compare the certs written by the managers.
	contentCheck := func(path1, path2 string) {
		c1, err := ioutil.ReadFile(path1)
		assert.Nil(t, err)
		c2, err := ioutil.ReadFile(path2)
		assert.Nil(t, err)
		assert.Equal(t, c1, c2)
	}

	contentCheck(filepath.Join(certDir1, defaultCertName), filepath.Join(certDir2, defaultCertName))
	contentCheck(filepath.Join(certDir1, defaultKeyName), filepath.Join(certDir2, defaultKeyName))
}

func TestOptionsSetDefault(t *testing.T) {
	testcases := map[string]struct {
		name      string
		inputOpts Options
		wantOpts  Options
	}{
		"empty": {
			inputOpts: Options{},
			wantOpts: Options{
				Port:                int32(webhook.DefaultPort),
				CertDir:             os.TempDir() + "/k8s-webhook-server/serving-certs",
				CertName:            "tls.crt",
				KeyName:             "tls.key",
				CertRefreshInterval: 30 * time.Minute,
			},
		},
		"custom": {
			inputOpts: Options{
				Port:                int32(2222),
				CertDir:             "/tmp/foo",
				CertName:            "abc.xyz",
				KeyName:             "xyz.abc",
				CertRefreshInterval: 5 * time.Second,
			},
			wantOpts: Options{
				Port:                int32(2222),
				CertDir:             "/tmp/foo",
				CertName:            "abc.xyz",
				KeyName:             "xyz.abc",
				CertRefreshInterval: 5 * time.Second,
			},
		},
	}

	for name, tc := range testcases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			// Take a copy of the input.
			resultOpts := tc.inputOpts

			// Set defaults.
			resultOpts.setDefault()

			// Check the values of defaulted attributes.
			assert.Equal(t, tc.wantOpts.Port, resultOpts.Port, "Port")
			assert.Equal(t, tc.wantOpts.CertDir, resultOpts.CertDir, "CertDir")
			assert.Equal(t, tc.wantOpts.CertName, resultOpts.CertName, "CertName")
			assert.Equal(t, tc.wantOpts.KeyName, resultOpts.KeyName, "KeyName")
			assert.Equal(t, tc.wantOpts.CertRefreshInterval, resultOpts.CertRefreshInterval, "CertRefreshInterval")
			assert.Equal(t, tc.wantOpts.CertValidity, resultOpts.CertValidity, "CertValidity")
		})
	}
}
