package mapper

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dgraph-io/badger/v2"
	"github.com/rs/zerolog"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/complete/wal"
)

// TransitionFunc is a function that is applied onto the state machine's
// state.
type TransitionFunc func(*State) error

// Transitions is what applies transitions to the state of an FSM.
type Transitions struct {
	cfg     Config
	log     zerolog.Logger
	chain   archive.Chain
	updates TrieUpdates
	read    archive.Reader
	write   archive.Writer
	once    *sync.Once
}

// NewTransitions returns a Transitions component using the given dependencies and using the given options
func NewTransitions(log zerolog.Logger, chain archive.Chain, updates TrieUpdates, read archive.Reader, write archive.Writer, options ...Option) *Transitions {

	cfg := DefaultConfig
	for _, option := range options {
		option(&cfg)
	}

	t := Transitions{
		log:     log.With().Str("component", "mapper_transitions").Logger(),
		cfg:     cfg,
		chain:   chain,
		updates: updates,
		read:    read,
		write:   write,
		once:    &sync.Once{},
	}

	return &t
}

// InitializeMapper initializes the mapper by either going into bootstrapping or
// into resuming, depending on the configuration.
func (t *Transitions) InitializeMapper(s *State) error {
	if s.status != StatusInitialize {
		return fmt.Errorf("invalid status for initializing mapper (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Logger()

	isBootstrapped := false // default

	first, err := t.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}

	last, err := t.read.Last()
	if err == nil {
		isBootstrapped = last >= first
	} else if !errors.Is(err, badger.ErrKeyNotFound) {
		return fmt.Errorf("could not get last height: %w", err)
	}

	log.Info().
		Bool("is_bootstrapped", isBootstrapped).
		Uint64("first", first).
		Uint64("last", last).
		Msg("initializing")

	if isBootstrapped {
		// we have bootstrapped
		s.status = StatusResume
		return nil
	}

	s.status = StatusBootstrap
	s.height = first
	return nil
}

// BootstrapState bootstraps the state by loading the checkpoint if there is one
// and initializing the elements subsequently used by the FSM.
func (t *Transitions) BootstrapState(s *State) error {
	if s.status != StatusBootstrap {
		return fmt.Errorf("invalid status for bootstrapping state (%s)", s.status)
	}

	first, err := t.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}
	t.once.Do(func() { err = t.write.First(first) })
	if err != nil {
		return fmt.Errorf("could not write first: %w", err)
	}

	log := t.log.With().Uint64("height", s.height).Logger()

	log.Info().Msgf("bootstrap with checkpoint file %v%v", s.checkpointDir, s.checkpointFileName)

	// read leaf will be blocked if the consumer is not processing the leaf nodes fast
	// enough, which also help limit the amount of memory being used for holding unprocessed
	// leaf nodes.
	bufSize := 1000
	leafNodesCh := make(chan *wal.LeafNode, bufSize)

	batchSize := 1000
	batch := make([]*wal.LeafNode, 0, batchSize)
	total := 0

	log.Info().Msgf("start processing leaf nodes with batchSize: %v", batchSize)

	nWorker := 10
	jobs := make(chan []*wal.LeafNode, nWorker)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	go func() {
		defer close(jobs)

		for leafNode := range leafNodesCh {
			total++
			batch = append(batch, leafNode)

			// save registers in batch, which could result better speed
			if len(batch) >= batchSize {
				jobs <- batch
				batch = make([]*wal.LeafNode, 0, batchSize)
			}
		}

		if len(batch) > 0 {
			jobs <- batch
		}
	}()

	// reading the leaf nodes on a goroutine
	// and use a doneRead channel to report if running into any error
	doneRead := make(chan error, 1)
	go func() {
		err := wal.OpenAndReadLeafNodesFromCheckpointV6(leafNodesCh, s.checkpointDir, s.checkpointFileName, &t.log)
		if err != nil {
			log.Error().Err(err).Msg("fail to read leaf node")
			// if running into error, then cancel the context to stop all workers
			workerCancel()
		}
		doneRead <- err
		close(doneRead)
	}()

	// wait for all workers to finish in order to close the results channel
	var wg sync.WaitGroup
	wg.Add(nWorker)

	// create enough buffer to hold the max number of errors,
	// there will be at most nWorker number of errors, because a worker
	// will stop as soon as hitting an error
	workerErrors := make(chan error, nWorker)
	// create nWorker number of workers to process jobs.
	// each worker will save a batch of registers to storage
	for w := 0; w < nWorker; w++ {
		go func() {
			defer wg.Done()

			// process batches from jobs channel
			for batch := range jobs {
				err := t.write.Registers(s.height, batch)

				if err != nil {
					workerErrors <- err
					// if worker is running into exception, then
					// cancel the context to stop other workers.
					workerCancel()
					return
				}

				select {
				case <-workerCtx.Done():
					// we don't want to push workerCancel.Err() to workerErrors because,
					// that would print the context cancelled error
					// instead of the root cause error from the job creator.
					return
				default:
				}
			}
		}()
	}

	wg.Wait()
	close(workerErrors)

	// read from results channel and check for errors
	for err := range workerErrors {
		if err != nil {
			return err
		}
	}

	// make sure there is no error from reading the leaf node
	err = <-doneRead
	if err != nil {
		return fmt.Errorf("fail to read leaf node: %w", err)
	}

	log.Info().Msgf("finish importing payloads to storage for height %v, %v payloads", s.height, total)

	// We have successfully bootstrapped. However, no chain data for the root
	// block has been indexed yet. This is why we "pretend" that we just
	// forwarded the state to this height, so we go straight to the chain data
	// indexing.
	s.status = StatusResume // StatusResume will save the height

	// we have imported all payloads, we can skip the StatusCollect and StatusMap
	// status which is only needed after the bootstrap
	// since t.updates.AllUpdates() returns nil for bootstrap case, we will do nothing
	// in the StatusCollect and StatusMap status, which is equivalent to skipping

	return nil
}

// ResumeIndexing resumes indexing the data from a previous run.
func (t *Transitions) ResumeIndexing(s *State) error {
	if s.status != StatusResume {
		return fmt.Errorf("invalid status for resuming indexing (%s)", s.status)
	}
	// When resuming, we want to avoid overwriting the `first` height in the
	// index with the height we are resuming from. Theoretically, all that would
	// be needed would be to execute a no-op on `once`, which would subsequently
	// be skipped in the height forwarding code. However, this bug was already
	// released, so we have databases where `first` was incorrectly set to the
	// height we resume from. In order to fix them, we explicitly write the
	// correct `first` height here again, while at the same time using `once` to
	// disable any subsequent attempts to write it.
	first, err := t.chain.Root()
	if err != nil {
		return fmt.Errorf("could not get root height: %w", err)
	}
	t.once.Do(func() { err = t.write.First(first) })
	if err != nil {
		return fmt.Errorf("could not write first: %w", err)
	}

	// We need to know what the last indexed height was at the point we stopped
	// indexing.
	last, err := t.read.Last()
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			last = first
		} else {
			return fmt.Errorf("could not get last height: %w", err)
		}
	}

	// We just need to point to the next height. The chain indexing will
	// then proceed with the first non-indexed block and index values
	s.height = last + 1

	// At this point, we should be able to start indexing the register data for
	// the next height.
	s.status = StatusUpdate
	return nil
}

// UpdateTree gets all trie updates and stores it in the state
func (t *Transitions) UpdateTree(s *State) error {
	if s.status != StatusUpdate {
		return fmt.Errorf("invalid status for updating tree (%s)", s.status)
	}
	log := t.log.With().Uint64("height", s.height).Logger()
	// grab updates and move on to next state
	updates, err := t.updates.AllUpdates()
	if errors.Is(err, archive.ErrUnavailable) {
		time.Sleep(t.cfg.WaitInterval)
		log.Debug().Msg("waiting for next trie update")
		return nil
	}
	if err != nil {
		return fmt.Errorf("unable to retrieve trie updates for block height %x", s.height)
	}
	s.updates = updates
	log.Info().Int("updates", len(s.updates)).Msg("collected trie updates (registers) to be mapped")

	// Now that we have collected TrieUpdates for the current block,
	// we can create a path -> payload mapping that we will use to index to the local storage
	s.status = StatusCollect
	return nil
}

// CollectRegisters reads the payloads for the next block to be indexed from the state's forest, unless payload
// indexing is disabled.
func (t *Transitions) CollectRegisters(s *State) error {
	if s.status != StatusCollect {
		return fmt.Errorf("invalid status for collecting registers (%s)", s.status)
	}

	// If indexing payloads is disabled, we can bypass collection and indexing
	// of payloads and just go straight to indexing chain data
	if t.cfg.SkipRegisters {
		s.status = StatusIndex
		return nil
	}
	// collect paths/payload combinations
	for _, update := range s.updates {
		// guard for bootstrap case where t.updates.AllUpdates() returns nil as the queue is empty
		if update != nil {
			for i, path := range update.Paths {
				s.registers[path] = update.Payloads[i]
			}
		}
	}

	// At this point, we have collected all the payloads, so we go to the next
	// step, where we will index them.
	s.status = StatusMap
	return nil
}

// MapRegisters maps the collected registers to the current block.
func (t *Transitions) MapRegisters(s *State) error {
	if s.status != StatusMap {
		return fmt.Errorf("invalid status for indexing registers (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Logger()
	// If there are no registers left to be indexed, we can go to the next step,
	// which is indexing the chain data
	if len(s.registers) == 0 {
		s.status = StatusIndex
		return nil
	}

	// We will now collect and index 1000 registers at a time. This gives the
	// FSM the chance to exit the loop between every 1000 payloads we index. It
	// doesn't really matter for badger if they are in random order, so this
	// way of iterating should be fine.
	n := 1000
	payloads := make([]*ledger.Payload, 0, n)
	for path, payload := range s.registers {
		payloads = append(payloads, payload)
		delete(s.registers, path)
		if len(payloads) >= n {
			break
		}
	}

	// Then we store the (maximum) 1000 paths and payloads.
	err := t.write.Payloads(s.height, payloads)
	if err != nil {
		return fmt.Errorf("could not index registers: %w", err)
	}

	// skip the map status again, and its log
	if len(s.registers) == 0 {
		s.status = StatusIndex
		return nil
	}

	log.Debug().
		Int("batch", len(payloads)).
		Int("remaining", len(s.registers)).
		Msg("indexed register batch for finalized block")

	return nil
}

// IndexChain indexes chain data for the current height.
func (t *Transitions) IndexChain(s *State) error {
	if s.status != StatusIndex {
		return fmt.Errorf("invalid status for indexing chain (%s)", s.status)
	}

	log := t.log.With().Uint64("height", s.height).Logger()
	// We try to retrieve the next header until it becomes available, which
	// means all data coming from the protocol state is available after this
	// point.
	header, err := t.chain.Header(s.height)
	if errors.Is(err, archive.ErrUnavailable) {
		log.Info().Msgf("header is not available, sleep for %s", t.cfg.WaitInterval)
		time.Sleep(t.cfg.WaitInterval)
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not get header: %w", err)
	}

	// At this point, we can retrieve the data from the consensus state. This is
	// a slight optimization for the live indexer, as it allows us to process
	// some data before the full execution data becomes available.
	guarantees, err := t.chain.Guarantees(s.height)
	if err != nil {
		return fmt.Errorf("could not get guarantees: %w", err)
	}
	seals, err := t.chain.Seals(s.height)
	if err != nil {
		return fmt.Errorf("could not get seals: %w", err)
	}

	// We can also proceed to already indexing the data related to the consensus
	// state, before dealing with anything related to execution data, which
	// might go into the wait state.
	blockID := header.ID()
	err = t.write.Height(blockID, s.height)
	if err != nil {
		return fmt.Errorf("could not index height: %w", err)
	}
	err = t.write.Header(s.height, header)
	if err != nil {
		return fmt.Errorf("could not index header: %w", err)
	}
	err = t.write.Guarantees(s.height, guarantees)
	if err != nil {
		return fmt.Errorf("could not index guarantees: %w", err)
	}
	err = t.write.Seals(s.height, seals)
	if err != nil {
		return fmt.Errorf("could not index seals: %w", err)
	}

	// Next, we try to retrieve the next commit until it becomes available,
	// at which point all the data coming from the execution data should be
	// available.
	commit, err := t.chain.Commit(s.height)
	if errors.Is(err, archive.ErrUnavailable) {
		log.Debug().Msg("waiting for next state commitment")
		time.Sleep(t.cfg.WaitInterval)
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not get commit: %w", err)
	}
	collections, err := t.chain.Collections(s.height)
	if err != nil {
		return fmt.Errorf("could not get collections: %w", err)
	}
	transactions, err := t.chain.Transactions(s.height)
	if err != nil {
		return fmt.Errorf("could not get transactions: %w", err)
	}
	results, err := t.chain.Results(s.height)
	if err != nil {
		return fmt.Errorf("could not get transaction results: %w", err)
	}
	events, err := t.chain.Events(s.height)
	if err != nil {
		return fmt.Errorf("could not get events: %w", err)
	}

	// Next, all we need to do is index the remaining data and we have fully
	// processed indexing for this block height.
	err = t.write.Commit(s.height, commit)
	if err != nil {
		return fmt.Errorf("could not index commit: %w", err)
	}
	err = t.write.Collections(s.height, collections)
	if err != nil {
		return fmt.Errorf("could not index collections: %w", err)
	}
	err = t.write.Transactions(s.height, transactions)
	if err != nil {
		return fmt.Errorf("could not index transactions: %w", err)
	}
	err = t.write.Results(results)
	if err != nil {
		return fmt.Errorf("could not index transaction results: %w", err)
	}
	err = t.write.Events(s.height, events)
	if err != nil {
		return fmt.Errorf("could not index events: %w", err)
	}

	t.log.Info().
		Uint64("height", s.height).
		Str("block_id", blockID.String()).
		Int("n_guarantees", len(guarantees)).
		Int("n_seals", len(seals)).
		Hex("commit", commit[:]).
		Int("n_col", len(collections)).
		Int("n_tx", len(transactions)).
		Int("n_result", len(results)).
		Int("n_events", len(events)).
		Msgf("successfully indexed block")

	// After indexing the blockchain data, we can move on to the next height
	s.status = StatusForward
	return nil
}

// ForwardHeight increments the height at which the mapping operates, and updates the last indexed height.
func (t *Transitions) ForwardHeight(s *State) error {
	if s.status != StatusForward {
		return fmt.Errorf("invalid status for forwarding height (%s)", s.status)
	}

	// After finishing the indexing of the payloads for a finalized block, or
	// skipping it, we should document the last indexed height. On the first
	// pass, we will also index the first indexed height here.
	var err error
	t.once.Do(func() { err = t.write.First(s.height) })
	if err != nil {
		return fmt.Errorf("could not index first height: %w", err)
	}
	err = t.write.Last(s.height)
	if err != nil {
		return fmt.Errorf("could not index last height: %w", err)
	}

	// Now that we have indexed the heights, we can forward to the next height
	s.height++

	// Once the height is forwarded, we can set the status so that we index
	// the blockchain data next.
	s.status = StatusIndex
	return nil
}
