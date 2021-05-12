package rosetta

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
	"github.com/awfm9/flow-dps/rosetta/object"
)

type BlockResponse struct {
	Block             object.Block             `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions"`
}

type TransactionResponse struct {
	Transaction object.Transaction `json:"transaction"`
}

type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []object.Amount  `json:"balances"`
}
