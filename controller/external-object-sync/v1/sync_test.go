package v1

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/darkowlzz/operator-toolkit/controller/external-object-sync/v1/mocks"
	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

func TestCollectGarbage(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	// Create instances of the target object.
	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "test-ns",
		},
	}

	gameObj2 := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game2",
			Namespace: "test-ns2",
		},
	}

	existingObjs := []runtime.Object{gameObj, gameObj2}

	// Mock object list results from the external system.
	extObjNsNList := []types.NamespacedName{
		{Name: gameObj.GetName(), Namespace: gameObj.GetNamespace()},
		{Name: gameObj2.GetName(), Namespace: gameObj2.GetNamespace()},
		{Name: "oldobj1", Namespace: "somens1"},
		{Name: "oldobj2", Namespace: "somens2"},
	}

	// Create a mock of the controller.
	mctrl := gomock.NewController(t)
	defer mctrl.Finish()
	m := mocks.NewMockController(mctrl)

	// Set mock expectations.
	m.EXPECT().List(gomock.Any()).Return(extObjNsNList, nil)
	m.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(2)

	// Create a fake k8s client with existing objects.
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(existingObjs...).
		Build()

	// Initialize the reconciler.
	sr := Reconciler{}
	sr.SetGarbageCollectionPeriod(5 * time.Minute)
	// Set the delay to avoid running the GC automatically during the test.
	sr.SetStartupGarbageCollectionDelay(1 * time.Minute)
	err := sr.Init(nil, m, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
		syncv1.WithScheme(scheme),
		syncv1.WithClient(cli),
	)
	assert.Nil(t, err)

	// Run garbage collection sync functions.
	for _, f := range sr.SyncFuncs {
		f.Call()
	}
}
