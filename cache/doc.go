// Package cache provides a client-go informer based cache that can be
// populated with any API server objects, not just kubernetes API objects. The
// cache can be provided with any ListWatcher func for any API server. When the
// cache is started, it'll run List and Watch, and collect the API objects to
// populate the cache.
package cache
