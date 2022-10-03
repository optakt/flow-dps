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

// MetricsWriter wraps the writer and records metrics for the data it writes.
type MetricsWriter struct {
	write *Writer

	block       prometheus.Counter
	register    prometheus.Counter
	collection  prometheus.Counter
	transaction prometheus.Counter
	event       prometheus.Counter
	seal        prometheus.Counter
}

// NewMetricsWriter creates a counter that counts indexed elements and exposes this information
// as prometheus counters.
func NewMetricsWriter(write *Writer) *MetricsWriter {
	blockOpts := prometheus.CounterOpts{
		Name: "archive_indexed_blocks",
		Help: "number of indexed blocks",
	}
	block := promauto.NewCounter(blockOpts)

	registerOpts := prometheus.CounterOpts{
		Name: "archive_indexed_registers",
		Help: "number of indexed registers",
	}
	register := promauto.NewCounter(registerOpts)

	collectionOpts := prometheus.CounterOpts{
		Name: "archive_indexed_collections",
		Help: "number of indexed collections",
	}
	collection := promauto.NewCounter(collectionOpts)

	transactionsOpts := prometheus.CounterOpts{
		Name: "archive_indexed_transactions",
		Help: "number of indexed transactions",
	}
	transaction := promauto.NewCounter(transactionsOpts)

	eventOpts := prometheus.CounterOpts{
		Name: "archive_indexed_events",
		Help: "number of indexed events",
	}
	event := promauto.NewCounter(eventOpts)

	sealOpts := prometheus.CounterOpts{
		Name: "archive_indexed_seals",
		Help: "number of indexed seals",
	}
	seal := promauto.NewCounter(sealOpts)

	w := MetricsWriter{
		write: write,

		block:       block,
		register:    register,
		collection:  collection,
		transaction: transaction,
		event:       event,
		seal:        seal,
	}

	return &w
}

func (w *MetricsWriter) Header(height uint64, header *flow.Header) error {
	w.block.Inc()
	return w.write.Header(height, header)
}

func (w *MetricsWriter) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	w.register.Add(float64(len(paths)))
	return w.write.Payloads(height, paths, payloads)
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
