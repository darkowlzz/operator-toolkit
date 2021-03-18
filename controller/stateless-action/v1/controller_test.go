package v1

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action"
	actionmocks "github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/action/mocks"
	"github.com/darkowlzz/operator-toolkit/controller/stateless-action/v1/mocks"
)

const testActionManagerName = "testAM"

func TestReconcile(t *testing.T) {
	testcases := []struct {
		name         string
		reconciler   func(m Controller, am action.Manager) *Reconciler
		expectations func(*mocks.MockController, *actionmocks.MockManager)
		wantResult   ctrl.Result
		wantErr      bool
	}{
		{
			name: "object not found",
			reconciler: func(m Controller, am action.Manager) *Reconciler {
				r := &Reconciler{}
				r.Init(nil, m)
				return r
			},
			expectations: func(m *mocks.MockController, am *actionmocks.MockManager) {
				m.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return(nil, nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "object found, action not required",
			reconciler: func(m Controller, am action.Manager) *Reconciler {
				r := &Reconciler{}
				r.Init(nil, m)
				return r
			},
			expectations: func(m *mocks.MockController, am *actionmocks.MockManager) {
				m.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return("a", nil)
				m.EXPECT().RequireAction(gomock.Any(), gomock.Any()).Return(false, nil)
			},
			wantResult: ctrl.Result{},
		},
		{
			name: "object found, action required",
			reconciler: func(m Controller, am action.Manager) *Reconciler {
				r := &Reconciler{}
				r.Init(nil, m)
				return r
			},
			expectations: func(m *mocks.MockController, am *actionmocks.MockManager) {
				m.EXPECT().GetObject(gomock.Any(), gomock.Any()).Return("a", nil)
				m.EXPECT().RequireAction(gomock.Any(), gomock.Any()).Return(true, nil)
				m.EXPECT().BuildActionManager(gomock.Any()).Return(am, nil)
				am.EXPECT().GetObjects(gomock.Any()).Return(nil, nil)
			},
			wantResult: ctrl.Result{},
		},
		// Action run is tested separately since they runs in goroutine if
		// called from the reconciler.
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			mc := mocks.NewMockController(mctrl)
			mam := actionmocks.NewMockManager(mctrl)
			tc.expectations(mc, mam)

			r := tc.reconciler(mc, mam)

			request := ctrl.Request{NamespacedName: types.NamespacedName{
				Name:      "test-obj",
				Namespace: "test-ns",
			}}
			ctx := context.Background()

			res, err := r.Reconcile(ctx, request)
			if (err != nil) != tc.wantErr {
				t.Errorf("expected error %t, actual: %v", tc.wantErr, err)
			}
			if res != tc.wantResult {
				t.Errorf("unexpected reconcile result:\n(WNT) %v\n(GOT) %v", tc.wantResult, res)
			}
		})
	}
}

func TestRunAction(t *testing.T) {
	objA := "a"

	testcases := []struct {
		name string

		expectations func(m *actionmocks.MockManager)
	}{
		{
			name: "no retry",
			expectations: func(m *actionmocks.MockManager) {
				m.EXPECT().GetName(gomock.Any()).Return(testActionManagerName, nil)
				m.EXPECT().Run(gomock.Any(), objA)
				m.EXPECT().Defer(gomock.Any(), objA)
				m.EXPECT().Check(gomock.Any(), objA).Return(false)
			},
		},
		{
			name: "retry",
			expectations: func(m *actionmocks.MockManager) {
				m.EXPECT().GetName(gomock.Any()).Return(testActionManagerName, nil)
				m.EXPECT().Run(gomock.Any(), objA).Times(2)
				m.EXPECT().Defer(gomock.Any(), objA)
				// Check returns true the first time, causing a run retry.
				check1 := m.EXPECT().Check(gomock.Any(), objA).Return(true)
				m.EXPECT().Check(gomock.Any(), objA).Return(false).After(check1)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			mctrl := gomock.NewController(t)
			defer mctrl.Finish()
			m := actionmocks.NewMockManager(mctrl)
			tc.expectations(m)

			r := &Reconciler{
				log:           ctrl.Log,
				actionTimeout: 5 * time.Second,
			}
			r.RunAction(m, objA)
		})
	}
}
