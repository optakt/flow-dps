package invoker

// Cache represents a key/value store to use as a cache.
type Cache interface {
	Get(key interface{}) (interface{}, bool)
	Set(key, value interface{}, cost int64) bool
}
