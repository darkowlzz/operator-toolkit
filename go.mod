module github.com/darkowlzz/operator-toolkit

go 1.15

require (
	github.com/go-logr/logr v0.3.0
	github.com/golang/mock v1.4.4
	github.com/goombaio/dag v0.0.0-20181006234417-a8874b1f72ff
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/otel v0.15.0
	go.opentelemetry.io/otel/exporters/otlp v0.15.0
	go.opentelemetry.io/otel/exporters/trace/jaeger v0.15.0
	go.opentelemetry.io/otel/sdk v0.15.0
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0
	sigs.k8s.io/kubebuilder-declarative-pattern v0.0.0-20201209165851-b731a6217520
	sigs.k8s.io/kustomize/api v0.7.0
	sigs.k8s.io/kustomize/kyaml v0.10.3
	sigs.k8s.io/yaml v1.2.0
)
