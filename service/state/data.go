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

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/service/storage"
)

type Data struct {
	core *Core
}

func (d *Data) Header(height uint64) (*flow.Header, error) {
	var header flow.Header
	err := d.core.db.View(storage.RetrieveHeader(height, &header))
	return &header, err
}

func (d *Data) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	// Make sure that the request is for a height below the currently active
	// sentinel height; otherwise, we haven't indexed yet and we might return
	// false information.
	if height > d.core.height {
		return nil, fmt.Errorf("unknown height (current: %d, requested: %d)", d.core.height, height)
	}

	// Iterate over all keys within the events index which are prefixed with the right block height.
	var events []flow.Event
	err := d.core.db.View(storage.RetrieveEvents(height, types, &events))
	return events, err
}
