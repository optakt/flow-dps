package mapper

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/awfm9/flow-dps/model"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// MapperOptions contains optional parameters we can set for the mapper.
type MapperConfig struct {
	CheckpointFile string
}

// Static is a random access ledger that bootstraps the index from a static
// snapshot of the protocol state and the corresponding ledger write-ahead log.
type Mapper struct {
	log     zerolog.Logger
	chain   Chain
	feeder  Feeder
	indexer Indexer
	trie    *trie.MTrie
	deltas  []model.Delta
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feeder Feeder, indexer Indexer, options ...func(*MapperConfig)) (*Mapper, error) {

	cfg := MapperConfig{
		CheckpointFile: "",
	}
	for _, option := range options {
		option(&cfg)
	}

	m := &Mapper{
		log:     log.With().Str("component", "mapper").Logger(),
		chain:   chain,
		feeder:  feeder,
		indexer: indexer,
		trie:    nil,
		deltas:  []model.Delta{},
	}

	if cfg.CheckpointFile != "" {
		file, err := os.Open(cfg.CheckpointFile)
		if err != nil {
			return nil, fmt.Errorf("could not open checkpoint file: %w", err)
		}
		checkpoint, err := wal.ReadCheckpoint(file)
		if err != nil {
			return nil, fmt.Errorf("could not read checkpoint: %w", err)
		}
		tries, err := flattener.RebuildTries(checkpoint)
		if err != nil {
			return nil, fmt.Errorf("could not rebuild tries: %w", err)
		}
		if len(tries) != 1 {
			return nil, fmt.Errorf("should only have one trie in root checkpoint (tries: %d)", len(tries))
		}
		m.trie = tries[0]
	} else {
		trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
		if err != nil {
			return nil, fmt.Errorf("could not initialize empty memory trie: %w", err)
		}
		m.trie = trie
	}

	return m, nil
}

func (m *Mapper) Run() error {
	for {

		// First, we load the next update and apply it to the current trie.
		update, err := m.feeder.Feed()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not feed next update: %w", err)
		}
		delta := make(model.Delta, 0, len(update.Paths))
		for index, path := range update.Paths {
			payload := update.Payloads[index]
			change := model.Change{
				Path:    path,
				Payload: *payload,
			}
			delta = append(delta, change)
		}
		hash := m.trie.RootHash()
		if !bytes.Equal(update.RootHash, hash) {
			return fmt.Errorf("could not match root hash (trie: %x, update: %x)", hash, update.RootHash)
		}
		trie, err := trie.NewTrieWithUpdatedRegisters(m.trie, delta.Paths(), delta.Payloads())
		if err != nil {
			return fmt.Errorf("could not update trie: %w", err)
		}
		m.deltas = append(m.deltas, delta)
		m.trie = trie

		// Second, we check if we have reached the state commitment of the
		// currently active block in the chain. If we have, we keep forwarding
		// the chain until we reach a new state commitment.
		for {
			hash := m.trie.RootHash()
			height, blockID, commit := m.chain.Active()
			if !bytes.Equal(hash, commit) {
				break
			}
			err := m.indexer.Index(height, blockID, commit, m.deltas)
			if err != nil {
				return fmt.Errorf("could not index deltas: %w (height: %d, block: %x, commit: %x)", err, height, blockID, commit)
			}
			m.deltas = []model.Delta{}
			err = m.chain.Forward()
			if errors.Is(err, io.EOF) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("could not forward chain: %w", err)
			}
		}
	}
}
