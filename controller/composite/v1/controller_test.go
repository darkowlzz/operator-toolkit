package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/darkowlzz/operator-toolkit/controller/composite/v1/mocks"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

type fakeLogger struct{}

func (f fakeLogger) Info(msg string, keysAndValues ...interface{})             {}
func (f fakeLogger) Enabled() bool                                             { return false }
func (f fakeLogger) Error(err error, msg string, keysAndValues ...interface{}) {}
func (f fakeLogger) V(level int) logr.InfoLogger                               { return f }
func (f fakeLogger) WithValues(keysAndValues ...interface{}) logr.Logger       { return f }
func (f fakeLogger) WithName(name string) logr.Logger                          { return f }

func TestReconcile(t *testing.T) {
	testFinalizerName := "foofinalizer"

	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	// Create a NamespacedName object.
	gameNamespacedName := types.NamespacedName{
		Name:      "test-game",
		Namespace: "test-ns",
	}

	// Create an instance of the target object used for reconcile.
	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "test-ns",
		},
	}

	// Another instance of target object with initialized status.
	initializedGameObj := gameObj.DeepCopy()
	initializedGameObj.Status = tdv1alpha1.GameStatus{
		Conditions: []metav1.Condition{
			DefaultInitCondition,
		},
	}

	// Clone the initialized gameObj, add a delete timestamp to it and a
	// finalizer. Use for finalizer based cleanup testing.
	gameObjDeleteTimestamp := initializedGameObj.DeepCopy()
	timenow := metav1.Now()
	gameObjDeleteTimestamp.SetDeletionTimestamp(&timenow)
	gameObjDeleteTimestamp.SetFinalizers([]string{testFinalizerName})

	testcases := []struct {
		name         string
		existingObjs []runtime.Object
		reconciler   func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler
		expectations func(*mocks.MockController)
		wantResult   ctrl.Result
		wantErr      bool
	}{
		{
			name: "instance not found",
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {},
			wantResult:   ctrl.Result{},
		},
		{
			name:         "validation failure",
			existingObjs: []runtime.Object{gameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(errors.New("validation failure"))
			},
			wantResult: ctrl.Result{},
			wantErr:    true,
		},
		{
			name:         "init failure",
			existingObjs: []runtime.Object{gameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Initialize(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("init failure"))
			},
			wantResult: ctrl.Result{},
			wantErr:    true,
		},
		{
			name:         "init success",
			existingObjs: []runtime.Object{gameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Initialize(gomock.Any(), gomock.Any(), gomock.Any())
			},
			wantResult: ctrl.Result{},
		},
		{
			name:         "fetch status failure",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Operate(gomock.Any(), gomock.Any())
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.New("failed to get status"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name:         "operate failure",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any())
				m.EXPECT().Operate(gomock.Any(), gomock.Any()).Return(ctrl.Result{Requeue: true}, errors.New("operate error"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name:         "operate successful - requeue",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any())
				m.EXPECT().Operate(gomock.Any(), gomock.Any()).Return(ctrl.Result{Requeue: true}, nil)
			},
			wantResult: ctrl.Result{Requeue: true},
		},
		{
			name:         "updatestatus failure",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any()).Return(errors.New("failed to update status"))
				m.EXPECT().Operate(gomock.Any(), gomock.Any()).Return(ctrl.Result{}, nil)
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name:         "successful reconcile",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any())
				m.EXPECT().Operate(gomock.Any(), gomock.Any()).Return(ctrl.Result{}, nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name:         "finalizer based cleanup strategy",
			existingObjs: []runtime.Object{initializedGameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
					WithFinalizer(testFinalizerName),
					WithCleanupStrategy(FinalizerCleanup),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name:         "finalizer cleanup failure",
			existingObjs: []runtime.Object{gameObjDeleteTimestamp},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
					WithFinalizer(testFinalizerName),
					WithCleanupStrategy(FinalizerCleanup),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().UpdateStatus(gomock.Any(), gomock.Any())
				m.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(ctrl.Result{Requeue: true}, errors.New("failed to cleanup"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name:         "finalizer cleanup success",
			existingObjs: []runtime.Object{gameObjDeleteTimestamp},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *CompositeReconciler {
				cr := &CompositeReconciler{}
				_ = cr.Init(nil, m, &tdv1alpha1.Game{},
					WithScheme(scheme),
					WithClient(cli),
					WithLogger(fakeLogger{}),
					WithInitCondition(DefaultInitCondition),
					WithFinalizer(testFinalizerName),
					WithCleanupStrategy(FinalizerCleanup),
				)
				return cr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Default(gomock.Any(), gomock.Any())
				m.EXPECT().Validate(gomock.Any(), gomock.Any()).Return(nil)
				m.EXPECT().Cleanup(gomock.Any(), gomock.Any())
			},
			wantResult: ctrl.Result{},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create a fake client with some existing objects.
			cli := fake.NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(tc.existingObjs...).
				Build()

			// Create a mock controller to be used in the reconciler.
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			m := mocks.NewMockController(mctrl)
			tc.expectations(m)

			// Create a reconciler with the mock controller, scheme and client.
			c := tc.reconciler(m, scheme, cli)

			request := ctrl.Request{NamespacedName: gameNamespacedName}
			ctx := context.Background()

			res, err := c.Reconcile(ctx, request)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error %t, actual: %v", tc.wantErr, err)
			}
			if res != tc.wantResult {
				t.Errorf("unexpected reconcile result:\n(WNT) %v\n(GOT) %v", tc.wantResult, res)
			}
		})
	}
}
