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

package submitter

import (
	"context"
	"fmt"

	sdk "github.com/onflow/flow-go-sdk"
)

// Submitter submits transactions for execution.
type Submitter struct {
	// api is typically a Flow SDK client.
	api API
}

// New creates a new Submitter that uses the given API.
func New(api API) *Submitter {
	s := Submitter{
		api: api,
	}
	return &s
}

// Transaction submits the given transaction for execution.
func (s *Submitter) Transaction(tx *sdk.Transaction) error {
	err := s.api.SendTransaction(context.Background(), *tx)
	if err != nil {
		return fmt.Errorf("could not submit transaction: %w", err)
	}

	return nil
}
