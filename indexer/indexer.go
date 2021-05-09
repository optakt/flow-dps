package indexer

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"

	"github.com/dgraph-io/badger/v2"
	"github.com/fxamacker/cbor"
	"github.com/klauspost/compress/zstd"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model"
)

type Indexer struct {
	index      *badger.DB
	compressor *zstd.Encoder
}

func New(dir string) (*Indexer, error) {

	opts := badger.DefaultOptions(dir).WithLogger(nil)
	index, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("could not open index database: %w", err)
	}

	dict, err := hex.DecodeString(model.Dictionary)
	if err != nil {
		return nil, fmt.Errorf("could not decode dictionary: %w", err)
	}

	compressor, err := zstd.NewWriter(nil,
		zstd.WithEncoderDict(dict),
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compressor: %w", err)
	}

	i := &Indexer{
		index:      index,
		compressor: compressor,
	}

	return i, nil
}

// DB returns the database handle used by the indexer.
func (i *Indexer) DB() *badger.DB {
	return i.index
}

// Index is used to index a new set of state deltas for the given block.
func (i *Indexer) Index(height uint64, blockID flow.Identifier, commit flow.StateCommitment, deltas []model.Delta) error {

	// let's use a single transaction to make indexing of a new block atomic
	tx := i.index.NewTransaction(true)

	// first, map the block ID to the height for easy lookup later
	key := make([]byte, 1+len(blockID))
	key[0] = model.BlockToHeight
	copy(key[1:], blockID[:])
	val := make([]byte, 8)
	binary.BigEndian.PutUint64(val, height)
	err := tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not set block-to-height index (%w)", err)
	}

	// second, map the commit to the height for easy lookup later
	key = make([]byte, 1+len(commit))
	key[0] = model.CommitToHeight
	copy(key[1:], commit)
	err = tx.Set(key, val)
	if err != nil {
		return fmt.Errorf("could not set commit-to-height index (%w)", err)
	}

	// finally, we index the payload for every path that has changed in this block
	for _, delta := range deltas {
		for _, change := range delta {
			key = make([]byte, 1+pathfinder.PathByteSize+8)
			key[0] = model.PathDeltas
			copy(key[1:1+pathfinder.PathByteSize], change.Path)
			binary.BigEndian.PutUint64(key[1+pathfinder.PathByteSize:], height)
			val, err := cbor.Marshal(change.Payload, cbor.CanonicalEncOptions())
			if err != nil {
				return fmt.Errorf("could not encode payload (%w)", err)
			}
			val = i.compressor.EncodeAll(val, nil)
			err = tx.Set(key, val)
			if err != nil {
				return fmt.Errorf("could not set path payload (%w)", err)
			}
		}
	}

	// let's not forget to finalize the transaction
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("could not commit transaction (%w)", err)
	}

	return nil
}
