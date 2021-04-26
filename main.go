package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/tsdb/wal"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	exec "github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/storage"
	"github.com/onflow/flow-go/storage/badger/operation"
)

func main() {

	// As the first step, we initialize the badger database and retrieve the
	// root height. The below loop uses the height as the pointer to identify
	// the next execution state checkpoint, so we can merge all updates for the
	// same block together.
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

	// In the second stop, we just create an empty trie to replay the updates
	// from the WAL into.
	t, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		log.Fatal(err)
	}

	// Finally, we initialize the reader for the write-ahead log.
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

	var commit flow.StateCommitment
TopLoop:
	for {

	SealLoop:
		for {
			// We get the seal so that we know which state commitment to look for
			// when replaying updates onto the trie.
			var blockID flow.Identifier
			err = operation.LookupBlockHeight(height, &blockID)(db.NewTransaction(false))
			if errors.Is(err, storage.ErrNotFound) {
				break TopLoop
			}
			if err != nil {
				log.Fatal(err)
			}
			var sealID flow.Identifier
			err = operation.LookupBlockSeal(blockID, &sealID)(db.NewTransaction(false))
			if err != nil {
				log.Fatal(err)
			}
			var seal flow.Seal
			err = operation.RetrieveSeal(sealID, &seal)(db.NewTransaction(false))
			if err != nil {
				log.Fatal(err)
			}
			if !bytes.Equal(seal.FinalState, commit) {
				commit = seal.FinalState
				break SealLoop
			}
			height++
		}

		// Now we play updates into it until we reach our state commitment.
		var updates []*ledger.TrieUpdate
	UpdateLoop:
		for s.Next() {

			// Decode the update and do a sanity check to see if it actually is
			// supposed to be applied on the trie in the given state.
			operation, _, update, err := exec.Decode(s.Record())
			if err != nil {
				log.Fatal(err)
			}
			if operation != exec.WALUpdate {
				continue
			}
			if !bytes.Equal(t.RootHash(), update.RootHash) {
				log.Fatal("mismatched root hash for update")
			}

			// Now we can play the update into the trie and get our next commitment.
			payloads := make([]ledger.Payload, 0, len(update.Payloads))
			for _, payload := range update.Payloads {
				payloads = append(payloads, *payload)
			}
			t, err = trie.NewTrieWithUpdatedRegisters(t, update.Paths, payloads)
			if err != nil {
				log.Fatal(err)
			}

			// Append the update to the list and check if we reached a block
			// checkpoint.
			updates = append(updates, update)
			if !bytes.Equal(t.RootHash(), commit) {
				continue
			}

			// At this point, we have reached the checkpoint and we should
			// compound.
			// TODO: actually compound and store stuff
			fmt.Printf("%x: %d update(s)\n", commit, len(updates))
			updates = nil
			height++
			break UpdateLoop
		}
	}
}
