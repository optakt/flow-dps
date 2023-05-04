package mapper

import (
	"math"
	"path/filepath"

	"github.com/onflow/flow-go/ledger"
)

// State is the state machine's state for the current block being processed
type State struct {
	status             Status
	height             uint64 // the height to be indexed
	updates            []*ledger.TrieUpdate
	registers          map[ledger.Path]*ledger.Payload
	checkpointDir      string // the checkpoint file for bootstrapping
	checkpointFileName string
	done               chan struct{}
}

// EmptyState returns a new empty state
func EmptyState(checkpointFile string) *State {

	dir, fileName := filepath.Split(checkpointFile)

	s := State{
		status:             StatusInitialize,
		height:             math.MaxUint64,
		registers:          make(map[ledger.Path]*ledger.Payload),
		updates:            make([]*ledger.TrieUpdate, 0),
		checkpointDir:      dir,
		checkpointFileName: fileName,
		done:               make(chan struct{}),
	}

	return &s
}
