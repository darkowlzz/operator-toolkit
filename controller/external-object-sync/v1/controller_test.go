package v1

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/darkowlzz/operator-toolkit/controller/external-object-sync/v1/mocks"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

func TestReconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	// Create a NamespacedName object.
	gameNamespacedName := types.NamespacedName{
		Name:      "test-game",
		Namespace: "test-ns",
	}

	// Create instances of the target object.
	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "test-ns",
		},
	}

	testcases := []struct {
		name         string
		existingObjs []runtime.Object
		reconciler   func(m Controller, scheme *runtime.Scheme, cli client.Client) *SyncReconciler
		expectations func(*mocks.MockController)
		wantResult   ctrl.Result
		wantErr      bool
	}{
		{
			name:         "instance found",
			existingObjs: []runtime.Object{gameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *SyncReconciler {
				sr := &SyncReconciler{}
				_ = sr.Init(nil, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
					WithScheme(scheme),
					WithController(m),
					WithClient(cli),
					WithGarbageCollectorEnabled(false),
				)
				return sr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Ensure(gomock.Any(), gomock.Any())
			},
			wantResult: ctrl.Result{},
		},
		{
			name:         "ensure error",
			existingObjs: []runtime.Object{gameObj},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *SyncReconciler {
				sr := &SyncReconciler{}
				_ = sr.Init(nil, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
					WithScheme(scheme),
					WithController(m),
					WithClient(cli),
					WithGarbageCollectorEnabled(false),
				)
				return sr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Ensure(gomock.Any(), gomock.Any()).Return(errors.New("some error"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
		},
		{
			name:         "instance not found",
			existingObjs: []runtime.Object{},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *SyncReconciler {
				sr := &SyncReconciler{}
				_ = sr.Init(nil, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
					WithScheme(scheme),
					WithController(m),
					WithClient(cli),
					WithGarbageCollectorEnabled(false),
				)
				return sr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Delete(gomock.Any(), gomock.Any())
			},
			wantResult: ctrl.Result{},
		},
		{
			name:         "delete error",
			existingObjs: []runtime.Object{},
			reconciler: func(m Controller, scheme *runtime.Scheme, cli client.Client) *SyncReconciler {
				sr := &SyncReconciler{}
				_ = sr.Init(nil, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
					WithScheme(scheme),
					WithController(m),
					WithClient(cli),
					WithGarbageCollectorEnabled(false),
				)
				return sr
			},
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("some error"))
			},
			wantResult: ctrl.Result{Requeue: true},
			wantErr:    true,
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
			sr := tc.reconciler(m, scheme, cli)

			request := ctrl.Request{NamespacedName: gameNamespacedName}
			ctx := context.Background()

			res, err := sr.Reconcile(ctx, request)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error %t, actual: %v", tc.wantErr, err)
			}
			if res != tc.wantResult {
				t.Errorf("unexpected reconcile result:\n(WNT) %v\n(GOT) %v", tc.wantResult, res)
			}
		})
	}
}
