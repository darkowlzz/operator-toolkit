// NOTE: This is mostly based on
// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.8.3/pkg/cache/informer_cache.go,
// modified informerCache to use a different InformersMap that's not kubernetes
// ListWatcher API specific.

package cache

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	crCache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/darkowlzz/operator-toolkit/cache/informer"
)

// informerCache is a generic cache, based on the Kubernetes object cache,
// populated from InformersMap.  informerCache wraps an InformersMap.
type informerCache struct {
	*informer.InformersMap
}

// Get implements Reader
func (ic *informerCache) Get(ctx context.Context, key client.ObjectKey, out client.Object) error {
	gvk, err := apiutil.GVKForObject(out, ic.Scheme)
	if err != nil {
		return err
	}

	started, cache, err := ic.InformersMap.Get(ctx, gvk, out)
	if err != nil {
		return err
	}

	if !started {
		return &crCache.ErrCacheNotStarted{}
	}
	return cache.Reader.Get(ctx, key, out)
}

// List implements Reader
func (ic *informerCache) List(ctx context.Context, out client.ObjectList, opts ...client.ListOption) error {
	gvk, cacheTypeObj, err := ic.objectTypeForListObject(out)
	if err != nil {
		return err
	}

	started, cache, err := ic.InformersMap.Get(ctx, *gvk, cacheTypeObj)
	if err != nil {
		return err
	}

	if !started {
		return &crCache.ErrCacheNotStarted{}
	}

	return cache.Reader.List(ctx, out, opts...)
}

// objectTypeForListObject tries to find the runtime.Object and associated GVK
// for a single object corresponding to the passed-in list type. We need them
// because they are used as cache map key.
func (ic *informerCache) objectTypeForListObject(list client.ObjectList) (*schema.GroupVersionKind, runtime.Object, error) {
	gvk, err := apiutil.GVKForObject(list, ic.Scheme)
	if err != nil {
		return nil, nil, err
	}

	if !strings.HasSuffix(gvk.Kind, "List") {
		return nil, nil, fmt.Errorf("non-list type %T (kind %q) passed as output", list, gvk)
	}
	// we need the non-list GVK, so chop off the "List" from the end of the kind
	gvk.Kind = gvk.Kind[:len(gvk.Kind)-4]
	_, isUnstructured := list.(*unstructured.UnstructuredList)
	var cacheTypeObj runtime.Object
	if isUnstructured {
		u := &unstructured.Unstructured{}
		u.SetGroupVersionKind(gvk)
		cacheTypeObj = u
	} else {
		itemsPtr, err := apimeta.GetItemsPtr(list)
		if err != nil {
			return nil, nil, err
		}
		// http://knowyourmeme.com/memes/this-is-fine
		elemType := reflect.Indirect(reflect.ValueOf(itemsPtr)).Type().Elem()
		if elemType.Kind() != reflect.Ptr {
			elemType = reflect.PtrTo(elemType)
		}

		cacheTypeValue := reflect.Zero(elemType)
		var ok bool
		cacheTypeObj, ok = cacheTypeValue.Interface().(runtime.Object)
		if !ok {
			return nil, nil, fmt.Errorf("cannot get cache for %T, its element %T is not a runtime.Object", list, cacheTypeValue.Interface())
		}
	}

	return &gvk, cacheTypeObj, nil
}

// GetInformerForKind returns the informer for the GroupVersionKind
func (ic *informerCache) GetInformerForKind(ctx context.Context, gvk schema.GroupVersionKind) (crCache.Informer, error) {
	// Map the gvk to an object
	obj, err := ic.Scheme.New(gvk)
	if err != nil {
		return nil, err
	}

	_, i, err := ic.InformersMap.Get(ctx, gvk, obj)
	if err != nil {
		return nil, err
	}
	return i.Informer, err
}

// GetInformer returns the informer for the obj
func (ic *informerCache) GetInformer(ctx context.Context, obj client.Object) (crCache.Informer, error) {
	gvk, err := apiutil.GVKForObject(obj, ic.Scheme)
	if err != nil {
		return nil, err
	}

	_, i, err := ic.InformersMap.Get(ctx, gvk, obj)
	if err != nil {
		return nil, err
	}
	return i.Informer, err
}

// NeedLeaderElection implements the LeaderElectionRunnable interface
// to indicate that this can be started without requiring the leader lock.
func (ic *informerCache) NeedLeaderElection() bool {
	return false
}

// IndexField adds an indexer to the underlying cache, using extraction function to get
// value(s) from the given field.  This index can then be used by passing a field selector
// to List. For one-to-one compatibility with "normal" field selectors, only return one value.
// The values may be anything.  They will automatically be prefixed with the namespace of the
// given object, if present.  The objects passed are guaranteed to be objects of the correct type.
func (ic *informerCache) IndexField(ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	informer, err := ic.GetInformer(ctx, obj)
	if err != nil {
		return err
	}
	return indexByField(informer, field, extractValue)
}

func indexByField(indexer crCache.Informer, field string, extractor client.IndexerFunc) error {
	indexFunc := func(objRaw interface{}) ([]string, error) {
		// TODO(directxman12): check if this is the correct type?
		obj, isObj := objRaw.(client.Object)
		if !isObj {
			return nil, fmt.Errorf("object of type %T is not an Object", objRaw)
		}
		meta, err := apimeta.Accessor(obj)
		if err != nil {
			return nil, err
		}
		ns := meta.GetNamespace()

		rawVals := extractor(obj)
		var vals []string
		if ns == "" {
			// if we're not doubling the keys for the namespaced case, just re-use what was returned to us
			vals = rawVals
		} else {
			// if we need to add non-namespaced versions too, double the length
			vals = make([]string, len(rawVals)*2)
		}
		for i, rawVal := range rawVals {
			// save a namespaced variant, so that we can ask
			// "what are all the object matching a given index *in a given namespace*"
			vals[i] = informer.KeyToNamespacedKey(ns, rawVal)
			if ns != "" {
				// if we have a namespace, also inject a special index key for listing
				// regardless of the object namespace
				vals[i+len(rawVals)] = informer.KeyToNamespacedKey("", rawVal)
			}
		}

		return vals, nil
	}

	return indexer.AddIndexers(cache.Indexers{informer.FieldIndexName(field): indexFunc})
}
