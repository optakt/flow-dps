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
func (v *Validator) Block(rosBlockID identifier.Block) (identifier.Block, error) {

	// If both the index and the hash are missing, the block identifier is invalid.
	if rosBlockID.Index == nil && rosBlockID.Hash == "" {
		return identifier.Block{}, failure.InvalidBlock{
			Description: failure.NewDescription("block needs either a valid index or a valid hash"),
		}
	}

	// If a block hash is present, it should be a valid block ID for Flow.
	if rosBlockID.Hash != "" {
		_, err := flow.HexStringToIdentifier(rosBlockID.Hash)
		if err != nil {
			return identifier.Block{}, failure.InvalidBlock{
				Description: failure.NewDescription("block hash is not a valid hex-encoded string",
					failure.WithString("block_hash", rosBlockID.Hash),
				),
			}
		}
	}

	// If a block index is present, it should be a valid height for the DPS.
	if rosBlockID.Index != nil {
		first, err := v.index.First()
		if err != nil {
			return identifier.Block{}, fmt.Errorf("could not get first: %w", err)
		}
		if *rosBlockID.Index < first {
			return identifier.Block{}, failure.InvalidBlock{
				Description: failure.NewDescription("block index is below first indexed height",
					failure.WithUint64("block_index", *rosBlockID.Index),
					failure.WithUint64("first_index", first),
				),
			}
		}
		last, err := v.index.Last()
		if err != nil {
			return identifier.Block{}, fmt.Errorf("could not get last: %w", err)
		}
		if *rosBlockID.Index > last {
			return identifier.Block{}, failure.UnknownBlock{
				Index: *rosBlockID.Index,
				Hash:  rosBlockID.Hash,
				Description: failure.NewDescription("block index is above last indexed height",
					failure.WithUint64("last_index", last),
				),
			}
		}
	}

	// If we don't have a height, fill it in now.
	if rosBlockID.Index == nil {
		blockID, _ := flow.HexStringToIdentifier(rosBlockID.Hash)
		height, err := v.index.HeightForBlock(blockID)
		if err != nil {
			return identifier.Block{}, fmt.Errorf("could not get height for block: %w", err)
		}
		rosBlockID.Index = &height
	}

	// The given block ID should match the block ID at the given height.
	header, err := v.index.Header(*rosBlockID.Index)
	if err != nil {
		return identifier.Block{}, fmt.Errorf("could not get header: %w", err)
	}
	if rosBlockID.Hash != "" && rosBlockID.Hash != header.ID().String() {
		return identifier.Block{}, failure.InvalidBlock{
			Description: failure.NewDescription("block hash mismatches with authoritative hash for index",
				failure.WithUint64("block_index", *rosBlockID.Index),
				failure.WithString("block_hash", rosBlockID.Hash),
				failure.WithString("want_hash", header.ID().String()),
			),
		}
	}

	// At this point, they either matched, or the block ID is empty, so we
	// should insert it.
	rosBlockID.Hash = header.ID().String()

	return rosBlockID, nil
}
