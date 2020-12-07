package v1

import (
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/darkowlzz/composite-reconciler/mocks"
)

// NotFoundError returns a k8s object not found error.
func NotFoundError() error {
	sch := schema.GroupResource{
		Group:    "testgroup",
		Resource: "testresource",
	}
	return apierrors.NewNotFound(sch, "testobj")
}

type fakeLogger struct{}

func (f fakeLogger) Info(msg string, keysAndValues ...interface{})             {}
func (f fakeLogger) Enabled() bool                                             { return false }
func (f fakeLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (f fakeLogger) V(level int) logr.InfoLogger                               { return f }
func (f fakeLogger) WithValues(keysAndValues ...interface{}) logr.Logger       { return f }
func (f fakeLogger) WithName(name string) logr.Logger                          { return f }

func TestReconcile(t *testing.T) {
	testcases := []struct {
		name         string
		reconciler   func(m Controller) *CompositeReconciler
		expectations func(*mocks.MockController)
		wantResult   ctrl.Result
		wantErr      bool
	}{
		{
			name: "instance not found",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(NotFoundError())
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "validation failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(errors.New("validation failure"))
			},
			wantResult: ctrl.Result{},
			wantErr:    true,
		},
		{
			name: "init failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{}).Return(errors.New("init failure"))
			},
			wantResult: ctrl.Result{},
			wantErr:    true,
		},

		{
			name: "fetch status failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().Operate()
				m.EXPECT().UpdateStatus().Return(errors.New("failed to get status"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name: "operate failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().UpdateStatus()
				m.EXPECT().PatchStatus()
				m.EXPECT().Operate().Return(ctrl.Result{Requeue: true}, errors.New("operate error"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name: "operate successful - requeue",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().UpdateStatus()
				m.EXPECT().PatchStatus()
				m.EXPECT().Operate().Return(ctrl.Result{Requeue: true}, nil)
			},
			wantResult: ctrl.Result{Requeue: true},
		},
		{
			name: "updatestatus failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().UpdateStatus()
				m.EXPECT().PatchStatus().Return(errors.New("failed to update status"))
				m.EXPECT().Operate().Return(ctrl.Result{}, nil)
			},
			wantResult: ctrl.Result{},
			wantErr:    true,
		},
		{
			name: "successful reconcile",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr: m,
					Log:   fakeLogger{},
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().UpdateStatus()
				m.EXPECT().PatchStatus()
				m.EXPECT().Operate().Return(ctrl.Result{}, nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "add finalizer failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr:           m,
					Log:             fakeLogger{},
					FinalizerName:   "foofinalizer",
					CleanupStrategy: FinalizerCleanup,
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().UpdateStatus()
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().AddFinalizer(gomock.Any()).Return(errors.New("failed to add finalizer"))
				m.EXPECT().PatchStatus()
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name: "add finalizer success",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr:           m,
					Log:             fakeLogger{},
					FinalizerName:   "foofinalizer",
					CleanupStrategy: FinalizerCleanup,
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().UpdateStatus()
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().AddFinalizer(gomock.Any())
				m.EXPECT().PatchStatus()
				m.EXPECT().Operate().Return(ctrl.Result{}, nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "finalizer cleanup failure",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr:           m,
					Log:             fakeLogger{},
					FinalizerName:   "foofinalizer",
					CleanupStrategy: FinalizerCleanup,
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().PatchStatus()
				m.EXPECT().UpdateStatus()
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				// Create a time and add as delete timestamp.
				timenow := metav1.Now()
				metadata := metav1.ObjectMeta{
					DeletionTimestamp: &timenow,
					Finalizers:        []string{"foofinalizer"},
				}
				m.EXPECT().GetObjectMetadata().Return(metadata)
				m.EXPECT().Cleanup().Return(ctrl.Result{Requeue: true}, errors.New("failed to cleanup"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name: "finalizer cleanup success",
			reconciler: func(m Controller) *CompositeReconciler {
				return &CompositeReconciler{
					Ctrlr:           m,
					Log:             fakeLogger{},
					FinalizerName:   "foofinalizer",
					CleanupStrategy: FinalizerCleanup,
				}
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().InitReconcile(gomock.Any(), ctrl.Request{})
				m.EXPECT().FetchInstance().Return(nil)
				m.EXPECT().Default()
				m.EXPECT().Validate().Return(nil)
				m.EXPECT().SaveClone()
				m.EXPECT().IsUninitialized().Return(true)
				m.EXPECT().GetObjectMetadata()
				m.EXPECT().Initialize(conditionsv1.Condition{})
				// Create a time and add as delete timestamp.
				timenow := metav1.Now()
				metadata := metav1.ObjectMeta{
					DeletionTimestamp: &timenow,
					Finalizers:        []string{"foofinalizer"},
				}
				m.EXPECT().GetObjectMetadata().Return(metadata)
				m.EXPECT().Cleanup()
				m.EXPECT().UpdateStatus()
				m.EXPECT().PatchStatus()
			},
			wantResult: ctrl.Result{},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			m := mocks.NewMockController(mctrl)
			tc.expectations(m)
			c := tc.reconciler(m)
			request := ctrl.Request{}

			res, err := c.Reconcile(request)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error %t, actual: %v", tc.wantErr, err)
			}
			if res != tc.wantResult {
				t.Errorf("unexpected reconcile result:\n(WNT) %v\n(GOT) %v", tc.wantResult, res)
			}
		})
	}
}
