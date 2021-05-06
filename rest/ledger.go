package rest

import (
	"github.com/onflow/flow-go/ledger"
)

type RawConfig struct {
	height uint64
}

func WithHeight(height uint64) func(*RawConfig) {
	return func(cfg *RawConfig) {
		cfg.height = height
	}
}

type Core interface {
	Raw(options ...func(*RawConfig)) (Raw, error)
	Ledger(options ...func(*RawConfig)) (Ledger, error)
}

type Raw interface {
	Get(height uint64, key []byte) ([]byte, error)
}

type Ledger interface {
	Get(*ledger.Query) ([]ledger.Value, error)
}
