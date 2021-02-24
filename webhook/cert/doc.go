// Package cert provides a secret based certificate manager for webhook
// servers. If no existing certificate is found for the webhook server, the
// certificate manager generates a self signed certificate and writes it to a
// k8s secret object. The generated cert is written on disk and used by the
// webhook server. The manager periodically checks if the certificate is valid
// and refreshes it if needed. On restarts, the cert is fetched from the secret
// object and reused if the cert is still valid.
package cert
