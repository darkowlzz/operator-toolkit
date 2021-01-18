package rbac

import (
	"fmt"

	"github.com/pkg/errors"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// recordRule records RBAC rule for a given Object with a given verb.
func (c *RBACClient) recordRule(obj runtime.Object, verb string) {
	c.recordRuleWithStatus(obj, verb, false)
}

// recordRuleWithStatus records RBAC rule of a given Object with a given verb
// and status. The status is set to true when the rule is recorded for a status
// update.
// Since the record is called before calling the actual client, the recorded
// rules are collected with no optimization, simple list append. An optimized
// set of rules can be obtained from Result(), which reorders the rules before
// writing the RBAC manifests.
// NOTE: To avoid the recorder from interfering with the Client's operation,
// errors shouldn't cause a failure, but only log and store the error for later
// use.
func (c *RBACClient) recordRuleWithStatus(obj runtime.Object, verb string, status bool) {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		c.errors = append(c.errors, err)
		c.Log.Error(err, "failed to get GVK")
	}
	// We need only the plural form of resource.
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)

	namespaced, err := isNamespaced(c, obj)
	if err != nil {
		c.errors = append(c.errors, err)
		c.Log.Error(err, "failed to find isNamespaced")
	}

	resource := gvr.Resource

	// If it's a status rule, append the resource name with status.
	if status {
		resource = fmt.Sprintf("%s/%s", gvr.Resource, "status")
	}

	rule := rbacv1.PolicyRule{
		APIGroups: []string{gvr.Group},
		Resources: []string{resource},
		Verbs:     []string{verb},
	}

	if namespaced {
		c.Role.Rules = append(c.Role.Rules, rule)
	} else {
		c.ClusterRole.Rules = append(c.ClusterRole.Rules, rule)
	}
}

// isNamespaced returns true if the object is namespace scoped.
// For unstructured objects the gvk is found from the object itself.
// NOTE: Taken from https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.0/pkg/client/namespaced_client.go#L60
func isNamespaced(c client.Client, obj runtime.Object) (bool, error) {
	var gvk schema.GroupVersionKind
	var err error

	_, isUnstructured := obj.(*unstructured.Unstructured)
	_, isUnstructuredList := obj.(*unstructured.UnstructuredList)

	isUnstructured = isUnstructured || isUnstructuredList
	if isUnstructured {
		gvk = obj.GetObjectKind().GroupVersionKind()
	} else {
		gvk, err = apiutil.GVKForObject(obj, c.Scheme())
		if err != nil {
			return false, err
		}
	}

	gk := schema.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}
	restmapping, err := c.RESTMapper().RESTMapping(gk)
	if err != nil {
		return false, fmt.Errorf("failed to get restmapping: %w", err)
	}
	scope := restmapping.Scope.Name()

	if scope == "" {
		return false, errors.New("Scope cannot be identified. Empty scope returned")
	}

	if scope != meta.RESTScopeNameRoot {
		return true, nil
	}
	return false, nil
}
