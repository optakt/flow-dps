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

package rosetta

import (
	"github.com/awfm9/flow-dps/models/identifier"
)

// Block contains an array of transactions that occurred at a particular block
// identifier. A hard requirement for blocks returned by Rosetta implementations
// is that they must be inalterable: once a client has requested and received a
// block identified by a specific block identifier, all future calls for that
// same block identifier must return the same block contents.
//
// Examples given of metadata in the Rosetta API documentation are
// `transaction_root` and `difficulty`.
type Block struct {
	ID           identifier.Block `json:"block_identifier"`
	ParentID     identifier.Block `json:"parent_block_identifier"`
	Timestamp    int64            `json:"timestamp"`
	Transactions []Transaction    `json:"transactions"`
}
