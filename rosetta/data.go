package rosetta

import (
	"github.com/onflow/flow-go-sdk"
)

// DataAPI
// - sanity-check identifiers against BadgerDB
// - retrieve blocks frome BadgerDB
// - retrieve events from DPS
// - retrieve balances from Invoker
type DataAPI interface {
	Block(network rosetta.NetworkIdentifier, block rosetta.BlockIdentifier) (rosetta.Block, error)
	Transaction(network rosetta.NetworkIdentifier, block rosetta.BlockIdentifier, transaction rosetta.TransactionIdentifier) (rosetta.Transaction, error)
	Balance(network rosetta.NetworkIdentifier, block rosetta.BlockIdentifier, account rosetta.AccountIdentifier) (rosetta.Balance, error)
}

type Index interface {
	Events(txID flow.Identifier) ([]flow.Event, error)
}

// Accountant
// - uses Invoker to execute scripts
type Accountant interface {
	Balance(height uint64, address flow.Address) (uint64, error)
}

// Invoker
// - depends on index for register values
