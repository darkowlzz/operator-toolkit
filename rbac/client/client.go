package client

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

// Client embeds a controller-runtime generic Client. It implements the
// Client interface to be able to observe and register the API calls, and pass
// the call to the actual Client. The observed API calls are used to create a
// list of RBAC permissions that were used.
type Client struct {
	client.Client
	Role        *rbacv1.Role
	ClusterRole *rbacv1.ClusterRole
	Log         logr.Logger
	errors      []error
}

// ClientOption is used to configure Client.
type ClientOption func(*Client)

// WithRoleName sets the generated role name.
func WithRoleName(name string) ClientOption {
	return func(c *Client) {
		c.Role.SetName(name)
	}
}

// WithClusterRoleName sets the generated cluster role name.
func WithClusterRoleName(name string) ClientOption {
	return func(c *Client) {
		c.ClusterRole.SetName(name)
	}
}

// WithLogger sets the Logger in Client.
func WithLogger(log logr.Logger) ClientOption {
	return func(c *Client) {
		c.Log = log
	}
}

// NewClient returns a new RBAC Client from a given Client.
func NewClient(c client.Client, opts ...ClientOption) *Client {
	// Create defult Client.
	rc := &Client{
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
func (c *Client) Scheme() *runtime.Scheme {
	return c.Client.Scheme()
}

// RESTMapper returns the scheme this client is using.
func (c *Client) RESTMapper() meta.RESTMapper {
	return c.Client.RESTMapper()
}

// Create implements client.Client.
func (c *Client) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	c.recordRule(obj, VerbCreate)
	return c.Client.Create(ctx, obj, opts...)
}

// Update implements client.Client.
func (c *Client) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	c.recordRule(obj, VerbUpdate)
	return c.Client.Update(ctx, obj, opts...)
}

// Delete implements client.Client
func (c *Client) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	c.recordRule(obj, VerbDelete)
	return c.Client.Delete(ctx, obj, opts...)
}

// DeleteAllOf implements client.Client.
func (c *Client) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
	c.recordRule(obj, VerbDelete)
	return c.Client.DeleteAllOf(ctx, obj, opts...)
}

// Patch implements client.Client.
func (c *Client) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	c.recordRule(obj, VerbPatch)
	return c.Client.Patch(ctx, obj, patch, opts...)
}

// Get implements client.Client.
func (c *Client) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	c.recordRule(obj, VerbGet)
	return c.Client.Get(ctx, key, obj)
}

// List implements client.Client.
func (c *Client) List(ctx context.Context, obj client.ObjectList, opts ...client.ListOption) error {
	c.recordRule(obj, VerbList)
	return c.Client.List(ctx, obj, opts...)
}

func (c *Client) Status() client.StatusWriter {
	return &StatusWriter{client: c.Client, rbacClient: c}
}

// StatusWriter implements the StatusWriter interface. Similar to
// Client, it embeds a Client, observes API calls and passes the APIi call
// to the actual client.
type StatusWriter struct {
	client     client.Client
	rbacClient *Client
}

// Update implements client.StatusWriter.
func (sc *StatusWriter) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	sc.rbacClient.recordRuleWithStatus(obj, VerbUpdate, true)
	return sc.client.Status().Update(ctx, obj, opts...)
}

// Patch implements client.StatusWriter.
func (sc *StatusWriter) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
	sc.rbacClient.recordRuleWithStatus(obj, VerbPatch, true)
	return sc.client.Status().Patch(ctx, obj, patch, opts...)
}
