package main

import (
	"github.com/onflow/flow-go/ledger"
)

type Change struct {
	Path    ledger.Path
	Payload ledger.Payload
}

type ChangeSet []Change

func (cs ChangeSet) Size() uint {
	return uint(len(cs))
}

func (cs ChangeSet) Merge(update *ledger.TrieUpdate) ChangeSet {
	for index, path := range update.Paths {
		payload := update.Payloads[index]
		change := Change{Path: path, Payload: *payload}
		cs = append(cs, change)
	}
	return cs
}
