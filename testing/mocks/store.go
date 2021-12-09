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

package mocks

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
)

type Store struct {
	SaveFunc     func(hash hash.Hash, payload *ledger.Payload)
	RetrieveFunc func(hash hash.Hash) (*ledger.Payload, error)
}

func BaselineStore() *Store {
	s := Store{
		SaveFunc: func(hash hash.Hash, payload *ledger.Payload) {},
		RetrieveFunc: func(hash hash.Hash) (*ledger.Payload, error) {
			return GenericLedgerPayload(0), nil
		},
	}

	return &s
}

func (s *Store) Save(hash hash.Hash, payload *ledger.Payload) {
	s.SaveFunc(hash, payload)
}

func (s *Store) Retrieve(hash hash.Hash) (*ledger.Payload, error) {
	return s.RetrieveFunc(hash)
}
