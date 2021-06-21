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

package main

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"
)

type Index struct{}

func (*Index) First(height uint64) error {
	return nil
}

func (*Index) Last(height uint64) error {
	return nil
}

func (*Index) Header(height uint64, header *flow.Header) error {
	return nil
}

func (*Index) Commit(height uint64, commit flow.StateCommitment) error {
	return nil
}

func (*Index) Payloads(height uint64, paths []ledger.Path, payloads []*ledger.Payload) error {
	return nil
}

func (*Index) Events(height uint64, events []flow.Event) error {
	return nil
}

func (*Index) Height(blockID flow.Identifier, height uint64) error {
	return nil
}

func (*Index) Transactions(blockID flow.Identifier, transactions []flow.Transaction) error {
	return nil
}

func (*Index) Collections(blockID flow.Identifier, collections []flow.LightCollection) error {
	return nil
}
