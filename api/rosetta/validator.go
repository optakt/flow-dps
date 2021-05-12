package rosetta

import (
	"github.com/awfm9/flow-dps/rosetta/identifier"
)

type Validator interface {
	Network(network identifier.Network) error
	Block(block identifier.Block) error
	Currency(currency identifier.Currency) error
}
