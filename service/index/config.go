package index

import (
	"time"
)

// DefaultConfig is the default configuration for the DPS index.
var DefaultConfig = Config{
	ConcurrentTransactions: 16,          // same value as used for batches in badger
	FlushInterval:          time.Second, // maximum idle time before flushing transaction
}

// Config is the configuration of a DPS index.
type Config struct {
	ConcurrentTransactions uint
	FlushInterval          time.Duration
}

// WithConcurrentTransactions specifies the maximum concurrent transactions
// that a DPS index should have.
func WithConcurrentTransactions(concurrent uint) func(*Config) {
	return func(cfg *Config) {
		cfg.ConcurrentTransactions = concurrent
	}
}

// WithFlushInterval sets a custom interval after which we will flush Badger
// transactions, to avoid long waits for DB updates in cases where there is not
// enough data to quickly fill them.
func WithFlushInterval(interval time.Duration) func(*Config) {
	return func(cfg *Config) {
		cfg.FlushInterval = interval
	}
}
