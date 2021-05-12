package retriever

import (
	"github.com/awfm9/flow-dps/model/identifier"
	"github.com/awfm9/flow-dps/model/rosetta"
)

type Converter interface {
	Balance(currency identifier.Currency, amount uint64) rosetta.Amount
}
