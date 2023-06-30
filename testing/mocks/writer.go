package mocks

import (
	"testing"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
)

type Writer struct {
	FirstFunc                func(height uint64) error
	LastFunc                 func(height uint64) error
	LatestRegisterHeightFunc func(height uint64) error
	HeaderFunc               func(height uint64, header *flow.Header) error
	CommitFunc               func(height uint64, commit flow.StateCommitment) error
	PayloadsFunc             func(height uint64, payloads []*ledger.Payload) error
	RegistersFunc            func(height uint64, registers []*wal.LeafNode) error
	HeightFunc               func(blockID flow.Identifier, height uint64) error
	CollectionsFunc          func(height uint64, collections []*flow.LightCollection) error
	GuaranteesFunc           func(height uint64, guarantees []*flow.CollectionGuarantee) error
	TransactionsFunc         func(height uint64, transactions []*flow.TransactionBody) error
	ResultsFunc              func(results []*flow.TransactionResult) error
	EventsFunc               func(height uint64, events []flow.Event) error
	SealsFunc                func(height uint64, seals []*flow.Seal) error
	CloseFunc                func() error
}

func BaselineWriter(t *testing.T) *Writer {
	t.Helper()

	w := Writer{
		FirstFunc: func(height uint64) error {
			return nil
		},
		LastFunc: func(height uint64) error {
			return nil
		},
		LatestRegisterHeightFunc: func(height uint64) error {
			return nil
		},
		HeaderFunc: func(height uint64, header *flow.Header) error {
			return nil
		},
		CommitFunc: func(height uint64, commit flow.StateCommitment) error {
			return nil
		},
		PayloadsFunc: func(height uint64, payloads []*ledger.Payload) error {
			return nil
		},
		RegistersFunc: func(height uint64, registers []*wal.LeafNode) error {
			return nil
		},
		HeightFunc: func(blockID flow.Identifier, height uint64) error {
			return nil
		},
		CollectionsFunc: func(height uint64, collections []*flow.LightCollection) error {
			return nil
		},
		GuaranteesFunc: func(height uint64, guarantees []*flow.CollectionGuarantee) error {
			return nil
		},
		TransactionsFunc: func(height uint64, transactions []*flow.TransactionBody) error {
			return nil
		},
		ResultsFunc: func(results []*flow.TransactionResult) error {
			return nil
		},
		EventsFunc: func(height uint64, events []flow.Event) error {
			return nil
		},
		SealsFunc: func(height uint64, seals []*flow.Seal) error {
			return nil
		},
		CloseFunc: func() error {
			return nil
		},
	}

	return &w
}

func (w *Writer) First(height uint64) error {
	return w.FirstFunc(height)
}

func (w *Writer) Last(height uint64) error {
	return w.LastFunc(height)
}

func (w *Writer) LatestRegisterHeight(height uint64) error {
	return w.LatestRegisterHeightFunc(height)
}

func (w *Writer) Header(height uint64, header *flow.Header) error {
	return w.HeaderFunc(height, header)
}

func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return w.CommitFunc(height, commit)
}

func (w *Writer) Payloads(height uint64, payloads []*ledger.Payload) error {
	return w.PayloadsFunc(height, payloads)
}

func (w *Writer) Registers(height uint64, registers []*wal.LeafNode) error {
	return w.RegistersFunc(height, registers)
}

func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.HeightFunc(blockID, height)
}

func (w *Writer) Collections(height uint64, collections []*flow.LightCollection) error {
	return w.CollectionsFunc(height, collections)
}

func (w *Writer) Guarantees(height uint64, guarantees []*flow.CollectionGuarantee) error {
	return w.GuaranteesFunc(height, guarantees)
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	return w.TransactionsFunc(height, transactions)
}

func (w *Writer) Results(results []*flow.TransactionResult) error {
	return w.ResultsFunc(results)
}

func (w *Writer) Events(height uint64, events []flow.Event) error {
	return w.EventsFunc(height, events)
}

func (w *Writer) Seals(height uint64, seals []*flow.Seal) error {
	return w.SealsFunc(height, seals)
}

func (w *Writer) Close() error {
	return w.CloseFunc()
}
