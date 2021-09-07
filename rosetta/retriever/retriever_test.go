// Copyright 2021 Optakt Labs OÜ
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations under
// the License.

package retriever_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/rosetta/retriever"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestRetriever_Oldest(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		blockID, blockTime, err := ret.Oldest()

		require.NoError(t, err)
		wantBlockID := identifier.Block{
			Index: &mocks.GenericHeader.Height,
			Hash:  mocks.GenericHeader.ID().String(),
		}
		assert.Equal(t, wantBlockID, blockID)
		assert.Equal(t, mocks.GenericHeader.Timestamp, blockTime)
	})

	t.Run("handles index.First failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.FirstFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Oldest()

		assert.Error(t, err)
	})

	t.Run("handles index.Header failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Oldest()

		assert.Error(t, err)
	})
}

func TestRetriever_Current(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		blockID, blockTime, err := ret.Current()

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericHeader.ID().String(), blockID.Hash)
		assert.Equal(t, mocks.GenericHeader.Timestamp, blockTime)
	})

	t.Run("handles index.Last failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Current()

		assert.Error(t, err)
	})

	t.Run("handles index.Header failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Current()

		assert.Error(t, err)
	})
}

func TestRetriever_Balances(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(address identifier.Account) (flow.Address, error) {
			assert.Equal(t, mocks.GenericAccountID(0), address)
			return mocks.GenericAccount.Address, nil
		}
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}
		validator.CurrencyFunc = func(currency identifier.Currency) (string, uint, error) {
			assert.Equal(t, mocks.GenericCurrency, currency)
			return mocks.GenericCurrency.Symbol, mocks.GenericCurrency.Decimals, nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.GetBalanceFunc = func(symbol string) ([]byte, error) {
			assert.Equal(t, mocks.GenericCurrency.Symbol, symbol)
			return []byte(`test`), nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			assert.Equal(t, mocks.GenericBlockQualifier.Index, &height)
			assert.Equal(t, []byte(`test`), script)
			require.Len(t, parameters, 1)
			assert.Equal(t, cadence.NewAddress(mocks.GenericAccount.Address), parameters[0])
			return mocks.GenericAmount(0), nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			generator,
			invoker,
			mocks.BaselineConverter(t),
		)

		blockID, amounts, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBlockQualifier, blockID)

		wantAmounts := []object.Amount{
			mocks.GenericOperation(0).Amount,
		}
		assert.Equal(t, wantAmounts, amounts)
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return 0, flow.ZeroID, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles invalid account", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(account identifier.Account) (flow.Address, error) {
			return flow.EmptyAddress, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles invalid currency", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.CurrencyFunc = func(currency identifier.Currency) (string, uint, error) {
			return "", 0, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.GetBalanceFunc = func(symbol string) ([]byte, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			generator,
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles invoker failure", func(t *testing.T) {
		t.Parallel()

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			invoker,
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Balances(
			mocks.GenericBlockQualifier,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})
}

func TestRetriever_Block(t *testing.T) {
	transactions := mocks.GenericTransactionIDs(5)
	withdrawalType := mocks.GenericEventType(0)
	depositType := mocks.GenericEventType(1)
	withdrawals := mocks.GenericEvents(2, withdrawalType)
	deposits := mocks.GenericEvents(2, depositType)
	events := append(withdrawals, deposits...)

	t.Run("nominal case with limit not reached", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return transactions, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return events, nil
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(withdrawalType), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(depositType), nil
		}

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
			var op object.Operation
			switch event.Type {
			case withdrawalType:
				assert.Contains(t, withdrawals, event)
				op = mocks.GenericOperation(0)
			case depositType:
				assert.Contains(t, deposits, event)
				op = mocks.GenericOperation(1)
			}

			return &op, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			validator,
			generator,
			mocks.BaselineInvoker(t),
			convert,
			retriever.WithTransactionLimit(6),
		)

		block, extra, err := ret.Block(mocks.GenericBlockQualifier)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBlockQualifier, block.ID)
		assert.Len(t, block.Transactions, 5)

		assert.Empty(t, extra)
	})

	t.Run("nominal case with limit reached exactly", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericTransactionIDs(5), nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return events, nil
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(0)), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(1)), nil
		}

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {

			var op object.Operation
			switch event.Type {
			case withdrawalType:
				assert.Contains(t, withdrawals, event)
				op = mocks.GenericOperation(0)
			case depositType:
				assert.Contains(t, deposits, event)
				op = mocks.GenericOperation(1)
			}

			return &op, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			validator,
			generator,
			mocks.BaselineInvoker(t),
			convert,
			retriever.WithTransactionLimit(5),
		)

		block, extra, err := ret.Block(mocks.GenericBlockQualifier)

		require.NoError(t, err)
		assert.Len(t, block.Transactions, 5)
		assert.Equal(t, mocks.GenericBlockQualifier, block.ID)

		assert.Empty(t, extra)
	})

	t.Run("nominal case with more transactions than limit", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericTransactionIDs(6), nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return events, nil
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(0)), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(1)), nil
		}

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {

			var op object.Operation
			switch event.Type {
			case withdrawalType:
				assert.Contains(t, withdrawals, event)
				op = mocks.GenericOperation(0)
			case depositType:
				assert.Contains(t, deposits, event)
				op = mocks.GenericOperation(1)
			}

			return &op, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			validator,
			generator,
			mocks.BaselineInvoker(t),
			convert,
			retriever.WithTransactionLimit(5),
		)

		block, extra, err := ret.Block(mocks.GenericBlockQualifier)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBlockQualifier, block.ID)
		assert.Len(t, block.Transactions, 5)

		assert.Len(t, extra, 1)
	})

	t.Run("handles block without transactions", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			return []flow.Identifier{}, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		got, _, err := ret.Block(mocks.GenericBlockQualifier)
		require.NoError(t, err)
		assert.Empty(t, got.Transactions)
	})

	t.Run("handles block without relevant events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return []flow.Event{}, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		got, _, err := ret.Block(mocks.GenericBlockQualifier)
		require.NoError(t, err)
		require.NotEmpty(t, got.Transactions)
		for _, tx := range got.Transactions {
			assert.Empty(t, tx.Operations)
		}
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(identifier.Block) (uint64, flow.Identifier, error) {
			return 0, flow.ZeroID, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)

		assert.Error(t, err)
	})

	t.Run("handles deposit script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			generator,
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)

		assert.Error(t, err)
	})

	t.Run("handles withdrawal script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			generator,
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)

		assert.Error(t, err)
	})

	t.Run("handles index header retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)

		assert.Error(t, err)
	})

	t.Run("handles index event retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			require.Len(t, types, 2)
			assert.Equal(t, mocks.GenericEventType(0), types[0])
			assert.Equal(t, mocks.GenericEventType(1), types[1])
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)
		assert.Error(t, err)
	})

	t.Run("handles event converter failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericTransactionIDs(6), nil
		}
		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}
		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(0)), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, symbol, dps.FlowSymbol)
			return string(mocks.GenericEventType(1)), nil
		}
		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(flow.Event) (*object.Operation, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			validator,
			generator,
			mocks.BaselineInvoker(t),
			convert,
		)

		_, _, err := ret.Block(mocks.GenericBlockQualifier)
		assert.Error(t, err)
	})
}

func TestRetriever_Transaction(t *testing.T) {
	withdrawalType := mocks.GenericEventType(0)
	depositType := mocks.GenericEventType(1)
	withdrawals := mocks.GenericEvents(2, withdrawalType)
	deposits := mocks.GenericEvents(2, depositType)
	events := append(withdrawals, deposits...)

	txQual := mocks.GenericTransactionQualifier(0)
	txIDs := mocks.GenericTransactionIDs(5)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(rosBlockID identifier.Block) (uint64, flow.Identifier, error) {
			assert.Equal(t, mocks.GenericBlockQualifier, rosBlockID)
			return mocks.GenericHeader.Height, mocks.GenericHeader.ID(), nil
		}
		validator.TransactionFunc = func(transaction identifier.Transaction) (flow.Identifier, error) {
			assert.Equal(t, txQual, transaction)
			return txIDs[0], nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, dps.FlowSymbol, symbol)
			return string(withdrawalType), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, dps.FlowSymbol, symbol)
			return string(depositType), nil
		}

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			require.Len(t, types, 2)
			assert.Equal(t, withdrawalType, types[0])
			assert.Equal(t, depositType, types[1])

			return events, nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return txIDs, nil
		}

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {

			var op object.Operation
			switch event.Type {
			case withdrawalType:
				assert.Contains(t, withdrawals, event)
				op = mocks.GenericOperation(0)
			case depositType:
				assert.Contains(t, deposits, event)
				op = mocks.GenericOperation(1)
			}

			return &op, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			validator,
			generator,
			mocks.BaselineInvoker(t),
			convert,
		)

		got, err := ret.Transaction(mocks.GenericBlockQualifier, txQual)

		require.NoError(t, err)
		assert.Equal(t, txQual, got.ID)
		assert.Len(t, got.Operations, 2)
	})

	t.Run("handles transaction with no relevant operations", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return []flow.Event{
				{
					Type: mocks.GenericEventType(0),
					// Here we use the wrong resource ID on purpose so that it does not match any of transaction ID.
					TransactionID: mocks.GenericSeal(0).ID(),
				},
			}, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		got, err := ret.Transaction(mocks.GenericBlockQualifier, txQual)

		require.NoError(t, err)
		assert.Empty(t, got.Operations)
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(identifier.Block) (uint64, flow.Identifier, error) {
			return 0, flow.ZeroID, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles invalid transaction", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.TransactionFunc = func(transaction identifier.Transaction) (flow.Identifier, error) {
			return flow.ZeroID, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			validator,
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("block does not contain transaction", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return []flow.Identifier{}, nil
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles transactions index failure", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles deposit script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			generator,
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles withdrawal script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			generator,
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles index event retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			index,
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			mocks.BaselineConverter(t),
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})

	t.Run("handles converter failure", func(t *testing.T) {
		t.Parallel()

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
			return nil, mocks.GenericError
		}

		ret := retriever.New(
			mocks.GenericParams,
			mocks.BaselineReader(t),
			mocks.BaselineValidator(t),
			mocks.BaselineGenerator(t),
			mocks.BaselineInvoker(t),
			convert,
		)

		_, err := ret.Transaction(mocks.GenericBlockQualifier, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})
}
