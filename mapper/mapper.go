package mapper

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/awfm9/flow-dps/model"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
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

	// The first loop is responsible for reading deltas from the feeder and
	// updating the state trie with the delta in order to get the next trie
	// state. If the feeder times out (which can happen, for example, for a
	// feeder receiving trie updates over the network), it will go into a tight
	// loop until the state delta to be applied to the trie with the current
	// root hash has been successfully retrieved.
	for {
		commit := flow.StateCommitment(m.trie.RootHash())
		delta, err := m.feeder.Feed(commit)
		if errors.Is(err, model.ErrTimeout) {
			m.log.Warn().Msg("delta feeding has timed out")
			continue
		}
		if errors.Is(err, model.ErrFinished) {
			m.log.Debug().Msg("no more trie updates available")
			return nil
		}
		if err != nil {
			return fmt.Errorf("could not feed next update: %w", err)
		}
		trie, err := trie.NewTrieWithUpdatedRegisters(m.trie, delta.Paths(), delta.Payloads())
		if err != nil {
			return fmt.Errorf("could not update trie: %w", err)
		}
		m.deltas = append(m.deltas, delta)
		m.trie = trie

		// The second loop is responsible for mapping the currently active block
		// to the set of deltas that were collected. If the state commitment for
		// the block we are looking for isn't the same as the trie root hash, we
		// will immediately go to the next iteration of the outer loop to keep
		// adding deltas to the trie. If it does match, we will index the block
		// with the set of deltas we collected. This might happen more than once
		// if no change to the state trie happens between multiple blocks, at
		// which point we map the second and any subsequent blocks without
		// change to an empty set of deltas.
		commit = flow.StateCommitment(m.trie.RootHash())
		for {
			height, blockID, sentinel := m.chain.Active()
			if !bytes.Equal(sentinel, commit) {
				break
			}
			err := m.indexer.Index(height, blockID, commit, m.deltas)
			if err != nil {
				return fmt.Errorf("could not index deltas: %w (height: %d, block: %x, commit: %x)", err, height, blockID, commit)
			}
			m.deltas = []model.Delta{}

			// The third loop is responsible for forwarding the chain to the
			// next block after each block indexing. This basically forwards the
			// pointer to the active block, for which we will look for the
			// state commitment in the trie next. The loop is required in case
			// there is a timeout (which can happen if we load the chain data
			// over the network), so that we only resume the rest of the logic
			// once we have successfully forwarded to the next block.
			for {
				err = m.chain.Forward()
				if errors.Is(err, model.ErrTimeout) {
					m.log.Warn().Msg("chain forwarding has timed out")
					continue
				}
				if errors.Is(err, model.ErrFinished) {
					m.log.Debug().Msg("no more sealed blocks available")
					return nil
				}
				if err != nil {
					return fmt.Errorf("could not forward chain: %w", err)
				}
				break
			}
		}
	}
}
