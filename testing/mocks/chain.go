package mocks

import (
	"testing"

	"github.com/onflow/flow-go/model/flow"
)

type Chain struct {
	RootFunc         func() (uint64, error)
	HeaderFunc       func(height uint64) (*flow.Header, error)
	CommitFunc       func(height uint64) (flow.StateCommitment, error)
	CollectionsFunc  func(height uint64) ([]*flow.LightCollection, error)
	GuaranteesFunc   func(height uint64) ([]*flow.CollectionGuarantee, error)
	TransactionsFunc func(height uint64) ([]*flow.TransactionBody, error)
	ResultsFunc      func(height uint64) ([]*flow.TransactionResult, error)
	EventsFunc       func(height uint64) ([]flow.Event, error)
	SealsFunc        func(height uint64) ([]*flow.Seal, error)
}

func BaselineChain(t *testing.T) *Chain {
	t.Helper()

	c := Chain{
		RootFunc: func() (uint64, error) {
			return GenericHeight, nil
		},
		HeaderFunc: func(height uint64) (*flow.Header, error) {
			return GenericHeader, nil
		},
		CommitFunc: func(height uint64) (flow.StateCommitment, error) {
			return GenericCommit(0), nil
		},
		CollectionsFunc: func(height uint64) ([]*flow.LightCollection, error) {
			return GenericCollections(2), nil
		},
		GuaranteesFunc: func(height uint64) ([]*flow.CollectionGuarantee, error) {
			return GenericGuarantees(2), nil
		},
		TransactionsFunc: func(height uint64) ([]*flow.TransactionBody, error) {
			return GenericTransactions(4), nil
		},
		ResultsFunc: func(height uint64) ([]*flow.TransactionResult, error) {
			return GenericResults(4), nil
		},
		EventsFunc: func(height uint64) ([]flow.Event, error) {
			return GenericEvents(4), nil
		},
		SealsFunc: func(height uint64) ([]*flow.Seal, error) {
			return GenericSeals(4), nil
		},
	}

	return &c
}

func (c *Chain) Root() (uint64, error) {
	return c.RootFunc()
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	return c.HeaderFunc(height)
}

func (c *Chain) Commit(height uint64) (flow.StateCommitment, error) {
	return c.CommitFunc(height)
}

func (c *Chain) Collections(height uint64) ([]*flow.LightCollection, error) {
	return c.CollectionsFunc(height)
}

func (c *Chain) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	return c.GuaranteesFunc(height)
}

func (c *Chain) Transactions(height uint64) ([]*flow.TransactionBody, error) {
	return c.TransactionsFunc(height)
}

func (c *Chain) Results(height uint64) ([]*flow.TransactionResult, error) {
	return c.ResultsFunc(height)
}

func (c *Chain) Events(height uint64) ([]flow.Event, error) {
	return c.EventsFunc(height)
}

func (c *Chain) Seals(height uint64) ([]*flow.Seal, error) {
	return c.SealsFunc(height)
}
