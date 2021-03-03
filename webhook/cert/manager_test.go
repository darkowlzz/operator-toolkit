package cert

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/darkowlzz/operator-toolkit/internal/pkiutil"
)

func TestManager(t *testing.T) {
	// Use this secret when referring to the cert secret. Let the cert manager
	// create it.
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-secret",
			Namespace: "default",
		},
	}

	// Create webhook configurations to be managed by the cert manager.
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

	// Create a fake client with the webhook configurations.
	cli := fake.NewFakeClient(mutatingWebhookConfig, validatingWebhookConfig)

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
