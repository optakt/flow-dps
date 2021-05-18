package feeder

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/ef-ds/deque"
	"github.com/fxamacker/cbor/v2"
	"github.com/klauspost/compress/zstd"
	"github.com/rs/zerolog"

	"go.nanomsg.org/mangos/v3"
	"go.nanomsg.org/mangos/v3/protocol/req"
	"go.nanomsg.org/mangos/v3/protocol/sub"
	_ "go.nanomsg.org/mangos/v3/transport/all" // register transports

	"github.com/onflow/flow-go/ledger"
	"github.com/optakt/flow-dps/models/dps"
)

type Synchronizer interface {
	GetRootHash() (ledger.RootHash, error)
}

type Live struct {
	log zerolog.Logger

	sync Synchronizer

	node dps.LiveNodeConfig

	sub mangos.Socket
	req mangos.Socket

	updates chan *ledger.TrieUpdate

	cache        deque.Deque
	compressor   *zstd.Encoder
	decompressor *zstd.Decoder

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
		log: log.With().Str("component", "live-feeder").Logger(),

		sync: sync,

		node: liveNode,

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

	// Upon starting up, retrieve the first update in order to know the current
	// state of the network, and how far the live feeder needs to catch up.
	current, err := l.subUpdate()
	if err != nil {
		return fmt.Errorf("could not get initial trie update: %w", err)
	}

	l.cache.PushBack(current)

	from, err := l.sync.GetRootHash()
	if err != nil {
		return fmt.Errorf("could not get current state root hash: %w", err)
	}

	to := current.RootHash

	// Retrieve 20 more trie updates after the initial one, to make sure that the initial one
	// is part of a sealed block and that the live node is able to produce a sync response that includes
	// all the requested trie updates.
	for i := 0; i < 20; i++ {
		update, err := l.subUpdate()
		if err != nil {
			return fmt.Errorf("could not get initial updates: %w", err)
		}

		l.cache.PushBack(update)
	}

	// Synchronize with the execution node to receive all missing updates
	// in order.
	wait := make(chan struct{})
	syncFailed := make(chan error)
	go func() {
		defer close(wait)

		for {
			l.log.Info().Hex("from", from[:]).Hex("to", to[:]).Msg("requesting trie update sync")

			// Request trie updates between two given root hashes. The node might respond with only a subset of the requested
			// range, if it is too big, in which case the Live Feeder keeps sending sync requests until it has caught up
			// with the updates from the sub socket.
			err = l.reqSync(from, to)
			if err != nil {
				l.log.Error().Err(err).Msg("could not send trie sync request")
				syncFailed <- err
				return
			}

			updates, err := l.recvSync()
			if err != nil {
				l.log.Error().Err(err).Msg("could not process trie sync response")
				syncFailed <- err
				return
			}

			l.log.Info().Int("sync_updates", len(updates)).Msg("received trie sync response")

			for _, update := range updates {
				l.log.Debug().Hex("hash", update.RootHash[:]).Msg("received sync response")

				l.updates <- &update

				// End of sync reached, stop this routine.
				if update.RootHash == to {
					return
				}

				// Always keep `from` at the rootHash of the last update, to avoid requesting multiple times the same data.
				from = update.RootHash
			}
		}
	}()

	go func() {
		for {
			select {
			case <-wait:
				return
			case <-l.done:
				return
			default:
			}

			update, err := l.subUpdate()
			if err != nil {
				l.log.Error().Err(err).Msg("could not receive published trie update while waiting for sync response")
				syncFailed <- err
				return
			}

			l.log.Info().Msg("caching update while waiting for sync response")

			l.cache.PushBack(update)
		}
	}()

	// Wait for cache to be filled with all required updates to catch up.
	select {
	case <-l.done:
		return nil
	case err := <-syncFailed:
		return err
	case <-wait:
	}

	l.log.Info().Msg("bootstrapping done, unstack cached updates")

	// Consume the cache and push its elements to the update channel for
	// the mapper to use.
	for l.cache.Len() > 0 {
		cached, _ := l.cache.PopFront()
		update, ok := cached.(*ledger.TrieUpdate)
		if !ok {
			return errors.New("invalid update in cache")
		}

		l.updates <- update
	}

	l.log.Info().Msg("cached updates consumed, begin live feeding")

	// Start following live updates.
	for {
		select {
		case <-l.done:
			return nil
		default:
		}

		update, err := l.subUpdate()
		if err != nil {
			return err
		}

		l.updates <- update
	}
}

func (l *Live) Update() (*ledger.TrieUpdate, error) {
	return <-l.updates, nil
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
	err = psock.SetOption(mangos.OptionSubscribe, []byte("pub_trie"))
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

func (l *Live) subUpdate() (*ledger.TrieUpdate, error) {
	msg, err := l.sub.Recv()
	if err != nil {
		l.log.Warn().Err(err).Msg("could not receive update")
		return nil, err
	}

	l.log.Info().Msg("received trie update")

	// Get payload from message by removing topic header.
	payload := bytes.TrimPrefix(msg, []byte("pub_trie"))

	decoded, err := l.decompressor.DecodeAll(payload, nil)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decompress trie update payload")
		return nil, err
	}

	var update ledger.TrieUpdate
	err = cbor.Unmarshal(decoded, &update)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decode trie update payload")
		return nil, err
	}

	return &update, nil
}

func (l *Live) reqSync(from, to ledger.RootHash) error {
	request := struct {
		From ledger.RootHash
		To   ledger.RootHash
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
	message := append([]byte(`sync_trie`), compressed...)

	l.log.Debug().Str("from", from.String()).Str("to", to.String()).Msg("sending sync request")

	return l.req.Send(message)
}

func (l *Live) recvSync() ([]ledger.TrieUpdate, error) {
	msg, err := l.req.Recv()
	if err != nil {
		l.log.Warn().Err(err).Msg("could not receive sync response")
		return nil, err
	}

	l.log.Info().Msg("received trie sync response")

	payload, err := l.decompressor.DecodeAll(msg, nil)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decompress trie sync payloads")
		return nil, err
	}

	// Check if the sync server answered with an error response.
	var syncErr error
	err = cbor.Unmarshal(payload, &syncErr)
	if err == nil {
		return nil, syncErr
	}

	// If it did not, the message contains a trie update and it can be unmarshalled.
	var updates []ledger.TrieUpdate
	err = cbor.Unmarshal(payload, &updates)
	if err != nil {
		l.log.Error().Err(err).Msg("unable to decode trie sync payload")
		return nil, err
	}

	return updates, nil
}
