package main

import (
	"log"

	"github.com/awfm9/flow-dps/ral"
	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/tsdb/wal"

	exec "github.com/onflow/flow-go/ledger/complete/wal"
)

func main() {

	// Initialize the badger database that contains the protocol state data.
	opts := badger.DefaultOptions("data").WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize the static random access ledger that uses the protocol state
	// data to index execution state updates.
	static, err := ral.NewStatic(db)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}
	r, err := wal.NewSegmentsReader(w.Dir())
	if err != nil {
		log.Fatal(err)
	}
	s := wal.NewReader(r)
	for s.Next() {
		operation, _, update, err := exec.Decode(s.Record())
		if err != nil {
			log.Fatal(err)
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
			log.Fatal(err)
		}
	}
}
