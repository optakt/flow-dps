package retriever

import (
	"fmt"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/awfm9/flow-dps/model/identifier"
	"github.com/awfm9/flow-dps/model/rosetta"
)

func (r *Retriever) Balances(network identifier.Network, block identifier.Block, account identifier.Account, currencies []identifier.Currency) ([]rosetta.Amount, error) {

	// get the cadence value that is the result of the script execution
	amounts := make([]rosetta.Amount, 0, len(currencies))
	address := cadence.NewAddress(flow.HexToAddress(account.Address))
	for _, currency := range currencies {
		token, ok := r.contracts.Token(currency.Symbol)
		if !ok {
			return nil, fmt.Errorf("could not find token contract (symbol: %s)", currency.Symbol)
		}
		script := r.scripts.GetBalance(token)
		value, err := r.invoke.Script(block.Index, script, []cadence.Value{address})
		if err != nil {
			return nil, fmt.Errorf("could not invoke script: %w", err)
		}
		balance, ok := value.ToGoValue().(uint64)
		if !ok {
			return nil, fmt.Errorf("could not convert balance (type: %T)", value.ToGoValue())
		}
		amount := r.convert.Balance(currency, balance)
		amounts = append(amounts, amount)
	}

	return amounts, nil
}
