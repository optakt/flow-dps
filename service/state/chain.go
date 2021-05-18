// Copyright 2021 Alvalor S.A.
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

package state

import (
	"fmt"

	"github.com/dgraph-io/badger/v2"

	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/service/storage"
)

type Chain struct {
	core *Core
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	var header flow.Header
	err := c.core.db.View(func(tx *badger.Txn) error {
		return storage.RetrieveHeader(height, &header)(tx)
	})

	return &header, err
}

func (c *Chain) Events(height uint64, types ...string) ([]flow.Event, error) {
	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information.
	if height > c.core.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", c.core.height, height)
	}

	// Iterate over all keys within the events index which are prefixed with the right block height.
	var events []flow.Event
	err := c.core.db.View(func(tx *badger.Txn) error {
		return storage.RetrieveEvents(height, types, &events)(tx)
	})
	if err != nil {
		return nil, fmt.Errorf("could not retrieve events: %w", err)
	}

	return events, nil
}
