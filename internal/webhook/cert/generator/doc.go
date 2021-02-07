/*
Package generator provides an interface and implementation to provision
certificates.
Create an instance of certGenerator.
	cg := SelfSignedCertGenerator{}
Generate the certificates.
	certs, err := cg.Generate("foo.bar.com")
	if err != nil {
		// handle error
	}
*/

// NOTE: This package originates from controller-runtime v0.1. The later
// versions of controller-runtime removed support for self signed certificate
// generation. The dependencies of this package no longer have the functions it
// depended on, mostly client-go, and have been moved to other projects as part
// of re-organization of certificate and key helper libraries. Refer
// https://github.com/kubernetes/kubernetes/issues/71004 for more details.
package generator
