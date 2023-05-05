package metrics

import (
	"github.com/onflow/flow-archive/service/index"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
)

// MetricsWriter wraps the writer and records metrics for the data it writes.
type MetricsWriter struct {
	write          *index.Writer
	block          prometheus.Counter
	consensusBlock prometheus.Gauge
	register       prometheus.Counter
	collection     prometheus.Counter
	transaction    prometheus.Counter
	event          prometheus.Counter
	seal           prometheus.Counter
}

// NewMetricsWriter creates a counter that counts indexed elements and exposes this information
// as prometheus counters.
func NewMetricsWriter(write *index.Writer) *MetricsWriter {
	blockOpts := prometheus.CounterOpts{
		Name:      "indexed_blocks",
		Namespace: namespaceArchive,
		Help:      "number of indexed blocks",
	}
	block := promauto.NewCounter(blockOpts)

	consensusBlockOpts := prometheus.GaugeOpts{
		Name:      "consensus_block_height",
		Namespace: namespaceArchive,
		Help:      "latest block synced by consensus follower",
	}
	consensusBlock := promauto.NewGauge(consensusBlockOpts)

	registerOpts := prometheus.CounterOpts{
		Name:      "indexed_registers",
		Namespace: namespaceArchive,
		Help:      "number of indexed registers",
	}
	register := promauto.NewCounter(registerOpts)

	collectionOpts := prometheus.CounterOpts{
		Name:      "indexed_collections",
		Namespace: namespaceArchive,
		Help:      "number of indexed collections",
	}
	collection := promauto.NewCounter(collectionOpts)

	transactionsOpts := prometheus.CounterOpts{
		Name:      "indexed_transactions",
		Namespace: namespaceArchive,
		Help:      "number of indexed transactions",
	}
	transaction := promauto.NewCounter(transactionsOpts)

	eventOpts := prometheus.CounterOpts{
		Name:      "indexed_events",
		Namespace: namespaceArchive,
		Help:      "number of indexed events",
	}
	event := promauto.NewCounter(eventOpts)

	sealOpts := prometheus.CounterOpts{
		Name:      "indexed_seals",
		Namespace: namespaceArchive,
		Help:      "number of indexed seals",
	}
	seal := promauto.NewCounter(sealOpts)

	w := MetricsWriter{
		write:          write,
		block:          block,
		consensusBlock: consensusBlock,
		register:       register,
		collection:     collection,
		transaction:    transaction,
		event:          event,
		seal:           seal,
	}

	return &w
}

func (w *MetricsWriter) Header(height uint64, header *flow.Header) error {
	w.block.Inc()
	w.consensusBlock.Set(float64(height))
	return w.write.Header(height, header)
}

func (w *MetricsWriter) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	w.register.Add(float64(len(paths)))
	return w.write.Payloads(height, paths, payloads)
}

func (w *MetricsWriter) Registers(height uint64, registers []*wal.LeafNode) error {
	w.register.Add(float64(len(registers)))
	return w.write.Registers(height, registers)
}

func (w *MetricsWriter) Collections(height uint64, collections []*flow.LightCollection) error {
	w.collection.Add(float64(len(collections)))
	return w.write.Collections(height, collections)
}

func (w *MetricsWriter) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	w.transaction.Add(float64(len(transactions)))
	return w.write.Transactions(height, transactions)
}

func (w *MetricsWriter) Events(height uint64, events []flow.Event) error {
	w.event.Add(float64(len(events)))
	return w.write.Events(height, events)
}

func (w *MetricsWriter) Seals(height uint64, seals []*flow.Seal) error {
	w.seal.Add(float64(len(seals)))
	return w.write.Seals(height, seals)
}

func (w *MetricsWriter) First(height uint64) error {
	return w.write.First(height)
}

func (w *MetricsWriter) Last(height uint64) error {
	return w.write.Last(height)
}

func (w *MetricsWriter) Height(blockID flow.Identifier, height uint64) error {
	return w.write.Height(blockID, height)
}

func (w *MetricsWriter) Commit(height uint64, commit flow.StateCommitment) error {
	return w.write.Commit(height, commit)
}

func (w *MetricsWriter) Guarantees(height uint64, guarantees []*flow.CollectionGuarantee) error {
	return w.write.Guarantees(height, guarantees)
}

func (w *MetricsWriter) Results(results []*flow.TransactionResult) error {
	return w.write.Results(results)
}
