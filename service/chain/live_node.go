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

package chain

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/req"
	"go.nanomsg.org/mangos/v3/protocol/sub"
	_ "go.nanomsg.org/mangos/v3/transport/all" // register transports

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type Synchronizer interface {
	GetBlockHeight() (uint64, error)
}

type Live struct {
	log zerolog.Logger

	sync Synchronizer

	node dps.LiveNodeConfig
	sub  mangos.Socket
	req  mangos.Socket

	compressor   *zstd.Encoder
	decompressor *zstd.Decoder

	blocks blocks

	done chan struct{}
}

func FromLiveNode(log zerolog.Logger, liveNode dps.LiveNodeConfig, sync Synchronizer) (*Live, error) {
	compressor, err := zstd.NewWriter(nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize compressor: %w", err)
	}

	decompressor, err := zstd.NewReader(nil)
	if err != nil {
		return nil, fmt.Errorf("could not initialize decompressor: %w", err)
	}

	live := Live{
		log: log.With().Str("component", "live-chain").Logger(),

		sync: sync,

		node: liveNode,

		blocks: newBlocks(),

		compressor:   compressor,
		decompressor: decompressor,

		done: make(chan struct{}),
	}

	return &live, nil
}

func (l *Live) Run() error {
	select {
	case <-l.done:
		return nil
	default:
	}

	// Connect to the execution node's sockets.
	err := l.dial()
	if err != nil {
		return fmt.Errorf("could not connect to execution node: %w", err)
	}

	// Upon starting up, retrieve the first three blocks in order to know the current
	// sealed state of the chain, and how far the live chain needs to catch up.
	// The first block's height is the reference to set the upper boundary of the
	// synchronization request.
	current, err := l.subBlock()
	if err != nil {
		return fmt.Errorf("could not get initial blocks: %w", err)
	}
	err = l.blocks.cache(current)
	if err != nil {
		return fmt.Errorf("could not get cache block: %w", err)
	}
	to := current.Header.Height

	// Retrieve two more blocks after the initial one, to make sure that the initial block
	// is sealed and that the live node is able to produce a sync response that includes
	// all the requested blocks.
	for i := 0; i < 2; i++ {
		block, err := l.subBlock()
		if err != nil {
			return fmt.Errorf("could not get initial blocks: %w", err)
		}

		err = l.blocks.cache(block)
		if err != nil {
			return fmt.Errorf("could not get cache block: %w", err)
		}
	}

	from, err := l.sync.GetBlockHeight()
	if err != nil {
		return fmt.Errorf("could not get current state height: %w", err)
	}

	// Synchronize with the execution node to receive all missing blocks
	// in order.
	wg := &sync.WaitGroup{}
	wg.Add(2)
	stopCache := make(chan struct{})
	syncFailed := make(chan error)
	go func() {
		defer wg.Done()
		defer close(stopCache)

		for {
			l.log.Info().Uint64("from", from).Uint64("to", to).Msg("requesting block update sync")

			// Request blocks between two given heights. The node might respond with only a subset of the requested
			// range, if it is too big, in which case the Live Feeder keeps sending sync requests until it has caught up
			// with the blocks from the sub socket.
			err = l.reqSync(from, to)
			if err != nil {
				l.log.Error().Err(err).Msg("could not send block sync request")
				syncFailed <- err
				return
			}

			blocks, err := l.recvSync()
			if err != nil {
				l.log.Error().Err(err).Msg("could not process block sync response")
				syncFailed <- err
				return
			}

			l.log.Info().Int("sync_blocks", len(blocks)).Msg("received block sync response")

			for _, block := range blocks {
				l.log.Debug().Uint64("height", block.Header.Height).Msg("received sync block")

				err := l.blocks.add(block)
				if err != nil {
					l.log.Error().Err(err).Msg("could not process block sync response")
					syncFailed <- err
					return
				}

				// End of sync reached, stop this routine.
				if block.Header.Height == to {
					l.log.Debug().Uint64("height", block.Header.Height).Msg("finished sync!")
					return
				}

				// Always keep `from` at the height of the last received block, to avoid requesting multiple times the same data.
				from = block.Header.Height
			}
		}
	}()

	go func() {
		defer wg.Done()

		for {
			select {
			case <-l.done:
				return
			case <-stopCache:
				return
			default:
			}

			block, err := l.subBlock()
			if err != nil {
				l.log.Error().Err(err).Msg("could not receive published block update while waiting for sync response")
				return
			}

			println("BEFORE CACHE BLOCK")
			err = l.blocks.cache(block)
			if err != nil {
				l.log.Error().Err(err).Msg("received invalid published block update while waiting for sync response")
				return
			}

			l.log.Info().Msg("received block while waiting for sync response")
		}
	}()

	// Signals that both the caching and syncing routines are done.
	wait := make(chan struct{})
	go func() {
		wg.Wait()
		close(wait)
	}()

	// Wait for cache to be filled with all required updates to catch up, for the sync process to have failed or for
	// the chain to be ordered to shut down.
	select {
	case <-l.done:
		return nil
	case err := <-syncFailed:
		return err
	case <-wait:
		println("DONE WAITING FOR BOTH ROUTINES")
	}

	// Consume the cache and push its elements to the update channel for
	// the mapper to use.
	println("BEFORE CONSUME CACHE")
	l.blocks.consumeCache()

	l.log.Info().Msg("bootstrapping done, begin live feeding")

	// Start following live updates.
	for {
		select {
		case <-l.done:
			return nil
		default:
		}

		block, err := l.subBlock()
		if err != nil {
			return err
		}

		err = l.blocks.add(block)
		if err != nil {
			return err
		}
	}
}

func (l *Live) Root() (uint64, error) {
	// Find the lowest height we have available, to use as root height.
	rootBlock, err := l.blocks.first()
	if err != nil {
		return 0, dps.ErrTimeout
	}

	return rootBlock.Header.Height, nil
}

func (l *Live) Commit(height uint64) (flow.StateCommitment, error) {
	// Verify that at least three new blocks have been generated, which
	// means the block at the height that is being queried is part of
	// the canonical chain.
	sealedBlock, err := l.blocks.get(height + 3)
	if err != nil {
		return flow.StateCommitment{}, dps.ErrTimeout
	}

	// Verify that the height at which we want to get the state
	// commitment is also available in our map.
	wantBlock, err := l.blocks.get(height)
	if err != nil {
		return flow.StateCommitment{}, dps.ErrTimeout
	}

	// Look for the seal of the block whose state we want.
	var seal *flow.Seal
	for _, s := range sealedBlock.Payload.Seals {
		if s.BlockID == wantBlock.ID() {
			seal = s
		}
	}

	// If none is found, time out.
	if seal == nil {
		return flow.StateCommitment{}, dps.ErrTimeout
	}

	return seal.FinalState, nil
}

func (l *Live) Header(height uint64) (*flow.Header, error) {
	// Verify that at least three new blocks have been generated, which
	// means the block at the height that is being queried is part of
	// the canonical chain.
	_, err := l.blocks.get(height + 3)
	if err != nil {
		return nil, dps.ErrTimeout
	}

	block, err := l.blocks.get(height)
	if err != nil {
		return nil, dps.ErrTimeout
	}

	// Delete block since Header is the last call for each block from the mapper
	l.blocks.delete(height)

	return block.Header, nil
}

func (l *Live) Events(_ uint64) ([]flow.Event, error) {
	// Not implemented, but should not panic since it is used.
	// TODO: Implement this. (https://github.com/optakt/flow-dps/issues/99)
	return nil, nil
}

func (l *Live) Stop() {
	close(l.done)
}

func (l *Live) dial() error {
	psock, err := sub.NewSocket()
	if err != nil {
		return fmt.Errorf("could not get new sub socket: %w", err)
	}

	// TODO: Switch to WSS (https://github.com/optakt/flow-dps/issues/97)
	err = psock.Dial(fmt.Sprint("ws://", l.node.Host, ":", l.node.SubPort))
	if err != nil {
		return fmt.Errorf("could not dial on sub socket: %w", err)
	}
	err = psock.SetOption(mangos.OptionSubscribe, []byte("pub_block"))
	if err != nil {
		return fmt.Errorf("could not subscribe: %w", err)
	}

	rsock, err := req.NewSocket()
	if err != nil {
		return fmt.Errorf("could not get new req socket: %w", err)
	}
	err = rsock.Dial(fmt.Sprint("tcp://", l.node.Host, ":", l.node.ReqPort))
	if err != nil {
		return fmt.Errorf("could not dial on req socket: %w", err)
	}

	l.sub = psock
	l.req = rsock

	return nil
}

func (l *Live) subBlock() (*flow.Block, error) {
	msg, err := l.sub.Recv()
	if err != nil {
		l.log.Warn().Err(err).Msg("could not receive block")
		return nil, err
	}

	l.log.Info().Msg("received block update")

	// Get payload from message by removing topic header.
	payload := bytes.TrimPrefix(msg, []byte("pub_block"))

	decoded, err := l.decompressor.DecodeAll(payload, nil)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decompress block payload")
		return nil, err
	}

	var block flow.Block
	err = cbor.Unmarshal(decoded, &block)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decode block payload")
		return nil, err
	}

	return &block, nil
}

func (l *Live) reqSync(from, to uint64) error {
	request := struct {
		From uint64
		To   uint64
	}{
		From: from,
		To:   to,
	}

	payload, err := cbor.Marshal(request)
	if err != nil {
		return err
	}

	compressed := l.compressor.EncodeAll(payload, nil)

	// Add header "topic".
	message := append([]byte(`sync_block`), compressed...)

	l.log.Debug().Uint64("from", from).Uint64("to", to).Msg("sending sync request")

	return l.req.Send(message)
}

func (l *Live) recvSync() ([]*flow.Block, error) {
	msg, err := l.req.Recv()
	if err != nil {
		l.log.Warn().Err(err).Msg("could not receive sync response")
		return nil, err
	}

	l.log.Info().Msg("received block sync response")

	payload, err := l.decompressor.DecodeAll(msg, nil)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decompress block sync payload")
		return nil, err
	}

	// Check if the sync server answered with an error response.
	var syncErr error
	err = cbor.Unmarshal(payload, &syncErr)
	if err == nil {
		return nil, syncErr
	}

	// If it did not, the message contains a block and it can be unmarshalled.
	var blocks []*flow.Block
	err = cbor.Unmarshal(payload, &blocks)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decode block sync payload")
		return nil, err
	}

	return blocks, nil
}

// This structure is a simple short-term cache for blocks.
type blocks struct {
	// The cached map is used to cache blocks that should not be
	// used yet, as they do not follow the current state of the
	// chain. It is merged with the blocks map once the
	// synchronization process is complete.
	cached map[uint64]*flow.Block

	// The blocks map contains the blocks that should be handled by the
	// mapper immediately. While the sync process takes place, it is
	// progressively filled. Once the sync process is over, it is kept
	// up-to-date by the pub/sub mechanism.
	blocksMu sync.RWMutex
	blocks   map[uint64]*flow.Block
}

func newBlocks() blocks {
	println("INIT MAP")
	return blocks{
		cached: make(map[uint64]*flow.Block),
		blocks: make(map[uint64]*flow.Block),
	}
}

func (b *blocks) add(block *flow.Block) error {
	if block == nil {
		return errors.New("nil block")
	}

	b.blocksMu.Lock()
	b.blocks[block.Header.Height] = block
	b.blocksMu.Unlock()

	return nil
}

func (b *blocks) cache(block *flow.Block) error {
	if block == nil {
		return errors.New("nil block")
	}
	println("WRITE IN CACHE")

	b.cached[block.Header.Height] = block

	println("WRITE IN CACHE ALL GOOD")

	return nil
}

func (b *blocks) consumeCache() {
	println("EAT CACHE")

	b.blocksMu.Lock()
	for height, block := range b.cached {
		b.blocks[height] = block
	}
	b.blocksMu.Unlock()

	b.cached = nil
}

func (b *blocks) get(height uint64) (*flow.Block, error) {
	b.blocksMu.RLock()
	block, exists := b.blocks[height]
	b.blocksMu.RUnlock()

	if !exists {
		return nil, errors.New("block not found")
	}

	return block, nil
}

func (b *blocks) delete(height uint64) {
	b.blocksMu.Lock()
	delete(b.blocks, height)
	b.blocksMu.Unlock()
}

func (b *blocks) first() (*flow.Block, error) {
	b.blocksMu.RLock()
	defer b.blocksMu.RUnlock()

	if len(b.blocks) == 0 {
		return nil, errors.New("no blocks ready")
	}

	var minHeight uint64
	for minHeight = range b.blocks {
		break
	}
	for height := range b.blocks {
		if height < minHeight {
			minHeight = height
		}
	}

	return b.blocks[minHeight], nil
}
