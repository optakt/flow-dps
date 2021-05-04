package model

import (
	"github.com/onflow/flow-go/ledger"
)

type Change struct {
	Path    ledger.Path
	Payload ledger.Payload
}

type Delta []Change

func (d Delta) Paths() []ledger.Path {
	paths := make([]ledger.Path, 0, len(d))
	for _, change := range d {
		paths = append(paths, change.Path)
	}
	return paths
}

func (d Delta) Payloads() []ledger.Payload {
	payloads := make([]ledger.Payload, 0, len(d))
	for _, change := range d {
		payloads = append(payloads, change.Payload)
	}
	return payloads
}
