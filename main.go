package main

import (
	"errors"
	"log"

	"github.com/awfm9/flow-dps/ral"
	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/tsdb/wal"

	exec "github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
)

func main() {

	// As a first step, we load the protocol state database in order to identify
	// the root block and the root state commitment to bootstrap the ledger from.
	opts := badger.DefaultOptions("data").WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatal(err)
	}
	var height uint64
	err = operation.RetrieveRootHeight(&height)(db.NewTransaction(false))
	if err != nil {
		log.Fatal(err)
	}
	var rootID flow.Identifier
	err = operation.LookupBlockHeight(height, &rootID)(db.NewTransaction(false))
	if err != nil {
		log.Fatal(err)
	}
	var sealID flow.Identifier
	err = operation.LookupBlockSeal(rootID, &sealID)(db.NewTransaction(false))
	if err != nil {
		log.Fatal(err)
	}
	var seal flow.Seal
	err = operation.RetrieveSeal(sealID, &seal)(db.NewTransaction(false))
	if err != nil {
		log.Fatal(err)
	}

	// In the second step, we use this information to bootstrap a random access
	// ledger streamer, that allows us to stream data into a ledger that can
	// access any register at any block height.
	streamer, err := ral.NewStreamer(rootID, seal.FinalState)
	if err != nil {
		log.Fatal(err)
	}

	// The third step is about pushing all of the block and seal information
	// into the streamer, so it knows which commits to look for in the stream
	// of updates that change the state root hash.
	parentID := rootID
	var blockID flow.Identifier
	for {
		height++
		err = operation.LookupBlockHeight(height, &blockID)(db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			break
		}
		err = streamer.Block(parentID, blockID)
		if err != nil {
			log.Fatal(err)
		}
		parentID = blockID
		err = operation.LookupBlockSeal(blockID, &sealID)(db.NewTransaction(false))
		if errors.Is(err, storage.ErrNotFound) {
			continue
		}
		if err != nil {
			log.Fatal(err)
		}
		err = operation.RetrieveSeal(sealID, &seal)(db.NewTransaction(false))
		if err != nil {
			log.Fatal(err)
		}
		err = streamer.Seal(blockID, seal.FinalState)
		if err != nil {
			log.Fatal(err)
		}
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
		err = streamer.Delta(update.RootHash, delta)
		if err != nil {
			log.Fatal(err)
		}
	}
}
