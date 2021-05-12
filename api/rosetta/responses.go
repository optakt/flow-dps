package rosetta

import (
	"github.com/awfm9/flow-dps/models/identifier"
	"github.com/awfm9/flow-dps/models/rosetta"
)

type BlockResponse struct {
	Block             rosetta.Block            `json:"block"`
	OtherTransactions []identifier.Transaction `json:"other_transactions"`
}

type TransactionResponse struct {
	Transaction rosetta.Transaction `json:"transaction"`
}

type BalanceResponse struct {
	BlockID  identifier.Block `json:"block_identifier"`
	Balances []rosetta.Amount `json:"balances"`
}
