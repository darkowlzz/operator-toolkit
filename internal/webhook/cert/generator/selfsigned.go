/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package generator

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
	"time"

	cert "github.com/darkowlzz/operator-toolkit/internal/pkiutil"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

const oneYear = 365 * 24 * time.Hour

// ServiceToCommonName generates the CommonName for the certificate when using a k8s service.
func ServiceToCommonName(serviceNamespace, serviceName string) string {
	return fmt.Sprintf("%s.%s.svc", serviceName, serviceNamespace)
}

// SelfSignedCertGenerator implements the certGenerator interface.
// It provisions self-signed certificates.
// NOTE: The self signed root CA cert is created with a validity of 10 years.
// This is set by the upstream client-go's cert utils package.
type SelfSignedCertGenerator struct {
	caKey  []byte
	caCert []byte
	// Validity is the length of the generated certificate's validity and signed by the
	// root CA cert.
	Validity time.Duration
}

var _ CertGenerator = &SelfSignedCertGenerator{}

// SetCA sets the PEM-encoded CA private key and CA cert for signing the generated serving cert.
func (cp *SelfSignedCertGenerator) SetCA(caKey, caCert []byte) {
	cp.caKey = caKey
	cp.caCert = caCert
}

// Generate creates and returns a CA certificate, certificate and
// key for the server. serverKey and serverCert are used by the server
// to establish trust for clients, CA certificate is used by the
// client to verify the server authentication chain.
// The cert will be valid for 365 days.
func (cp *SelfSignedCertGenerator) Generate(commonName string) (*Artifacts, error) {
	var signingKey *rsa.PrivateKey
	var signingCert *x509.Certificate
	var valid bool
	var err error

	// If the validity is not set, set the default to a year.
	if cp.Validity == 0 {
		cp.Validity = oneYear
	}

	// Calculate validity
	certBestBefore := time.Now().Add(cp.Validity)

	// Public key algorithm.
	// TODO: Maybe allow passing the algorithm as an argument or a field in the
	// generator.
	keyType := x509.RSA

	valid, signingKey, signingCert = cp.validCACert(certBestBefore)
	if !valid {
		signer, err := cert.NewPrivateKey(keyType)
		if err != nil {
			return nil, fmt.Errorf("failed to create the CA private key: %v", err)
		}
		var ok bool
		signingKey, ok = signer.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("failed to convert CA signer to RSA private key")
		}
		signingCert, err = certutil.NewSelfSignedCACert(certutil.Config{CommonName: "webhook-cert-ca"}, signingKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create the CA cert: %v", err)
		}
	}

	signer, err := cert.NewPrivateKey(keyType)
	if err != nil {
		return nil, fmt.Errorf("failed to create the private key: %v", err)
	}
	key, ok := signer.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("failed to conver signer to RSA private key")
	}
	signedCert, err := cert.NewSignedCertWithValidity(
		&cert.CertConfig{
			Config: certutil.Config{
				CommonName: commonName,
				// Read more about the AltNames requirement since go 1.15 from
				// https://github.com/golang/go/issues/39568#issuecomment-671424481.
				AltNames: certutil.AltNames{
					DNSNames: []string{commonName},
				},
				Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			},
		},
		key, signingCert, signingKey, false, certBestBefore,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create the cert: %v", err)
	}
	return &Artifacts{
		Key:    cert.EncodePrivateKeyPEM(key),
		Cert:   cert.EncodeCertPEM(signedCert),
		CAKey:  cert.EncodePrivateKeyPEM(signingKey),
		CACert: cert.EncodeCertPEM(signingCert),
	}, nil
}

func (cp *SelfSignedCertGenerator) validCACert(time time.Time) (bool, *rsa.PrivateKey, *x509.Certificate) {
	if !ValidCACert(cp.caKey, cp.caCert, cp.caCert, "", time) {
		return false, nil, nil
	}

	var ok bool
	key, err := keyutil.ParsePrivateKeyPEM(cp.caKey)
	if err != nil {
		return false, nil, nil
	}
	privateKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return false, nil, nil
	}

	certs, err := certutil.ParseCertsPEM(cp.caCert)
	if err != nil {
		return false, nil, nil
	}
	if len(certs) != 1 {
		return false, nil, nil
	}
	return true, privateKey, certs[0]
}
