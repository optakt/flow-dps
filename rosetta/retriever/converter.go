package retriever

import (
	"github.com/awfm9/flow-dps/models/identifier"
	"github.com/awfm9/flow-dps/models/rosetta"
)

type Converter interface {
	Balance(currency identifier.Currency, amount uint64) rosetta.Amount
}
