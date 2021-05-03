package main

import (
	"os"
	"os/signal"
	"time"

	"github.com/awfm9/flow-dps/ral"
	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/tsdb/wal"
	"github.com/rs/zerolog"

	exec "github.com/onflow/flow-go/ledger/complete/wal"
)

func main() {

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)

	// Initialize the badger database that contains the protocol state data.
	data, err := badger.Open(badger.DefaultOptions("data").WithLogger(nil))
	if err != nil {
		log.Fatal().Err(err)
	}

	// Initialize the badger database for the random access ledger index.
	index, err := badger.Open(badger.DefaultOptions("index").WithLogger(nil))
	if err != nil {
		log.Fatal().Err(err)
	}

	// Initialize the random access ledger core.
	core, err := ral.NewCore(log, index)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Initialize the static random access ledger that uses the protocol state
	// data to index execution state updates.
	static, err := ral.NewStatic(log, core, data)
	if err != nil {
		log.Fatal().Err(err)
	}

	// Lastly, we can stream the updates from the write-ahead log into the
	// streamer to map them to blocks according to the information it has.
	w, err := wal.NewSize(
		nil,
		prometheus.DefaultRegisterer,
		"trie",
		32*1024*1024,
	)
	if err != nil {
		log.Fatal().Err(err)
	}
	r, err := wal.NewSegmentsReader(w.Dir())
	if err != nil {
		log.Fatal().Err(err)
	}
	s := wal.NewReader(r)
	for s.Next() {
		select {
		case <-sig:
			os.Exit(0)
		default:
		}
		operation, _, update, err := exec.Decode(s.Record())
		if err != nil {
			log.Fatal().Err(err)
		}
		if operation != exec.WALUpdate {
			continue
		}
		delta := make(ral.Delta, 0, len(update.Paths))
		for index, path := range update.Paths {
			payload := *update.Payloads[index]
			change := ral.Change{
				Path:    path,
				Payload: payload,
			}
			delta = append(delta, change)
		}
		err = static.Delta(update.RootHash, delta)
		if err != nil {
			log.Fatal().Err(err)
		}
	}
}
