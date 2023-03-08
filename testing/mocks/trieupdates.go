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
