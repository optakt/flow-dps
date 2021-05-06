package mapper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/awfm9/flow-dps/model"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// Static is a random access ledger that bootstraps the index from a static
// snapshot of the protocol state and the corresponding ledger write-ahead log.
type Mapper struct {
	log     zerolog.Logger
	chain   Chain
	feeder  Feeder
	indexer Indexer
	trie    *trie.MTrie
	deltas  []model.Delta
	wg      *sync.WaitGroup
	stop    chan struct{}
}

// New creates a new mapper that uses chain data to map trie updates to blocks
// and then passes on the details to the indexer for indexing.
func New(log zerolog.Logger, chain Chain, feeder Feeder, indexer Indexer, options ...func(*MapperConfig)) (*Mapper, error) {

	// By default, we don't have a checkpoint to bootstrap from, so check if we
	// explicitly passed one using the variadic option parameters.
	cfg := MapperConfig{
		CheckpointFile: "",
	}
	for _, option := range options {
		option(&cfg)
	}

	// If we have a checkpoint file, it should be a root checkpoint, so it
	// should only contain a single trie that we load as our initial root state.
	// Otherwise, the root state is an empty memory trie.
	var t *trie.MTrie
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
		t = tries[0]
	} else {
		trie, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
		if err != nil {
			return nil, fmt.Errorf("could not initialize empty memory trie: %w", err)
		}
		t = trie
	}

	// NOTE: there might be a number of trie updates in the WAL before the root
	// block, which means that we can not sanity check the state trie against
	// the root block state commitment here.

	m := &Mapper{
		log:     log,
		chain:   chain,
		feeder:  feeder,
		indexer: indexer,
		trie:    t,
		deltas:  []model.Delta{},
		wg:      &sync.WaitGroup{},
		stop:    make(chan struct{}),
	}

	return m, nil
}

func (m *Mapper) Stop(ctx context.Context) error {
	close(m.stop)
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (m *Mapper) Run() error {
	m.wg.Add(1)
	defer m.wg.Done()

	// The first loop is responsible for reading deltas from the feeder and
	// updating the state trie with the delta in order to get the next trie
	// state. If the feeder times out (which can happen, for example, for a
	// feeder receiving trie updates over the network), it will go into a tight
	// loop until the state delta to be applied to the trie with the current
	// root hash has been successfully retrieved.
	// NOTE: We moved the logic of the first loop behind the logic for the
	// second loop, as it allows us to map all matching blocks before retrieving
	// the next delta. This covers the edge case of trie updates before the root
	// block, including zero and many. It also covers the edge case of indexing
	// the last block, regardless of whether their are additional trie updates
	// behind it.
First:
	for {

		// The second loop is responsible for mapping the currently active block
		// to the set of deltas that were collected. If the state commitment for
		// the block we are looking for isn't the same as the trie root hash, we
		// will immediately go to the next iteration of the outer loop to keep
		// adding deltas to the trie. If it does match, we will index the block
		// with the set of deltas we collected. This might happen more than once
		// if no change to the state trie happens between multiple blocks, at
		// which point we map the second and any subsequent blocks without
		// change to an empty set of deltas.
		hash := m.trie.RootHash()
	Second:
		for {
			height, blockID, commit := m.chain.Active()
			if !bytes.Equal(commit, hash) {
				break Second
			}
			err := m.indexer.Index(height, blockID, commit, m.deltas)
			if err != nil {
				return fmt.Errorf("could not index deltas: %w (height: %d, block: %x, commit: %x)", err, height, blockID, commit)
			}

			m.log.Info().
				Uint64("height", height).
				Hex("block", blockID[:]).
				Hex("commit", commit).
				Int("deltas", len(m.deltas)).
				Msg("block deltas indexed")

			m.deltas = []model.Delta{}

			// The third loop is responsible for forwarding the chain to the
			// next block after each block indexing. This basically forwards the
			// pointer to the active block, for which we will look for the
			// state commitment in the trie next. The loop is required in case
			// there is a timeout (which can happen if we load the chain data
			// over the network), so that we only resume the rest of the logic
			// once we have successfully forwarded to the next block.
		Third:
			for {

				// We also want to check for shutdown before forwarding to the
				// next block; if the block isn't available, we could loop
				// indefinitely otherwise. It's important to break out of the
				// surrounding second loop, otherwise we could keep coming back
				// here.
				select {
				case <-m.stop:
					break Second
				default:
					// keep going
				}

				err = m.chain.Forward()
				if errors.Is(err, model.ErrTimeout) {
					m.log.Warn().Msg("chain forwarding has timed out")
					continue Third
				}
				if errors.Is(err, model.ErrFinished) {
					m.log.Debug().Msg("no more sealed blocks available")
					return nil
				}
				if err != nil {
					return fmt.Errorf("could not forward chain: %w", err)
				}

				break Third
			}
		}

		// We do want to check for shutdown before pulling the next delta; both
		// because it starts a new "round" of processing, and because it could
		// enter into a tight loop until a delta becomes available.
		select {
		case <-m.stop:
			break First
		default:
			// keep going
		}

		delta, err := m.feeder.Feed(hash)
		if errors.Is(err, model.ErrTimeout) {
			m.log.Warn().Msg("delta feeding has timed out")
			continue First
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

		m.log.Info().
			Hex("hash_before", hash).
			Hex("hash_after", trie.RootHash()).
			Int("changes", len(delta)).
			Msg("state trie updated")

		m.deltas = append(m.deltas, delta)
		m.trie = trie
	}

	return nil
}
