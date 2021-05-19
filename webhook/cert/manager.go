package cert

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	webhookcert "github.com/darkowlzz/operator-toolkit/internal/webhook/cert"
	"github.com/darkowlzz/operator-toolkit/internal/webhook/cert/generator"
	"github.com/darkowlzz/operator-toolkit/internal/webhook/cert/writer"
)

var log = ctrl.Log.WithName("webhook").WithName("cert").WithName("manager")

// Short refresh interval by default to reduce the impact of out of sync
// instances of the certificate manager.
var defaultCertRefreshInterval = 30 * time.Minute

const (
	defaultCertName = "tls.crt"
	defaultKeyName  = "tls.key"
)

// Manager is a webhook server certificate manager. It needs to know
// about the webhook configuration and service or host of the webhook in order
// to provision self signed certificate and inject the cert into the webhook
// configurations. The generated certificate is stored in a k8s secret object
// and is reused if it already exists.
type Manager struct {
	// Option is the certificate provisioner options.
	Options

	// certProvisioner is the certificate provisioner.
	certProvisioner webhookcert.Provisioner
}

// Options are options for the certificate Manager.
type Options struct {
	// CertRefreshInterval is the interval at which the cert is refreshed.
	CertRefreshInterval time.Duration

	// Service is a reference to the k8s service fronting the webhook server
	// pod(s). This field is optional. But one and only one of Service and
	// Host need to be set.
	// This maps to field .webhooks.getClientConfig.service
	Service *admissionregistrationv1.ServiceReference

	// Host is the host name of .webhooks.clientConfig.url
	// This field is optional. But one and only one of Service and Host need to be set.
	Host *string

	// Port is the port number that the server will serve.
	// It will be defaulted to controller-runtime's default webhook server port
	// if unspecified.
	Port int32

	// MutatingWebhookConfigRefs is the reference to mutating webhook
	// configurations to update with the provisioned certificate.
	MutatingWebhookConfigRefs []types.NamespacedName

	// ValidatingWebhookConfigRefs is the reference to validating webhook
	// configurations to update with the provisioned certificate.
	ValidatingWebhookConfigRefs []types.NamespacedName

	// Client is a k8s client.
	Client client.Client

	// CertWriter is a certificate writer.
	CertWriter writer.CertWriter

	// SecretRef is a reference to the secret where the generated secret is
	// stored for persistence.
	SecretRef *types.NamespacedName

	// CertDir is the directory that contains the server key and certificate. The
	// server key and certificate.
	CertDir string

	// CertName is the server certificate name. Defaults to tls.crt.
	CertName string

	// KeyName is the server key name. Defaults to tls.key.
	KeyName string

	// CertValidity is the length of the generated certificate's validity. This is not
	// the validity of the root CA cert. That's set to 10 years by default in
	// the client-go cert utils package.
	// If not set, this defaults to a year.
	CertValidity time.Duration
}

// setDefault sets the default options.
func (o *Options) setDefault() {
	if o.Port <= 0 {
		o.Port = int32(webhook.DefaultPort)
	}

	if len(o.CertDir) == 0 {
		o.CertDir = filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs")
	}

	if len(o.CertName) == 0 {
		o.CertName = defaultCertName
	}

	if len(o.KeyName) == 0 {
		o.KeyName = defaultKeyName
	}

	if o.CertRefreshInterval == 0*time.Second {
		o.CertRefreshInterval = defaultCertRefreshInterval
	}
}

// NewManager creates a certificate manager managed by the controller manager.
// If the manager is nil, the manager is started independently, unmanaged.
func NewManager(mgr manager.Manager, ops Options) error {
	certManager, err := newManager(ops)
	if err != nil {
		return err
	}
	// If a manager is provided, add the certificate manager to the manager,
	// else start the cert manager immediately.
	if mgr != nil {
		return mgr.Add(certManager)
	}

	return certManager.Start(context.Background())
}

// newManager exists separately to help with testing. Since NewManager does not
// returns the cert manager, newManager can be used to get the cert manager
// instance for testing and calling its methods.
func newManager(ops Options) (*Manager, error) {
	ops.setDefault()

	// If CertWriter is not set, create a default CertWriter.
	if ops.CertWriter == nil {
		secretCWOpts := writer.SecretCertWriterOptions{
			Client: ops.Client,
			CertGenerator: &generator.SelfSignedCertGenerator{
				Validity: ops.CertValidity,
			},
			Secret: ops.SecretRef,
		}
		cw, err := writer.NewSecretCertWriter(secretCWOpts)
		if err != nil {
			return nil, err
		}
		ops.CertWriter = cw
	}

	certManager := &Manager{
		certProvisioner: webhookcert.Provisioner{CertWriter: ops.CertWriter},
		Options:         ops,
	}

	return certManager, nil
}

// NeedLeaderElection implements the LeaderElectionRunnable interface.
func (m *Manager) NeedLeaderElection() bool {
	return false
}

// certExists checks if cert already exist that are not managed by certificate
// manager.
func (m *Manager) certExists() bool {
	_, err := os.Stat(filepath.Join(m.CertDir, m.CertName))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(err, "error checking server cert")
		}
		return false
	}

	_, err = os.Stat(filepath.Join(m.CertDir, m.KeyName))
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error(err, "error checking server key")
		}
		return false
	}

	return true
}

// provision implements the Runnable interface. It starts the certificate
// manager.
func (m *Manager) Start(ctx context.Context) error {
	// If cert already exists, skip. Certificate is manager by another
	// certificate manager.
	if m.certExists() {
		log.Info("existing certs found, skipping self signed certificate manager")
		return nil
	}

	log.Info("starting certificate manager to manage webhook server certificate")

	// Ensure certificate at startup.
	if err := m.run(); err != nil {
		return err
	}

	go func() {
		// Refresh certs at refresh interval.
		ticker := time.NewTicker(wait.Jitter(m.CertRefreshInterval, 0.1))
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Info("stopping cert manager")
				return
			case <-ticker.C:
				log.Info("cert refresh check")
				if err := m.run(); err != nil {
					log.Error(err, "failed to run cert provisioner")
				}
			}
		}
	}()

	return nil
}

// run ensures that a valid certificate exists and upon certificate update, it
// updates the certificate on the host.
func (m *Manager) run() error {
	needHostCertUpdate := false

	// Check if the certs exist on the host.
	if !m.certExists() {
		log.Info("cert not found on host")
		needHostCertUpdate = true
	}

	// Refresh existing cert in secret if needed.
	ctx := context.Background()
	changed, err := m.refreshCert(ctx)
	if err != nil {
		return err
	}
	if changed {
		log.Info("generated new cert")
	}

	// Update the cert on host.
	if changed || needHostCertUpdate {
		log.Info(fmt.Sprintf("updating the cert in %s", m.CertDir))
		return m.writeCertOnDisk(ctx)
	}

	return nil
}

func (m *Manager) writeCertOnDisk(ctx context.Context) error {
	// Get the cert and write on disk.
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
	}
	err := m.Client.Get(ctx, *m.SecretRef, secret)
	if apierrors.IsNotFound(err) {
		return err
	}
	cert := secret.Data[writer.ServerCertName]
	key := secret.Data[writer.ServerKeyName]

	if err := os.MkdirAll(m.CertDir, 0700); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(m.CertDir, m.CertName), cert, 0666); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(m.CertDir, m.KeyName), key, 0666); err != nil {
		return err
	}

	return nil
}

// refreshCert refreshes the certificate using cert provisioner if the
// certificate is expiring. It also updates the webhook configurations with the
// current certificate. The caller can decide to reload the webhook server
// when the cert changes.
func (m *Manager) refreshCert(ctx context.Context) (bool, error) {
	cc, err := m.getClientConfig()
	if err != nil {
		return false, err
	}

	// Fetch the webhook configurations.
	whConfigs := []client.Object{}

	// NOTE: Since the webhook configurations managed by certificate manager
	// are for the same webhook server, they all will have the same CABundle.
	// Get a CABundle from any webhook configuration and ensure all the
	// webhook configurations have the same CABundle. If not, pass an empty
	// CABundle to the provisioner to signal a cert change that needs to be
	// propagated to all the webhook configurations.
	var caBundle []byte
	var differentCABundles bool

	for _, nn := range m.MutatingWebhookConfigRefs {
		mwc := &admissionregistrationv1.MutatingWebhookConfiguration{}
		if err := m.Client.Get(ctx, nn, mwc); err != nil {
			return false, err
		}
		whConfigs = append(whConfigs, mwc)

		// Ensure CABundles are equal. Skip comparison once differentCABundles
		// is true.
		for _, wh := range mwc.Webhooks {
			if differentCABundles {
				break
			}
			caBundle, differentCABundles = compareCABundles(caBundle, wh.ClientConfig.CABundle)
		}
	}

	for _, nn := range m.ValidatingWebhookConfigRefs {
		vwc := &admissionregistrationv1.ValidatingWebhookConfiguration{}
		if err := m.Client.Get(ctx, nn, vwc); err != nil {
			return false, err
		}
		whConfigs = append(whConfigs, vwc)

		// Ensure CABundles are equal. Skip comparison once differentCABundles
		// is true.
		for _, wh := range vwc.Webhooks {
			if differentCABundles {
				break
			}
			caBundle, differentCABundles = compareCABundles(caBundle, wh.ClientConfig.CABundle)
		}
	}

	// Set the determined CABundle.
	cc.CABundle = caBundle

	// Ensure cert.
	changed, err := m.certProvisioner.Provision(ctx, webhookcert.Options{
		ClientConfig: cc,
		Objects:      whConfigs,
	})
	if err != nil {
		return false, err
	}

	// Update the webhook configurations.
	return changed, batchUpdate(ctx, m.Client, whConfigs...)
}

// compareCABundles compares common CABundle with a given webhook's CABundle.
// If the common CABundle is empty, set it to the webhook's CABundle.
// On difference in the CABundle, return false as differentCABundles return
// argument. This comparison is needed to identify any client cert
// misconfiguration in any of the webhook configurations.
func compareCABundles(commonCABundle []byte, caBundle []byte) ([]byte, bool) {
	if len(commonCABundle) == 0 {
		commonCABundle = caBundle
	}
	// Compare the byte slices. If unequal, CABundles are different. Return
	// empty slice and true differentCABundles.
	res := bytes.Compare(commonCABundle, caBundle)
	if res != 0 {
		return []byte{}, true
	}
	return commonCABundle, false
}

// batchUpdate updates all the given objects.
func batchUpdate(ctx context.Context, c client.Client, objs ...client.Object) error {
	for _, obj := range objs {
		if err := c.Update(ctx, obj); err != nil {
			return err
		}
	}
	return nil
}

// getClientConfig returns a WebhookClientConfig with the provided host or
// service of the webhook server.
func (m *Manager) getClientConfig() (*admissionregistrationv1.WebhookClientConfig, error) {
	if m.Host != nil && m.Service != nil {
		return nil, errors.New("URL and Service can't be set at the same time")
	}
	// Create a webhook client config with empty CA bundle.
	cc := &admissionregistrationv1.WebhookClientConfig{
		CABundle: []byte{},
	}
	// Set the host or service of the server.
	if m.Host != nil {
		u := url.URL{
			Scheme: "https",
			Host:   net.JoinHostPort(*m.Host, strconv.Itoa(int(m.Port))),
		}
		urlString := u.String()
		cc.URL = &urlString
	}
	if m.Service != nil {
		cc.Service = &admissionregistrationv1.ServiceReference{
			Name:      m.Service.Name,
			Namespace: m.Service.Namespace,
		}
	}
	return cc, nil
}
