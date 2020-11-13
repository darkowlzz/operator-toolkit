package v1

import (
	"testing"

	"github.com/golang/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// ZeroResult is the zero value of a reconcile result.
var ZeroResult ctrl.Result = ctrl.Result{}

func TestInstanceNotFound(t *testing.T) {
	mctrl := gomock.NewController(t)

	defer mctrl.Finish()

	m := mocks.NewMockController(mctrl)
	m.EXPECT().InitReconcile(gomock.Any(), gomock.Any())
	m.EXPECT().FetchInstance().Return(NotFoundError())

	c := &CompositeReconciler{
		C: m,
	}
	req := ctrl.Request{}
	res, err := c.Reconcile(req)
	if err != nil {
		t.Errorf("expected reconcile error to be nil, got: %v", err)
	}
	if res != ZeroResult {
		t.Errorf("unexpected reconcile result:\n(WNT) %v\n(GOT) %v", ZeroResult, res)
	}
}
