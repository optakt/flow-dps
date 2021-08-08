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

	"github.com/onflow/flow-go-sdk"
)

// Submitter uses the Flow Access API to submit specified transaction for execution.
type Submitter struct {
	api API
}

// New creates a new Submitter that uses the specified API, typically a Flow SDK client.
func New(api API) *Submitter {
	s := &Submitter{
		api: api,
	}
	return s
}

// Transaction will submit the specified transaction to the Flow Access API.
func (s *Submitter) Transaction(tx *flow.Transaction) error {
	err := s.api.SendTransaction(context.Background(), *tx)
	if err != nil {
		return fmt.Errorf("could not submit transaction: %w", err)
	}
	return nil
}
