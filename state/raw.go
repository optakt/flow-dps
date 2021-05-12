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

	"github.com/awfm9/flow-dps/model/dps"
)

type Raw struct {
	core   *Core
	height uint64
}

func (r *Raw) WithHeight(height uint64) dps.Raw {
	r.height = height
	return r
}

// Get returns the raw ledger data from the given ledger key, without the
// original key information.
func (r *Raw) Get(key []byte) ([]byte, error) {

	payload, err := r.core.payload(r.height, key)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve payload: %w", err)
	}

	return payload.Value, nil
}
