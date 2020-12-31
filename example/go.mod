module github.com/darkowlzz/operator-toolkit/example

go 1.15

require (
	github.com/darkowlzz/operator-toolkit v0.0.0
	github.com/go-logr/logr v0.3.0
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/kubebuilder-declarative-pattern v0.0.0-20201209165851-b731a6217520
	sigs.k8s.io/kustomize/api v0.7.0
)

replace github.com/darkowlzz/operator-toolkit v0.0.0 => ../
