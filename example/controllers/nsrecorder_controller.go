package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	actionv1 "github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1"
	"github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action"
	"github.com/darkowlzz/operator-toolkit/telemetry"
)

const (
	// configmapCreatedKey is the key of the data written in the namespace
	// configmap.
	configmapCreatedKey = "createdOn"
)

// NamespaceRecorderReconciler reconciles Namespace objects and records them in
// configmaps per namespace in a target namespace.
// This example demonstrates usage of the stateless-action controller. For
// every event of a target kind, it checks if an action is needed, if needed,
// an action manager is built for the target object and run in a separate
// goroutine. The action manager ensure that the action is executed
// successfully.
// In this example, for every namespace event, an action manager is created to
// perform the action of recording the namespace. For every namespace, a
// configmap is created at a given namespace with creation timestamp data. If
// the record already exists, the action is not executed.
type NamespaceRecorderReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	Instrumentation *telemetry.Instrumentation

	actionv1.Reconciler
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceRecorderReconciler) SetupWithManager(mgr ctrl.Manager) error {
	_, _, _, log := r.Instrumentation.Start(context.Background(), "namespaceRecorder.SetupWithManager")

	nsc := &nsRecorder{
		Client: r.Client,
		instrumentation: telemetry.NewInstrumentationWithProviders(
			InstrumentationName, nil, nil, log),
		configmapNamespace: "default",
	}

	// Initialize the reconciler with the namespace recorder controller.
	r.Reconciler.Init(mgr, nsc,
		actionv1.WithName("ns-recorder-controller"),
		actionv1.WithScheme(mgr.GetScheme()),
		actionv1.WithActionTimeout(10*time.Second),
		actionv1.WithActionRetryPeriod(2*time.Second),
		actionv1.WithInstrumentation(nil, nil, log),
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

// nsRecorder implements the stateless-action controller interface for
// namespace recorder controller.
type nsRecorder struct {
	client.Client
	instrumentation *telemetry.Instrumentation

	// configmapNamespace is the namespace where the configmaps will be created.
	configmapNamespace string
}

// GetObject implements the stateless-action controller interface. It returns
// an object given an object key.
func (n *nsRecorder) GetObject(ctx context.Context, key client.ObjectKey) (interface{}, error) {
	// Fetch and return the target namespace object from k8s.
	ns := &corev1.Namespace{}
	err := n.Get(ctx, key, ns)
	return ns, err
}

// RequireAction implements the stateless-action controller interface. It
// checks if an action is required given a target object.
func (n *nsRecorder) RequireAction(ctx context.Context, i interface{}) (bool, error) {
	_, _, _, log := n.instrumentation.Start(ctx, "nsRecorder.RequireAction")

	ns, ok := i.(*corev1.Namespace)
	if !ok {
		log.Info("failed to convert into Namespace", "object", i)
		return false, fmt.Errorf("failed to convert into Namespace")
	}

	// Check if the configmap to store the record exists.
	key := client.ObjectKey{Namespace: n.configmapNamespace, Name: ns.Name}
	cm := &corev1.ConfigMap{}

	if err := n.Client.Get(ctx, key, cm); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		log.Info("failed to get configmap", "error", err)
	}

	return false, nil
}

// BuildActionManager implements the stateless-action controller interface. It
// builds an action manager with the target object and returns it.
func (n *nsRecorder) BuildActionManager(i interface{}) (action.Manager, error) {
	_, _, _, log := n.instrumentation.Start(context.Background(), "nsRecorder.BuildActionManager")

	ns, ok := i.(*corev1.Namespace)
	if !ok {
		log.Info("failed to convert into Namespace", "object", i)
		return nil, fmt.Errorf("failed to convert into Namespace")
	}

	return &nsActionManager{
		Client: n.Client,
		instrumentation: telemetry.NewInstrumentationWithProviders(
			InstrumentationName, nil, nil, log),
		ns:                 ns,
		configmapNamespace: n.configmapNamespace,
	}, nil
}

// nsActionManager implements the action manager interface to be used with a
// stateless-action controller. It manages the actions for recording namespace.
type nsActionManager struct {
	client.Client
	instrumentation *telemetry.Instrumentation

	ns                 *corev1.Namespace
	configmapNamespace string
}

// GetName implements the action manager interface. It returns a unique name
// for the manager based on the given object.
func (am *nsActionManager) GetName(i interface{}) (string, error) {
	_, _, _, log := am.instrumentation.Start(context.Background(), "nsActionManager.GetName")
	ns, ok := i.(*corev1.Namespace)
	if !ok {
		log.Info("failed to convert into Namespace", "object", i)
		return "", fmt.Errorf("failed to convert into Namespace")
	}

	return ns.Name, nil
}

// GetObjects implements the action manager interface. It returns the target
// object on which the action is to be executed.
func (am *nsActionManager) GetObjects(context.Context) ([]interface{}, error) {
	// Return the target namespace itself.
	return []interface{}{am.ns}, nil
}

// Check checks if the action is needed anymore.
func (am *nsActionManager) Check(ctx context.Context, i interface{}) (bool, error) {
	ns, ok := i.(*corev1.Namespace)
	if !ok {
		return true, fmt.Errorf("failed to convert into Namespace %v", i)
	}

	// Check if the target configmap exists, if not, there's no data to check,
	// action is needed.
	key := client.ObjectKey{Namespace: am.configmapNamespace, Name: ns.Name}

	cm := &corev1.ConfigMap{}
	if err := am.Client.Get(ctx, key, cm); err != nil {
		return true, errors.Wrapf(err, "failed to get configmap")
	}

	// If there's no data, action is needed.
	if cm.Data == nil {
		return true, nil
	}

	// If the creation data is not found, action is needed.
	if _, exists := cm.Data[configmapCreatedKey]; !exists {
		return true, nil
	}

	// Action successful, action not needed.
	return false, nil
}

// Run runs the action on the given object.
func (am *nsActionManager) Run(ctx context.Context, i interface{}) error {
	ns, ok := i.(*corev1.Namespace)
	if !ok {
		return fmt.Errorf("failed to convert into Namespace: %v", i)
	}

	// Ensure the target configmap exists.
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ns.Name,
			Namespace: am.configmapNamespace,
		},
	}
	key := client.ObjectKeyFromObject(cm)

	if getErr := am.Get(ctx, key, cm); getErr != nil {
		if apierrors.IsNotFound(getErr) {
			// Create the configmap.
			if createErr := am.Create(ctx, cm); createErr != nil {
				return errors.Wrapf(createErr, "failed to create configmap")
			}
		} else {
			// Return and let the action manager retry.
			return errors.Wrapf(getErr, "failed to get the configmap")
		}
	}

	// Update the configmap with the target namespace record.
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	cm.Data[configmapCreatedKey] = time.Now().String()

	if updateErr := am.Update(ctx, cm); updateErr != nil {
		return errors.Wrapf(updateErr, "failed to update configmap")
	}

	return nil
}

// Defer is executed at the end of run to execute once run ends.
func (am *nsActionManager) Defer(context.Context, interface{}) error {
	// no-op
	return nil
}
