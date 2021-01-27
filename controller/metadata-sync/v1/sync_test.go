package v1

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/darkowlzz/operator-toolkit/controller/metadata-sync/v1/mocks"
	syncv1 "github.com/darkowlzz/operator-toolkit/controller/sync/v1"
	"github.com/darkowlzz/operator-toolkit/object"
	tdv1alpha1 "github.com/darkowlzz/operator-toolkit/testdata/api/v1alpha1"
)

func TestResync(t *testing.T) {
	scheme := runtime.NewScheme()
	assert.Nil(t, tdv1alpha1.AddToScheme(scheme))

	// Create instances of the target object.
	gameObj := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game",
			Namespace: "test-ns",
			Labels: map[string]string{
				"genre": "pvp",
			},
		},
	}

	gameObj2 := &tdv1alpha1.Game{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-game2",
			Namespace: "test-ns2",
			Labels: map[string]string{
				"genre": "simulator",
			},
		},
	}

	existingObjs := []runtime.Object{gameObj, gameObj2}

	// Only obj2 needs update.
	diffList, err := object.ClientObjects(scheme, []runtime.Object{gameObj2})
	require.NoError(t, err)

	// Create a mock of the controller.
	mctrl := gomock.NewController(t)
	defer mctrl.Finish()
	m := mocks.NewMockController(mctrl)

	// Set mock expectations.
	m.EXPECT().Diff(gomock.Any(), gomock.Any()).Return(diffList, nil)
	m.EXPECT().Ensure(gomock.Any(), gomock.Any()).Times(1).Do(func(ctx context.Context, obj client.Object) {
		assert.Equal(t, gameObj2.GetName(), obj.GetName())
		assert.Equal(t, gameObj2.GetLabels(), obj.GetLabels())
	})

	// Create a fake k8s client with existing objects.
	cli := fake.NewClientBuilder().
		WithScheme(scheme).
		WithRuntimeObjects(existingObjs...).
		Build()

	// Initialize the reconciler.
	sr := Reconciler{}
	sr.SetResyncPeriod(5 * time.Minute)
	// Set the delay to avoid running the sync automatically during the test.
	sr.SetStartupSyncDelay(1 * time.Minute)
	err = sr.Init(nil, m, &tdv1alpha1.Game{}, &tdv1alpha1.GameList{},
		syncv1.WithScheme(scheme),
		syncv1.WithClient(cli),
	)
	assert.Nil(t, err)

	// Run resync functions.
	for _, f := range sr.SyncFuncs {
		f.Call()
	}
}
