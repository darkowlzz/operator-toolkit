// Package pkiutil provides utilities to help with certificates and keys.
// NOTE: The code in this package originates from kubernetes/kubernetes repo's
// kubeadm component. After a re-organization of the certificate and key helper
// libraries (refer https://github.com/kubernetes/kubernetes/issues/71004), a
// lot of the cert related code was moved to kubeadm component. In
// operator-toolkit, this package is used for self signed certificate
// generation for admission controller. To avoid dependency on k/k, the code in
// this package has been modified to comment out the functions that are not
// required for cert generation. The package tests in the upstream are kubeadm
// specific. Since no functionality is changed, the tests have not been copied.
// The code should be updated from the upstream repo from time to time.
package pkiutil
