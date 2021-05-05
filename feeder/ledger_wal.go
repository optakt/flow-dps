package feeder

import (
	"bytes"
	"fmt"

	"github.com/awfm9/flow-dps/model"
	"github.com/onflow/flow-go/ledger/complete/wal"
	"github.com/onflow/flow-go/model/flow"
	"github.com/prometheus/client_golang/prometheus"
	pwal "github.com/prometheus/tsdb/wal"
)

type LedgerWAL struct {
	reader *pwal.Reader
}

// FromLedgerWAL creates a trie update feeder that sources state deltas
// directly from an execution node's trie directory.
func FromLedgerWAL(dir string) (*LedgerWAL, error) {

	w, err := pwal.NewSize(
		nil,
		prometheus.DefaultRegisterer,
		dir,
		32*1024*1024,
	)
	if err != nil {
		return nil, fmt.Errorf("could not initialize WAL: %w", err)
	}
	segments, err := pwal.NewSegmentsReader(w.Dir())
	if err != nil {
		return nil, fmt.Errorf("could not initialize segments reader: %w", err)
	}

	l := &LedgerWAL{
		reader: pwal.NewReader(segments),
	}

	return l, nil
}

func (l *LedgerWAL) Feed(commit flow.StateCommitment) (model.Delta, error) {
	for {
		next := l.reader.Next()
		err := l.reader.Err()
		if !next && err != nil {
			return nil, fmt.Errorf("could not read next record: %w", err)
		}
		if !next {
			return nil, model.ErrFinished
		}
		record := l.reader.Record()
		operation, _, update, err := wal.Decode(record)
		if err != nil {
			return nil, fmt.Errorf("could not decode record: %w", err)
		}
		if operation != wal.WALUpdate {
			continue
		}
		if !bytes.Equal(commit, update.RootHash) {
			return nil, fmt.Errorf("mismatch for next trie update in sequence (trie: %x, feed: %x)", commit, update.RootHash)
		}
		delta := make(model.Delta, 0, len(update.Paths))
		for index, path := range update.Paths {
			payload := update.Payloads[index]
			change := model.Change{
				Path:    path,
				Payload: *payload,
			}
			delta = append(delta, change)
		}
		return delta, nil
	}
}
