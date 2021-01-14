package v1

import (
	"context"
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	"github.com/darkowlzz/operator-toolkit/object"
)

const (
	// DefaultResyncPeriod is the default period at which resync is executed.
	DefaultResyncPeriod time.Duration = 5 * time.Minute

	zeroDuration time.Duration = 0 * time.Minute
)

// Reconciler defines an external metadata sync reconciler based on the Sync
// reconciler with a sync function for resynchronization of the external
// objects.
type Reconciler struct {
	syncv1.Reconciler
	Ctrlr         Controller
	disableResync bool
	resyncPeriod  time.Duration
}

// DisableResync disables the resync operation.
func (s *Reconciler) DisableResync() {
	s.disableResync = true
}

// SetResyncPeriod sets the resync interval.
func (s *Reconciler) SetResyncPeriod(period time.Duration) {
	s.resyncPeriod = period
}

// Init initializes the reconciler.
func (s *Reconciler) Init(mgr ctrl.Manager, ctrlr Controller, prototype client.Object, prototypeList client.ObjectList, opts ...syncv1.ReconcilerOption) error {
	// Add a resync func if resync is not disabled.
	if !s.disableResync {
		// If the period is zero, use the default period.
		if s.resyncPeriod == zeroDuration {
			s.resyncPeriod = DefaultResyncPeriod
		}

		sf := syncv1.NewSyncFunc(s.resync, s.resyncPeriod)
		sfs := []syncv1.SyncFunc{sf}

		opts = append(opts, syncv1.WithSyncFuncs(sfs))
	}

	// Set controller.
	s.Ctrlr = ctrlr

	// TODO: remove when Init() takes controller as an argument.
	opts = append(opts, syncv1.WithController(ctrlr))

	// Initialize the base sync reconciler.
	return s.Reconciler.Init(mgr, prototype, prototypeList, opts...)
}

// resync lists all the prototype objects in k8s and performs a diff against
// objects in the external system.  Any objects that are determined to require a
// resynchronization then get the metadata re-applied.
func (s *Reconciler) resync() {
	s.Log.WithValues("resync", s.Name)

	controller := s.Ctrlr

	// TODO: Provide option to set timeout for the resync. Since this runs in a
	// goroutine, when the reconcile has a timeout duration, use it with the
	// created context.
	ctx := context.Background()

	// List all the k8s objects.
	instances := s.PrototypeList.DeepCopyObject().(client.ObjectList)
	// TODO: Provide option to set a namespace and other list options.
	if listErr := s.Client.List(ctx, instances); listErr != nil {
		s.Log.Info("failed to list", "error", listErr)
		return
	}

	items, err := apimeta.ExtractList(instances)
	if err != nil {
		s.Log.Info("failed to extract list", "error", err)
		return
	}

	k8sObjList, err := object.ClientObjects(s.Scheme, items)
	if err != nil {
		s.Log.Info("failed to convert", "error", err)
		return
	}

	// Get list of objects requiring resync.
	resyncObjs, listErr := controller.Diff(ctx, k8sObjList)
	if listErr != nil {
		s.Log.Info("failed to list external objects", "error", listErr)
		return
	}

	// Apply metadata for each object.
	for _, obj := range resyncObjs {
		if err := controller.Ensure(ctx, obj); err != nil {
			s.Log.Info("failed to resync metadata to external object", "name", obj.GetName(), "error", err)
		}
	}
	s.Log.Info("resync of metadata completed", "count", len(resyncObjs))
}
