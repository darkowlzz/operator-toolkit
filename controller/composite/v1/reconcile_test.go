package v1

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/darkowlzz/operator-toolkit/controller/composite/v1/mocks"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

func TestCleanupHandler(t *testing.T) {
	// Create a scheme with testdata scheme info.
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	finalizerName := "test-finalizer"
	someFinalizerX := "some-finalizer-x"
	someErr := errors.New("some cleanup error")

	cases := []struct {
		name           string
		obj            *tdv1alpha1.Game
		wantFinalizers []string
		wantDelEnabled bool
		wantResult     ctrl.Result
		wantErr        error
		expectations   func(*mocks.MockController)
	}{
		{
			name: "add new finalizer",
			obj: &tdv1alpha1.Game{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "foo/v1",
					Kind:       "Game",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-game",
					Namespace: "default",
				},
			},
			wantFinalizers: []string{finalizerName},
			expectations:   func(m *mocks.MockController) {},
		},
		{
			name: "keep existing finalizer",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "my-game",
					Namespace:  "default",
					Finalizers: []string{someFinalizerX, finalizerName},
				},
			},
			wantFinalizers: []string{someFinalizerX, finalizerName},
			expectations:   func(m *mocks.MockController) {},
		},
		{
			name: "delete enabled, no finalizer",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "my-game",
					Namespace:         "default",
					Finalizers:        []string{someFinalizerX},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			wantFinalizers: []string{someFinalizerX},
			wantDelEnabled: true,
			expectations:   func(m *mocks.MockController) {},
		},
		{
			name: "call cleanup",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "my-game",
					Namespace:         "default",
					Finalizers:        []string{someFinalizerX, finalizerName},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			wantFinalizers: []string{someFinalizerX, finalizerName},
			wantDelEnabled: true,
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Cleanup(gomock.Any(), gomock.Any())
			},
		},
		{
			name: "call cleanup with error",
			obj: &tdv1alpha1.Game{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "my-game",
					Namespace:         "default",
					Finalizers:        []string{someFinalizerX, finalizerName},
					DeletionTimestamp: &metav1.Time{Time: time.Now()},
				},
			},
			wantFinalizers: []string{someFinalizerX, finalizerName},
			wantDelEnabled: true,
			wantResult:     ctrl.Result{Requeue: true},
			wantErr:        someErr,
			expectations: func(m *mocks.MockController) {
				m.EXPECT().Cleanup(gomock.Any(), gomock.Any()).Return(ctrl.Result{Requeue: true}, someErr)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock controller.
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			m := mocks.NewMockController(mctrl)
			tc.expectations(m)

			cr := CompositeReconciler{}
			err := cr.Init(nil, nil,
				WithScheme(scheme),
				WithFinalizer(finalizerName),
				WithController(m),
			)
			assert.Nil(t, err)

			delEnabled, res, err := cr.cleanupHandler(context.Background(), tc.obj)
			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.wantFinalizers, tc.obj.GetFinalizers(), "finalizers after cleanupHandler call")
			assert.Equal(t, tc.wantDelEnabled, delEnabled, "delete enabled result")
			assert.Equal(t, tc.wantResult, res, "cleanup result")
		})
	}
}
