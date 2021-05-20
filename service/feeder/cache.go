package feeder

import (
	"github.com/gammazero/deque"

	"github.com/onflow/flow-go/model/flow"
)

type Cache struct {
	commit flow.StateCommitment
	forks  *deque.Deque
	expiry uint64
}
