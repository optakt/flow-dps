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

	"github.com/awfm9/flow-dps/rest"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
)

type Ledger struct {
	core    *Core
	version uint8
}

func (l *Ledger) WithVersion(version uint8) rest.Ledger {
	l.version = version
	return l
}

func (l *Ledger) Get(query *ledger.Query) ([]ledger.Value, error) {

	// convert the query state commitment to a height, so we can use the core
	// API to retrieve the payloads
	commit := query.State()
	height, err := l.core.Height(commit)
	if err != nil {
		return nil, fmt.Errorf("could not get height for commit (%w)", err)
	}

	// this code replicates the original ledger code to convert keys to paths
	// on input and payloads to values on output; the relevant difference is
	// that we use the core API to retrieve payloads one by one by path, which
	// will use the underlying index database that allows accessing payloads
	// at any block height
	paths, err := pathfinder.KeysToPaths(query.Keys(), l.version)
	if err != nil {
		return nil, fmt.Errorf("could not convert keys to paths (%w)", err)
	}
	payloads := make([]*ledger.Payload, 0, len(paths))
	for _, path := range paths {
		payload, err := l.core.Payload(height, path)
		if err != nil {
			return nil, fmt.Errorf("could not get payload (%w)", err)
		}
		payloads = append(payloads, payload)
	}
	values, err := pathfinder.PayloadsToValues(payloads)
	if err != nil {
		return nil, fmt.Errorf("could not convert payloads to values (%w)", err)
	}

	return values, nil
}
