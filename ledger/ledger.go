package ledger

import (
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/pathfinder"
)

type Ledger struct {
	core    *Core
	version uint8
}

func (l *Ledger) WithVersion(version uint8) *Ledger {
	l.version = version
	return l
}

func (l *Ledger) Get(query *ledger.Query) ([]ledger.Value, error) {

	// convert the query state commitment to a height, so we can use the core
	// API to retrieve the payloads
	commit := query.State()
	height, err := l.core.Height(commit)
	if err != nil {
		return nil, fmt.Errorf("could not get height for commit (%w)", err)
	}

	// this code replicates the original ledger code to convert keys to paths
	// on input and payloads to values on output; the relevant difference is
	// that we use the core API to retrieve payloads one by one by path, which
	// will use the underlying index database that allows accessing payloads
	// at any block height
	paths, err := pathfinder.KeysToPaths(query.Keys(), l.version)
	if err != nil {
		return nil, fmt.Errorf("could not convert keys to paths (%w)", err)
	}
	payloads := make([]*ledger.Payload, 0, len(paths))
	for _, path := range paths {
		payload, err := l.core.Payload(height, path)
		if err != nil {
			return nil, fmt.Errorf("could not get payload (%w)", err)
		}
		payloads = append(payloads, payload)
	}
	values, err := pathfinder.PayloadsToValues(payloads)
	if err != nil {
		return nil, fmt.Errorf("could not convert payloads to values (%w)", err)
	}

	return values, nil
}
