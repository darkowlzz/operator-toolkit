package v1

import (
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/darkowlzz/operator-toolkit/constant"
	"github.com/darkowlzz/operator-toolkit/telemetry"
)

// CleanupStrategy is the resource cleanup strategy used by the reconciler.
type CleanupStrategy int

const (
	// Name of the instrumentation.
	instrumentationName = constant.LibraryName + "/controller/composite"

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
	name            string
	initCondition   metav1.Condition
	finalizerName   string
	cleanupStrategy CleanupStrategy
	ctrlr           Controller
	prototype       client.Object
	client          client.Client
	scheme          *runtime.Scheme
	inst            *telemetry.Instrumentation
}

// CompositeReconcilerOption is used to configure CompositeReconciler.
type CompositeReconcilerOption func(*CompositeReconciler)

// WithName sets the name of the CompositeReconciler.
func WithName(name string) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.name = name
	}
}

// WithClient sets the k8s client in the reconciler.
func WithClient(cli client.Client) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.client = cli
	}
}

// WithPrototype sets a prototype of the object that's reconciled.
func WithPrototype(obj client.Object) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.prototype = obj
	}
}

// WithInitCondition sets the initial status Condition to be used by the
// CompositeReconciler on a resource object.
func WithInitCondition(cndn metav1.Condition) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.initCondition = cndn
	}
}

// WithFinalizer sets the name of the finalizer used by the
// CompositeReconciler.
func WithFinalizer(finalizer string) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.finalizerName = finalizer
	}
}

// WithCleanupStrategy sets the CleanupStrategy of the CompositeReconciler.
func WithCleanupStrategy(cleanupStrat CleanupStrategy) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.cleanupStrategy = cleanupStrat
	}
}

// WithScheme sets the runtime Scheme of the CompositeReconciler.
func WithScheme(scheme *runtime.Scheme) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		c.scheme = scheme
	}
}

// WithInstrumentation configures the instrumentation  of the
// CompositeReconciler.
func WithInstrumentation(tp trace.TracerProvider, mp metric.MeterProvider, log logr.Logger) CompositeReconcilerOption {
	return func(c *CompositeReconciler) {
		if log != nil && c.name != "" {
			log = log.WithValues("reconciler", c.name)
		}
		c.inst = telemetry.NewInstrumentationWithProviders(instrumentationName, tp, mp, log)
	}
}

// Init initializes the CompositeReconciler for a given Object with the given
// options.
func (c *CompositeReconciler) Init(mgr ctrl.Manager, ctrlr Controller, prototype client.Object, opts ...CompositeReconcilerOption) error {
	c.ctrlr = ctrlr

	// Use manager if provided. This is helpful in tests to provide explicit
	// client and scheme without a manager.
	if mgr != nil {
		c.client = mgr.GetClient()
		c.scheme = mgr.GetScheme()
	}

	// Use prototype if provided.
	if prototype != nil {
		c.prototype = prototype
	}

	// Add defaults.
	c.initCondition = DefaultInitCondition
	c.cleanupStrategy = OwnerReferenceCleanup

	// Run the options to override the defaults.
	for _, opt := range opts {
		opt(c)
	}

	// If finalizer name is not provided, use the controller name.
	if c.finalizerName == "" {
		c.finalizerName = c.name
	}

	// If instrumentation is nil, create a new instrumentation with default
	// providers.
	if c.inst == nil {
		WithInstrumentation(nil, nil, ctrl.Log)(c)
	}

	return nil
}

// DefaultInitCondition is the default init condition used by the composite
// reconciler to add to the status of a new resource.
var DefaultInitCondition metav1.Condition = metav1.Condition{
	Type:    "Progressing",
	Status:  metav1.ConditionTrue,
	Reason:  "Initializing",
	Message: "Component initializing",
}
