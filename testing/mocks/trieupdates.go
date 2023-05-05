package mocks

import (
	"testing"

	"github.com/onflow/flow-go/ledger"
)

type Parser struct {
	UpdatesFunc func() ([]*ledger.TrieUpdate, error)
}

func BaselineParser(t *testing.T) *Parser {
	t.Helper()

	f := Parser{
		UpdatesFunc: func() ([]*ledger.TrieUpdate, error) {
			return GenericTrieUpdates(4), nil
		},
	}

	return &f
}

func (f *Parser) AllUpdates() ([]*ledger.TrieUpdate, error) {
	return f.UpdatesFunc()
}
