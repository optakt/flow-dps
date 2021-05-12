package rosetta

import (
	"github.com/awfm9/flow-dps/model/identifier"
)

// Block contains an array of transactions that occurred at a particular block
// identifier. A hard requirement for blocks returned by Rosetta implementations
// is that they must be inalterable: once a client has requested and received a
// block identified by a specific block identifier, all future calls for that
// same block identifier must return the same block contents.
//
// Examples given of metadata in the Rosetta API documentation are
// `transaction_root` and `difficulty`.
type Block struct {
	ID           identifier.Block `json:"block_identifier"`
	ParentID     identifier.Block `json:"parent_block_identifier"`
	Timestamp    int64            `json:"timestamp"`
	Transactions []Transaction    `json:"transactions"`
}
