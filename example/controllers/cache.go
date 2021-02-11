package controllers

import (
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FakeCache is a cache to store event source object data.
type FakeCache struct {
	store map[string]string
	log   logr.Logger
}

// getKey returns the key in a cache, given name and namespace.
func getKey(name, namespace string) string {
	return fmt.Sprintf("%s/%s", name, namespace)
}

// NewFakeCache initializes and returns a new fake cache.
func NewFakeCache(log logr.Logger) *FakeCache {
	return &FakeCache{
		store: map[string]string{},
		log:   log.WithName("fakecache"),
	}
}

// Get returns the value of key in the cache if found with true boolean. If not
// found, an empty string is returned with a false boolean.
func (fc *FakeCache) Get(key string) (string, bool) {
	if val, ok := fc.store[key]; ok {
		return val, true
	}
	return "", false
}

// Set stores the given key and value in the cache.
func (fc *FakeCache) Set(key, val string) {
	fc.store[key] = val
}

// CacheMiss implements external controller Cache interface.
// It checks for cache miss with the given object. On cache miss, it updates
// the cache.
func (fc *FakeCache) CacheMiss(obj client.Object) bool {
	key := getKey(obj.GetName(), obj.GetNamespace())
	_, found := fc.Get(key)

	if found {
		fc.log.Info("cache hit", "name", obj.GetName(), "namespace", obj.GetNamespace())
		return false
	}

	fc.Set(key, "foo")
	fc.log.Info("cache miss", "name", obj.GetName(), "namespace", obj.GetNamespace())
	return true
}
