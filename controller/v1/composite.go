package v1

import (
	"fmt"

	"github.com/go-logr/logr"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CleanupStrategy is the resource cleanup strategy used by the reconciler.
type CleanupStrategy int

const (
	// OwnerReferenceCleanup depends on k8s garbage collector. All the child
	// objects of a parent are added with a reference of the parent object.
	// When the parent object gets deleted, all the child objects are garbage
	// collected.
	OwnerReferenceCleanup CleanupStrategy = iota
	// FinalizerCleanup allows using custom cleanup logic. When this strategy
	// is set, a finalizer is added to the parent object to avoid accidental
	// deletion of the object. When the object is marked for deletion with a
	// deletion timestamp, the custom cleanup code is executed to delete all
	// the child objects. Once all custom cleanup code finished, the finalizer
	// from the parent object is removed and the parent object is allowed to be
	// deleted.
	FinalizerCleanup
)

// CompositeReconciler defines a composite reconciler.
type CompositeReconciler struct {
	InitCondition   conditionsv1.Condition
	FinalizerName   string
	CleanupStrategy CleanupStrategy
	Log             logr.Logger
	Ctrlr           Controller
}

// CompositeReconcilerOptions is used to configure CompositeReconciler.
type CompositeReconcilerOptions func(*CompositeReconciler)

// WithLogger sets the Logger in a CompositeReconciler.
func WithLogger(log logr.Logger) CompositeReconcilerOptions {
	return func(c *CompositeReconciler) {
		c.Log = log
	}
}

// WithController sets the Controller in a CompositeReconciler.
func WithController(ctrlr Controller) CompositeReconcilerOptions {
	return func(c *CompositeReconciler) {
		c.Ctrlr = ctrlr
	}
}

// WithInitCondition sets the initial status Condition to be used by the
// CompositeReconciler on a resource object.
func WithInitCondition(cndn conditionsv1.Condition) CompositeReconcilerOptions {
	return func(c *CompositeReconciler) {
		c.InitCondition = cndn
	}
}

// WithFinalizer sets the name of the finalizer used by the
// CompositeReconciler.
func WithFinalizer(finalizer string) CompositeReconcilerOptions {
	return func(c *CompositeReconciler) {
		c.FinalizerName = finalizer
	}
}

// WithCleanupStrategy sets the CleanupStrategy of the CompositeReconciler.
func WithCleanupStrategy(cleanupStrat CleanupStrategy) CompositeReconcilerOptions {
	return func(c *CompositeReconciler) {
		c.CleanupStrategy = cleanupStrat
	}
}

// NewCompositeReconciler creates a new CompositeReconciler with defaults,
// overridden by the provided options.
func NewCompositeReconciler(opts ...CompositeReconcilerOptions) (*CompositeReconciler, error) {
	cr := &CompositeReconciler{
		Log:             ctrl.Log,
		InitCondition:   DefaultInitCondition,
		CleanupStrategy: OwnerReferenceCleanup,
	}

	for _, opt := range opts {
		opt(cr)
	}

	if cr.Ctrlr == nil {
		return nil, fmt.Errorf("must provide a Controller to the CompositeReconciler")
	}

	return cr, nil
}

// DefaultInitCondition is the default init condition used by the composite
// reconciler to add to the status of a new resource.
var DefaultInitCondition conditionsv1.Condition = conditionsv1.Condition{
	Type:    conditionsv1.ConditionProgressing,
	Status:  corev1.ConditionTrue,
	Reason:  "Initializing",
	Message: "Component initializing",
}
