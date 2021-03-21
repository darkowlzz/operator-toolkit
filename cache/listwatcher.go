package cache

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/darkowlzz/operator-toolkit/cache/informer"
)

// ListWatcherClient defines an interface for a list watcher client.
type ListWatcherClient interface {
	List(ctx context.Context, namespace string, obj runtime.Object) (runtime.Object, error)
	Watch(ctx context.Context, namespace string, kind string) (watch.Interface, error)
}

// ListWatcher embeds a ListWatcherClient and uses the client to provider a
// cache.ListWatch.
type ListWatcher struct {
	ListWatcherClient
}

// CreateListWatcherFunc returns a CreateListWatcherFunc that uses the
// ListWatcherClient.
func (r ListWatcher) CreateListWatcherFunc() informer.CreateListWatcherFunc {
	return func(gvk schema.GroupVersionKind, namespace string, scheme *runtime.Scheme) (*cache.ListWatch, error) {
		listGVK := gvk.GroupVersion().WithKind(gvk.Kind + "List")
		listObj, err := scheme.New(listGVK)
		if err != nil {
			return nil, err
		}

		ctx := context.TODO()

		return &cache.ListWatch{
			ListFunc: func(opts metav1.ListOptions) (runtime.Object, error) {
				res := listObj.DeepCopyObject()
				return r.List(ctx, namespace, res)
			},
			WatchFunc: func(opts metav1.ListOptions) (watch.Interface, error) {
				return r.Watch(ctx, namespace, gvk.Kind)
			},
		}, nil
	}
}
