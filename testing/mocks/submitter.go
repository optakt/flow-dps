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

	sdk "github.com/onflow/flow-go-sdk"
)

type Submitter struct {
	TransactionFunc func(tx *sdk.Transaction) error
}

func (s *Submitter) Transaction(tx *sdk.Transaction) error {
	return s.TransactionFunc(tx)
}

func BaselineSubmitter(t *testing.T) *Submitter {
	t.Helper()

	s := Submitter{
		TransactionFunc: func(tx *sdk.Transaction) error {
			return nil
		},
	}

	return &s
}
