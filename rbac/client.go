package rbac

import (
	"context"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	APIVersion      = "rbac.authorization.k8s.io/v1"
	RoleKind        = "Role"
	ClusterRoleKind = "ClusterRole"

	DefaultRoleName        = "generated-rbac-role"
	DefaultClusterRoleName = "generated-rbac-cluster-role"

	VerbGet    = "get"
	VerbList   = "list"
	VerbCreate = "create"
	VerbDelete = "delete"
	VerbUpdate = "update"
	VerbPatch  = "patch"
)

// RBACClient embeds a controller-runtime kubernetes Client. It implements the
// Client interface to be able to observe and register the API calls, and pass
// the call to the actual Client. The observed API calls are used to create a
// list of RBAC permissions that were used.
type RBACClient struct {
	client.Client
	Role        *rbacv1.Role
	ClusterRole *rbacv1.ClusterRole
	Log         logr.Logger
	errors      []error
}

// RBACClientOption is used to configure RBACClient.
type RBACClientOption func(*RBACClient)

// WithRoleName sets the generated role name.
func WithRoleName(name string) RBACClientOption {
	return func(c *RBACClient) {
		c.Role.SetName(name)
	}
}

// WithClusterRoleName sets the generated cluster role name.
func WithClusterRoleName(name string) RBACClientOption {
	return func(c *RBACClient) {
		c.ClusterRole.SetName(name)
	}
}

// WithLogger sets the Logger in a Reconciler.
func WithLogger(log logr.Logger) RBACClientOption {
	return func(c *RBACClient) {
		c.Log = log
	}
}

// NewClient returns a new RBACClient from a given Client.
func NewClient(c client.Client, opts ...RBACClientOption) *RBACClient {
	// Create defult RBACClient.
	rc := &RBACClient{
		Client:      c,
		Role:        newRole(DefaultRoleName),
		ClusterRole: newClusterRole(DefaultClusterRoleName),
		Log:         ctrl.Log,
		errors:      []error{},
	}

	// Apply options.
	for _, opt := range opts {
		opt(rc)
	}

	return rc
}

// Scheme returns the scheme this client is using.
func (c *RBACClient) Scheme() *runtime.Scheme {
	return c.Client.Scheme()
}

// RESTMapper returns the scheme this client is using.
func (c *RBACClient) RESTMapper() meta.RESTMapper {
	return c.Client.RESTMapper()
}

// Create implements client.Client.
func (c *RBACClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.recordRule(obj, VerbCreate)
	return c.Client.Create(ctx, obj, opts...)
}

// Update implements client.Client.
func (c *RBACClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.recordRule(obj, VerbUpdate)
	return c.Client.Update(ctx, obj, opts...)
}

// Delete implements client.Client
func (c *RBACClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.recordRule(obj, VerbDelete)
	return c.Client.Delete(ctx, obj, opts...)
}

// DeleteAllOf implements client.Client.
func (c *RBACClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	c.recordRule(obj, VerbDelete)
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

// Patch implements client.Client.
func (c *RBACClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.recordRule(obj, VerbPatch)
	return c.Client.Patch(ctx, obj, patch, opts...)
}

// Get implements client.Client.
func (c *RBACClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	c.recordRule(obj, VerbGet)
	return c.Client.Get(ctx, key, obj)
}

// List implements client.Client.
func (c *RBACClient) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	c.recordRule(obj, VerbList)
	return c.Client.List(ctx, obj, opts...)
}

func (c *RBACClient) Status() client.StatusWriter {
	return &RBACStatusWriter{client: c.Client, rbacClient: c}
}

// RBACStatusWriter implements the StatusWriter interface. Similar to
// RBACClient, It embeds a Client, observes API calls and passes the APIi call
// to the actual client.
type RBACStatusWriter struct {
	client     client.Client
	rbacClient *RBACClient
}

// Update implements client.StatusWriter.
func (sc *RBACStatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	sc.rbacClient.recordRuleWithStatus(obj, VerbUpdate, true)
	return sc.client.Status().Update(ctx, obj, opts...)
}

// Patch implements client.StatusWriter.
func (sc *RBACStatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	sc.rbacClient.recordRuleWithStatus(obj, VerbPatch, true)
	return sc.client.Status().Patch(ctx, obj, patch, opts...)
}
