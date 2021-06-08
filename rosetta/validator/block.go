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

package validator

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/rosetta/identifier"
)

// Block identifier tries to extrapolate the block identifier to a full version
// of itself. For now, we will always need a height.
// NOTE: We always pass a block identifier that in principle at least could be
// valid, so we will have at least a height or a hash.
func (v *Validator) Block(block *identifier.Block) error {

	// We currently only support retrieval by height, until we start indexing
	// the block IDs as part of the DPS index.
	if block.Index == 0 {
		return fmt.Errorf("block access with hash currently not supported")
	}

	// We should always be able to parse this at this point, if it is present,
	// as we already checked the format, so normal error is fine.
	var blockID flow.Identifier
	var err error
	if block.Hash != "" {
		blockID, err = flow.HexStringToIdentifier(block.Hash)
		if err != nil {
			return fmt.Errorf("could not parse block ID: %w", err)
		}
	}

	// The block index can't be below the first indexed height.
	first, err := v.index.First()
	if err != nil {
		return fmt.Errorf("could not get first: %w", err)
	}
	if block.Index < first {
		return failure.InvalidBlock{
			Height:  block.Index,
			BlockID: blockID,
			Message: fmt.Sprintf("block height below first indexed block (first: %d)", first),
		}
	}

	// The block index can't be above the last indexed height.
	last, err := v.index.Last()
	if err != nil {
		return fmt.Errorf("could not get last: %w", err)
	}
	if block.Index > last {
		return failure.UnknownBlock{
			Height:  block.Index,
			BlockID: blockID,
			Message: fmt.Sprintf("block height above last indexed block (last: %d)", last),
		}
	}

	// The given block ID should match the block ID at the given height.
	header, err := v.index.Header(block.Index)
	if err != nil {
		return fmt.Errorf("could not get header: %w", err)
	}
	if block.Hash != "" && block.Hash != header.ID().String() {
		return failure.InvalidBlock{
			Height:  block.Index,
			BlockID: blockID,
			Message: fmt.Sprintf("provided hash does not match real hash for height (real: %s)", header.ID().String()),
		}
	}

	// At this point, they either matched, or the block ID is empty, so we
	// should insert it.
	block.Hash = header.ID().String()

	return nil
}
