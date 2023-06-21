package mapper

import (
	"time"
)

// DefaultConfig is the default configuration for the Mapper.
var DefaultConfig = Config{
	BootstrapState: false,
	SkipRegisters:  false,
	WaitInterval:   10 * time.Millisecond,
}

// Config contains optional parameters for the Mapper.
type Config struct {
	BootstrapState bool
	SkipRegisters  bool
	WaitInterval   time.Duration
}

// Option is an option that can be given to the mapper to configure optional
// parameters on initialization.
type Option func(*Config)

// WithBootstrapState makes the mapper bootstrap the state from a root
// checkpoint. If not set, it will resume indexing from a previous trie.
func WithBootstrapState(bootstrap bool) Option {
	return func(cfg *Config) {
		cfg.BootstrapState = bootstrap
	}
}

// WithSkipRegisters makes the mapper skip indexing of all ledger registers,
// which speeds up the run significantly and can be used for debugging purposes.
func WithSkipRegisters(skip bool) Option {
	return func(cfg *Config) {
		cfg.SkipRegisters = skip
	}
}

// WithWaitInterval sets the wait interval that we will wait before retrying
// to retrieve a trie update when it wasn't available.
func WithWaitInterval(interval time.Duration) Option {
	return func(cfg *Config) {
		cfg.WaitInterval = interval
	}
}
