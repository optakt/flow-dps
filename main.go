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
	"github.com/spf13/pflag"

	"github.com/onflow/flow-go/ledger/complete/mtrie/flattener"
	exec "github.com/onflow/flow-go/ledger/complete/wal"
)

func main() {

	var (
		flagLevel      string
		flagData       string
		flagTrie       string
		flagIndex      string
		flagCheckpoint string
	)

	pflag.StringVarP(&flagLevel, "log-level", "l", "info", "log level")
	pflag.StringVarP(&flagData, "data-dir", "d", "data", "protocol state data directory")
	pflag.StringVarP(&flagTrie, "trie-dir", "t", "trie", "execution state trie directory")
	pflag.StringVarP(&flagIndex, "index-dir", "i", "index", "dps state index directory")
	pflag.StringVarP(&flagCheckpoint, "checkpoint-file", "c", "root.checkpoint", "execution state trie root checkpoint")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr).With().Timestamp().Logger().Level(zerolog.DebugLevel)
	level, err := zerolog.ParseLevel(flagLevel)
	if err != nil {
		log.Fatal().Err(err)
	}
	log = log.Level(level)

	// Initialize the first checkpoint.
	file, err := os.Open(flagCheckpoint)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open checkpoint file")
	}
	flat, err := exec.ReadCheckpoint(file)
	if err != nil {
		log.Fatal().Err(err).Msg("could not decode flattened tries")
	}
	tries, err := flattener.RebuildTries(flat)
	if err != nil {
		log.Fatal().Err(err).Msg("could not rebuild memory tries")
	}
	if len(tries) != 1 {
		log.Fatal().Int("tries", len(tries)).Msg("should have exactly one memory trie")
	}

	// Initialize the badger database that contains the protocol state data.
	data, err := badger.Open(badger.DefaultOptions(flagData).WithLogger(nil))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open protocol state database")
	}

	// Initialize the badger database for the random access ledger index.
	index, err := badger.Open(badger.DefaultOptions(flagIndex).WithLogger(nil))
	if err != nil {
		log.Fatal().Err(err).Msg("could not open DPS state index")
	}

	// Initialize the random access ledger core.
	core, err := ral.NewCore(log, tries[0], index)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize DPS indexer")
	}

	// Initialize the static random access ledger that uses the protocol state
	// data to index execution state updates.
	static, err := ral.NewStatic(log, core, data)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize DPS streamer")
	}

	// Lastly, we can stream the updates from the write-ahead log into the
	// streamer to map them to blocks according to the information it has.
	w, err := wal.NewSize(
		nil,
		prometheus.DefaultRegisterer,
		flagTrie,
		32*1024*1024,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize WAL")
	}
	r, err := wal.NewSegmentsReader(w.Dir())
	if err != nil {
		log.Fatal().Err(err).Msg("could not initialize segments reader")
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
			log.Fatal().Err(err).Msg("could not decode WAL operation")
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
			log.Fatal().Err(err).Msg("could not stream state delta")
		}
	}

	// TODO: implement component interfaces
	// file := NewFilesytemChain(data)
	// core := NewCore(index)
	// streamer := NewLedgerWALStreamer(wal, file, core)

	// net := NewNetworkChain(access)
	// core := NewCore(index)
	// streamer := NewLiveStreamer(pub, net, core)
}
