module github.com/darkowlzz/composite-reconciler

go 1.15

require (
	github.com/darkowlzz/composite-reconciler/testdata v0.0.0
	github.com/go-logr/logr v0.3.0
	github.com/golang/mock v1.4.4
	github.com/goombaio/dag v0.0.0-20181006234417-a8874b1f72ff
	github.com/kr/text v0.2.0 // indirect
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/openshift/custom-resource-status v0.0.0-20200602122900-c002fd1547ca
	github.com/stretchr/testify v1.6.1
	go.opentelemetry.io/otel v0.15.0
	golang.org/x/tools v0.0.0-20200714190737-9048b464a08d // indirect
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v0.19.2
	sigs.k8s.io/controller-runtime v0.7.0
)

replace github.com/darkowlzz/composite-reconciler/testdata => ./testdata
