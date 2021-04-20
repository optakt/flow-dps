package main

import (
	"bytes"
	"log"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/prometheus/client_golang/prometheus"
	pwal "github.com/prometheus/tsdb/wal"
)

func main() {

	// `--mtrie-cache-size` == 1000 => `capacity`
	// `ledger.DefaultPathFinderVersion` == 1
	// `pathfinder.PathByteSize` == 32

	// We initialize an empty trie with the default path length.
	t, err := trie.NewEmptyMTrie(pathfinder.PathByteSize)
	if err != nil {
		log.Fatal(err)
	}

	// We initialize a native prometheus WAL here, rather than our wrapper.
	w, err := pwal.NewSize(
		nil,
		prometheus.DefaultRegisterer,
		"trie",
		32*1024*1024,
	)
	if err != nil {
		log.Fatal(err)
	}

	// We prepare for reading all segments of the WAL.
	from, to, err := w.Segments()
	if err != nil {
		log.Fatal(err)
	}
	r, err := pwal.NewSegmentsRangeReader(pwal.SegmentRange{
		Dir:   w.Dir(),
		First: from,
		Last:  to,
	})
	if err != nil {
		log.Fatal(err)
	}

	// We step through the records one by one, trying to decode.
	s := pwal.NewReader(r)
	for s.Next() {

		// We decode the operation, the root hash and the potential update.
		operation, _, update, err := wal.Decode(s.Record())
		if err != nil {
			log.Fatal(err)
		}

		// Then, we apply updates and deletes appropriately.
		switch operation {

		// For updates, we update the underlying trie and discard the previous version.
		case wal.WALUpdate:

			if !bytes.Equal(t.RootHash(), update.RootHash) {
				log.Fatal("mismatched root hash for update")
			}

			payloads := make([]ledger.Payload, 0, len(update.Payloads))
			for _, payload := range update.Payloads {
				payloads = append(payloads, *payload)
			}
			t, err = trie.NewTrieWithUpdatedRegisters(t, update.Paths, payloads)
			if err != nil {
				log.Fatal(err)
			}

		// For deletes, we simply drop the branch of the sub-trie in question.
		case wal.WALDelete:
			// Note: these are actually irrelevant, as it's about deleting tries
			// from the cache, not register values.
		}
	}
}
