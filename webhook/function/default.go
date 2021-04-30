package function

import (
	"context"

	"github.com/darkowlzz/operator-toolkit/discovery/cluster"
	"github.com/darkowlzz/operator-toolkit/webhook/admission"
	"k8s.io/client-go/discovery"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AddLabels is a generic defaulter method for adding labels to any given
// object.
func AddLabels(cli client.Client, labels map[string]string) admission.DefaultFunc {
	return func(ctx context.Context, obj client.Object) {
		objLabels := obj.GetLabels()
		if objLabels == nil {
			objLabels = map[string]string{}
		}
		for k, v := range labels {
			objLabels[k] = v
		}
		obj.SetLabels(objLabels)
	}
}

// AddAnnotations is a generic defaulter method for adding annotations to any
// given object.
func AddAnnotations(cli client.Client, annotations map[string]string) admission.DefaultFunc {
	return func(ctx context.Context, obj client.Object) {
		objAnnot := obj.GetAnnotations()
		if objAnnot == nil {
			objAnnot = map[string]string{}
		}
		for k, v := range annotations {
			objAnnot[k] = v
		}
		obj.SetAnnotations(objAnnot)
	}
}

// AddClusterVersionAnnotation is a generic defaulter method for adding cluster
// version info on an object's annotations. This requires a discovery client to
// get the cluster info.
func AddClusterVersionAnnotation(d discovery.DiscoveryInterface) admission.DefaultFunc {
	return func(ctx context.Context, obj client.Object) {
		version, err := cluster.NewFromDiscoveryClient(d).GetClusterVersion()
		if err != nil {
			return
		}
		objAnnot := obj.GetAnnotations()
		if objAnnot == nil {
			objAnnot = map[string]string{}
		}
		objAnnot["cluster-version"] = version
		obj.SetAnnotations(objAnnot)
	}
}
