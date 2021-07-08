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

	"github.com/optakt/flow-dps/models/dps"
)

type Writer struct {
	write dps.Writer
	time  Time
}

func NewWriter(write dps.Writer, time Time) *Writer {
	w := Writer{
		write: write,
		time:  time,
	}
	return &w
}

func (w *Writer) First(height uint64) error {
	defer w.time.Duration(IOWrite, "first")()
	return w.write.First(height)
}

func (w *Writer) Last(height uint64) error {
	defer w.time.Duration(IOWrite, "last")()
	return w.write.Last(height)
}

func (w *Writer) Height(blockID flow.Identifier, height uint64) error {
	defer w.time.Duration(IOWrite, "height")()
	return w.write.Height(blockID, height)
}

func (w *Writer) Commit(height uint64, commit flow.StateCommitment) error {
	defer w.time.Duration(IOWrite, "commit")()
	return w.write.Commit(height, commit)
}

func (w *Writer) Header(height uint64, header *flow.Header) error {
	defer w.time.Duration(IOWrite, "header")()
	return w.write.Header(height, header)
}

func (w *Writer) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	defer w.time.Duration(IOWrite, "payloads")()
	return w.write.Payloads(height, paths, payloads)
}

func (w *Writer) Collections(height uint64, collections []*flow.LightCollection) error {
	defer w.time.Duration(IOWrite, "collections")()
	return w.write.Collections(height, collections)
}

func (w *Writer) Transactions(height uint64, transactions []*flow.TransactionBody) error {
	defer w.time.Duration(IOWrite, "transactions")()
	return w.write.Transactions(height, transactions)
}

func (w *Writer) Events(height uint64, events []flow.Event) error {
	defer w.time.Duration(IOWrite, "events")()
	return w.write.Events(height, events)
}
