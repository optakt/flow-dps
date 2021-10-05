// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package index

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

const (
	labelBlockID       = "blockid"
	labelPath          = "registerpath"
	labelCollectionID  = "collectionid"
	labelTransactionID = "transactionid"
	labelEventID       = "eventid"
	labelSealID        = "sealid"
)

// MetricsWriter wraps the writer and records metrics for the data it writes.
type MetricsWriter struct {
	write *Writer

	blocks       *prometheus.CounterVec
	registers    *prometheus.CounterVec
	collections  *prometheus.CounterVec
	transactions *prometheus.CounterVec
	events       *prometheus.CounterVec
	seals        *prometheus.CounterVec
}

// NewMetricsWriter creates a new index writer that writes new indexing data to the
// given Badger database.
func NewMetricsWriter(write *Writer) *MetricsWriter {
	blockOpts := prometheus.CounterOpts{
		Name: "indexed_blocks",
		Help: "the number of indexed blocks",
	}
	blocks := promauto.NewCounterVec(blockOpts, []string{labelBlockID})

	registerOpts := prometheus.CounterOpts{
		Name: "indexed_registers",
		Help: "the number of indexed registers",
	}
	registers := promauto.NewCounterVec(registerOpts, []string{labelPath})

	collectionOpts := prometheus.CounterOpts{
		Name: "indexed_collections",
		Help: "the number of indexed collections",
	}
	collections := promauto.NewCounterVec(collectionOpts, []string{labelCollectionID})

	transactionsOpts := prometheus.CounterOpts{
		Name: "indexed_transactions",
		Help: "the number of indexed transactions",
	}
	transactions := promauto.NewCounterVec(transactionsOpts, []string{labelTransactionID})

	eventOpts := prometheus.CounterOpts{
		Name: "indexed_events",
		Help: "the number of indexed events",
	}
	events := promauto.NewCounterVec(eventOpts, []string{labelEventID})

	sealOpts := prometheus.CounterOpts{
		Name: "indexed_seal",
		Help: "the number of indexed seal",
	}
	seals := promauto.NewCounterVec(sealOpts, []string{labelSealID})

	w := MetricsWriter{
		write: write,

		blocks:       blocks,
		registers:    registers,
		collections:  collections,
		transactions: transactions,
		events:       events,
		seals:        seals,
	}

	return &w
}

// First indexes the height of the first finalized block.
func (w *MetricsWriter) First(height uint64) error {
	return w.write.First(height)
}

// Last indexes the height of the last finalized block.
func (w *MetricsWriter) Last(height uint64) error {
	return w.write.Last(height)
}

// Height indexes the height for the given block ID.
func (w *MetricsWriter) Height(blockID flow.Identifier, height uint64) error {
	return w.write.Height(blockID, height)
}

// Commit indexes the given commitment of the execution state as it was after
// the execution of the finalized block at the given height.
func (w *MetricsWriter) Commit(height uint64, commit flow.StateCommitment) error {
	return w.write.Commit(height, commit)
}

// Header indexes the given header of a finalized block at the given height.
func (w *MetricsWriter) Header(height uint64, header *flow.Header) error {
	w.blocks.With(prometheus.Labels{labelBlockID: header.ID().String()}).Inc()
	return w.write.Header(height, header)
}

// Payloads indexes the given payloads, which should represent a trie update
// of the execution state contained within the finalized block at the given
// height.
func (w *MetricsWriter) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	for _, path := range paths {
		w.registers.With(prometheus.Labels{labelPath: path.String()}).Inc()
	}
	return w.write.Payloads(height, paths, payloads)
}

// Collections indexes the collections at the given height.
func (w *MetricsWriter) Collections(height uint64, collections []*flow.LightCollection) error {
	for _, collection := range collections {
		w.collections.With(prometheus.Labels{labelCollectionID: collection.ID().String()}).Inc()
	}
	return w.write.Collections(height, collections)
}

// Guarantees indexes the guarantees at the given height.
func (w *MetricsWriter) Guarantees(height uint64, guarantees []*flow.CollectionGuarantee) error {
	return w.write.Guarantees(height, guarantees)
}

// Transactions indexes the transactions at the given height.
func (w *MetricsWriter) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	for _, transaction := range transactions {
		w.transactions.With(prometheus.Labels{labelTransactionID: transaction.ID().String()}).Inc()
	}
	return w.write.Transactions(height, transactions)
}

// Results indexes the transaction results at the given height.
func (w *MetricsWriter) Results(results []*flow.TransactionResult) error {
	return w.write.Results(results)
}

// Events indexes the events, which should represent all events of the finalized
// block at the given height.
func (w *MetricsWriter) Events(height uint64, events []flow.Event) error {
	for _, event := range events {
		w.events.With(prometheus.Labels{labelEventID: event.ID().String()}).Inc()
	}
	return w.write.Events(height, events)
}

// Seals indexes the seals, which should represent all seals in the finalized
// block at the given height.
func (w *MetricsWriter) Seals(height uint64, seals []*flow.Seal) error {
	for _, seal := range seals {
		w.seals.With(prometheus.Labels{labelSealID: seal.ID().String()}).Inc()
	}
	return w.write.Seals(height, seals)
}

// Close closes the writer and commits the pending transaction, if there is one.
func (w *MetricsWriter) Close() error {
	return w.write.Close()
}
