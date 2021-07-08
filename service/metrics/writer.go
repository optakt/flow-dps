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

package metrics

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
	"github.com/optakt/flow-dps/service/index"
)

type Writer struct {
	write index.Writer
}

func NewWriter(write index.Writer) *Writer {
	w := Writer{
		write: write,
	}
	return &w
}

func (w *Writer) First(height uint64) error {
	return w.write.First(height)
}

func (w *Writer) Last(height uint64) error {
	return w.write.Last(height)
}

func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	return w.write.Height(blockID, height)
}

func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	return w.write.Commit(height, commit)
}

func (w *Writer) Header(height uint64, header *flow.Header) error {
	return w.write.Header(height, header)
}

func (w *Writer) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	return w.write.Payloads(height, paths, payloads)
}

func (w *Writer) Collections(height uint64, collections []*flow.LightCollection) error {
	return w.write.Collections(height, collections)
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	return w.write.Transactions(height, transactions)
}

func (w *Writer) Events(height uint64, events []flow.Event) error {
	return w.write.Events(height, events)
}
