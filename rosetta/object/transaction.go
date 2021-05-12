package object

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
)

// Transaction contains an array of operations that are attributable to the same
// transaction identifier.
//
// Examples of metadata given in the Rosetta API documentation are "size" and
// "lockTime".
type Transaction struct {
	ID         identifier.Transaction `json:"transaction_identifier"`
	Operations []Operation            `json:"operations"`
}
