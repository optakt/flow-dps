package invoker

// Config is the configuration for an invoker.
type Config struct {
	CacheSize uint64
}

// WithCacheSize specifies the size of the cache the invoker uses.
func WithCacheSize(size uint64) func(*Config) {
	return func(cfg *Config) {
		cfg.CacheSize = size
	}
}
