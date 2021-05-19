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

const zeroDuration time.Duration = 0 * time.Minute

// Reconciler defines an external metadata sync reconciler based on the Sync
// reconciler with a sync function for resynchronization of the external
// objects.
type Reconciler struct {
	syncv1.Reconciler
	Ctrlr            Controller
	resyncPeriod     time.Duration
	startupSyncDelay time.Duration
}

// SetResyncPeriod sets the resync interval.
func (s *Reconciler) SetResyncPeriod(period time.Duration) {
	s.resyncPeriod = period
}

// StartupSyncDelay sets a delay for the initial resync at startup.
// NOTE: Setting this too low can result in failure due to uninitialized
// controller components.
func (s *Reconciler) SetStartupSyncDelay(period time.Duration) {
	s.startupSyncDelay = period
}

// Init initializes the reconciler.
func (s *Reconciler) Init(mgr ctrl.Manager, ctrlr Controller, prototype client.Object, prototypeList client.ObjectList, opts ...syncv1.ReconcilerOption) error {
	// Add a resync func if resync period is not zero.
	if s.resyncPeriod > zeroDuration {
		sf := syncv1.NewSyncFunc(s.resync, s.resyncPeriod, s.startupSyncDelay)
		sfs := []syncv1.SyncFunc{sf}

		opts = append(opts, syncv1.WithSyncFuncs(sfs))
	}

	// Set controller.
	s.Ctrlr = ctrlr

	// Initialize the base sync reconciler.
	return s.Reconciler.Init(mgr, ctrlr, prototype, prototypeList, opts...)
}

// resync lists all the prototype objects in k8s and performs a diff against
// objects in the external system.  Any objects that are determined to require a
// resynchronization then get the metadata re-applied.
func (s *Reconciler) resync() {
	// TODO: Provide option to set timeout for the resync. Since this runs in a
	// goroutine, when the reconcile has a timeout duration, use it with the
	// created context.
	ctx, span, _, log := s.Inst.Start(context.Background(), "resync")
	defer span.End()
	log.WithValues("resync", s.Name)

	controller := s.Ctrlr

	// List all the k8s objects.
	instances := s.PrototypeList.DeepCopyObject().(client.ObjectList)
	// TODO: Provide option to set a namespace and other list options.
	if listErr := s.Client.List(ctx, instances); listErr != nil {
		log.Error(listErr, "failed to list")
		return
	}

	items, err := apimeta.ExtractList(instances)
	if err != nil {
		log.Error(err, "failed to extract list")
		return
	}

	k8sObjList, err := object.ClientObjects(s.Scheme, items)
	if err != nil {
		log.Error(err, "failed to convert")
		return
	}

	// Get list of objects requiring resync.
	resyncObjs, listErr := controller.Diff(ctx, k8sObjList)
	if listErr != nil {
		log.Error(listErr, "failed to list external objects")
		return
	}

	// Apply metadata for each object.
	for _, obj := range resyncObjs {
		if err := controller.Ensure(ctx, obj); err != nil {
			log.Error(err, "failed to resync metadata to external object", "name", obj.GetName())
		}
	}
	log.Info("resync of metadata completed", "count", len(resyncObjs))
}
