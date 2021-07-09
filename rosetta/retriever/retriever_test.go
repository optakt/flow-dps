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

package retriever

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNew(t *testing.T) {
	params := dps.Params{ChainID: dps.FlowTestnet}
	index := mocks.BaselineReader(t)
	validate := mocks.BaselineValidator(t)
	generator := mocks.BaselineGenerator(t)
	invoke := mocks.BaselineInvoker(t)
	convert := mocks.BaselineConverter(t)

	r := New(params, index, validate, generator, invoke, convert)

	if assert.NotNil(t, r) {
		assert.Equal(t, params, r.params)
		assert.Equal(t, index, r.index)
		assert.Equal(t, validate, r.validate)
		assert.Equal(t, generator, r.generator)
		assert.Equal(t, invoke, r.invoke)
		assert.Equal(t, convert, r.convert)
	}
}

func TestRetriever_Oldest(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		blockID, blockTime, err := ret.Oldest()

		if assert.NoError(t, err) {
			wantBlockID := identifier.Block{
				Index: &mocks.GenericHeader.Height,
				Hash:  mocks.GenericHeader.ID().String(),
			}
			assert.Equal(t, wantBlockID, blockID)
			assert.Equal(t, mocks.GenericHeader.Timestamp, blockTime)
		}
	})

	t.Run("handles index.First failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.FirstFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Oldest()

		assert.Error(t, err)
	})

	t.Run("handles index.Header failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Oldest()

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

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		blockID, blockTime, err := ret.Current()

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericHeader.ID().String(), blockID.Hash)
			assert.Equal(t, mocks.GenericHeader.Timestamp, blockTime)
		}
	})

	t.Run("handles index.Last failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LastFunc = func() (uint64, error) {
			return 0, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Current()

		assert.Error(t, err)
	})

	t.Run("handles index.Header failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Current()

		assert.Error(t, err)
	})
}

func TestRetriever_Balances(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(address identifier.Account) error {
			assert.Equal(t, mocks.GenericAccountID(0), address)
			return nil
		}
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return block, nil
		}
		validator.CurrencyFunc = func(currency identifier.Currency) (identifier.Currency, error) {
			assert.Equal(t, mocks.GenericCurrency, currency)
			return currency, nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.GetBalanceFunc = func(symbol string) ([]byte, error) {
			assert.Equal(t, mocks.GenericCurrency.Symbol, symbol)
			return []byte(`test`), nil
		}

		invoker := mocks.BaselineInvoker(t)
		invoker.ScriptFunc = func(height uint64, script []byte, parameters []cadence.Value) (cadence.Value, error) {
			assert.Equal(t, mocks.GenericBlockID.Index, &height)
			assert.Equal(t, []byte(`test`), script)
			if assert.Len(t, parameters, 1) {
				assert.Equal(t, cadence.NewAddress(mocks.GenericAccount.Address), parameters[0])
			}
			return mocks.GenericAmount(0), nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator
		ret.generator = generator
		ret.invoke = invoker

		blockID, amounts, err := ret.Balances(
			mocks.GenericBlockID,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericBlockID, blockID)

			wantAmounts := []object.Amount{
				mocks.GenericOperation(0).Amount,
			}
			assert.Equal(t, wantAmounts, amounts)
		}
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return identifier.Block{}, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, _, err = ret.Balances(
			mocks.GenericBlockID,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles invalid account", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.AccountFunc = func(address identifier.Account) error {
			return mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, _, err = ret.Balances(
			mocks.GenericBlockID,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})

	t.Run("handles invalid currency", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.CurrencyFunc = func(currency identifier.Currency) (identifier.Currency, error) {
			return currency, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, _, err = ret.Balances(
			mocks.GenericBlockID,
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

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.generator = generator

		_, _, err = ret.Balances(
			mocks.GenericBlockID,
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

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.invoke = invoker

		_, _, err = ret.Balances(
			mocks.GenericBlockID,
			mocks.GenericAccountID(0),
			[]identifier.Currency{mocks.GenericCurrency},
		)
		assert.Error(t, err)
	})
}

func TestRetriever_Block(t *testing.T) {
	t.Run("nominal case without limit", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			if assert.Len(t, types, 2) {
				assert.Equal(t, mocks.GenericEventType(0), types[0])
				assert.Equal(t, mocks.GenericEventType(1), types[1])
			}
			return mocks.GenericEvents(4), nil
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return block, nil
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
			assert.Contains(t, mocks.GenericEvents(4), event)

			var op object.Operation
			switch event.Type {
			case mocks.GenericEventType(0):
				op = mocks.GenericOperation(0)
			case mocks.GenericEventType(1):
				op = mocks.GenericOperation(1)
			}

			op.RelatedIDs = nil // unset RelatedIDs to prevent having duplicate related IDs.
			return &op, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index
		ret.validate = validator
		ret.generator = generator
		ret.convert = convert

		block, extra, err := ret.Block(mocks.GenericBlockID)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericBlockID, block.ID)
			assert.Len(t, block.Transactions, 2)

			assert.Empty(t, extra)
		}
	})

	t.Run("nominal case with limit reached exactly", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			if assert.Len(t, types, 2) {
				assert.Equal(t, mocks.GenericEventType(0), types[0])
				assert.Equal(t, mocks.GenericEventType(1), types[1])
			}
			return mocks.GenericEvents(4), nil
		}

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return block, nil
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
			assert.Contains(t, mocks.GenericEvents(4), event)

			var op object.Operation
			switch event.Type {
			case mocks.GenericEventType(0):
				op = mocks.GenericOperation(0)
			case mocks.GenericEventType(1):
				op = mocks.GenericOperation(1)
			}

			op.RelatedIDs = nil // unset RelatedIDs to prevent having duplicate related IDs.
			return &op, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index
		ret.validate = validator
		ret.generator = generator
		ret.convert = convert
		ret.cfg.TransactionLimit = 2

		block, extra, err := ret.Block(mocks.GenericBlockID)

		if assert.NoError(t, err) {
			assert.Len(t, block.Transactions, 2)
			assert.Equal(t, mocks.GenericBlockID, block.ID)

			assert.Empty(t, extra)
		}
	})

	t.Run("nominal case with more transactions than limit", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			if assert.Len(t, types, 2) {
				assert.Equal(t, mocks.GenericEventType(0), types[0])
				assert.Equal(t, mocks.GenericEventType(1), types[1])
			}
			return mocks.GenericEvents(4), nil
		}
		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return block, nil
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
			assert.Contains(t, mocks.GenericEvents(4), event)

			var op object.Operation
			switch event.Type {
			case mocks.GenericEventType(0):
				op = mocks.GenericOperation(0)
			case mocks.GenericEventType(1):
				op = mocks.GenericOperation(1)
			}

			op.RelatedIDs = nil // unset RelatedIDs to prevent having duplicate related IDs.
			return &op, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index
		ret.validate = validator
		ret.generator = generator
		ret.convert = convert
		ret.cfg.TransactionLimit = 1

		block, extra, err := ret.Block(mocks.GenericBlockID)

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericBlockID, block.ID)
			assert.Len(t, block.Transactions, 1)

			assert.Len(t, extra, 1)
		}
	})

	t.Run("handles block without transactions", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return []flow.Event{}, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		got, _, err := ret.Block(mocks.GenericBlockID)
		if assert.NoError(t, err) {
			assert.Empty(t, got.Transactions)
		}
	})

	t.Run("handles block without relevant events", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return []flow.Event{}, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		got, _, err := ret.Block(mocks.GenericBlockID)

		if assert.NoError(t, err) {
			assert.Empty(t, got.Transactions)
		}
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			return identifier.Block{}, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})

	t.Run("handles deposit script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.generator = generator

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})

	t.Run("handles withdrawal script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.generator = generator

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})

	t.Run("handles index header retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})

	t.Run("handles index event retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			if assert.Len(t, types, 2) {
				assert.Equal(t, mocks.GenericEventType(0), types[0])
				assert.Equal(t, mocks.GenericEventType(1), types[1])
			}
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})

	t.Run("handles event converter failure", func(t *testing.T) {
		t.Parallel()

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.convert = convert

		_, _, err = ret.Block(mocks.GenericBlockID)
		assert.Error(t, err)
	})
}

func TestRetriever_Transaction(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			assert.Equal(t, mocks.GenericBlockID, block)
			return block, nil
		}
		validator.TransactionFunc = func(transaction identifier.Transaction) error {
			assert.Equal(t, mocks.GenericTransactionQualifier(0), transaction)
			return nil
		}

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			assert.Equal(t, dps.FlowSymbol, symbol)
			return string(mocks.GenericEventType(0)), nil
		}
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			assert.Equal(t, dps.FlowSymbol, symbol)
			return string(mocks.GenericEventType(1)), nil
		}

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			if assert.Len(t, types, 2) {
				assert.Equal(t, mocks.GenericEventType(0), types[0])
				assert.Equal(t, mocks.GenericEventType(1), types[1])
			}

			return mocks.GenericEvents(4), nil
		}
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericIdentifiers(5), nil
		}

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
			assert.Contains(t, mocks.GenericEvents(4), event)

			var op object.Operation
			switch event.Type {
			case mocks.GenericEventType(0):
				op = mocks.GenericOperation(0)
			case mocks.GenericEventType(1):
				op = mocks.GenericOperation(1)
			}

			op.RelatedIDs = nil // unset RelatedIDs to prevent having duplicate related IDs.
			return &op, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator
		ret.generator = generator
		ret.index = index
		ret.convert = convert

		got, err := ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))

		if assert.NoError(t, err) {
			assert.Equal(t, mocks.GenericTransactionQualifier(0), got.ID)
			assert.Len(t, got.Operations, 2)
		}
	})

	t.Run("handles transaction with no relevant operations", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return []flow.Event{
				{
					Type:          mocks.GenericEventType(0),
					TransactionID: mocks.GenericIdentifier(4),
				},
			}, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		got, err := ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))

		if assert.NoError(t, err) {
			assert.Empty(t, got.Operations)
		}
	})

	t.Run("handles invalid block", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.BlockFunc = func(block identifier.Block) (identifier.Block, error) {
			return identifier.Block{}, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles invalid transaction", func(t *testing.T) {
		t.Parallel()

		validator := mocks.BaselineValidator(t)
		validator.TransactionFunc = func(transaction identifier.Transaction) error {
			return mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.validate = validator

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("block does not contain transaction", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return []flow.Identifier{}, nil
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles transactions index failure", func(t *testing.T) {
		index := mocks.BaselineReader(t)
		index.TransactionsByHeightFunc = func(height uint64) ([]flow.Identifier, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles deposit script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensDepositedFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.generator = generator

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles withdrawal script generator failure", func(t *testing.T) {
		t.Parallel()

		generator := mocks.BaselineGenerator(t)
		generator.TokensWithdrawnFunc = func(symbol string) (string, error) {
			return "", mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.generator = generator

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles index event retrieval failure", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.EventsFunc = func(height uint64, types ...flow.EventType) ([]flow.Event, error) {
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.index = index

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))
		assert.Error(t, err)
	})

	t.Run("handles converter failure", func(t *testing.T) {
		t.Parallel()

		convert := mocks.BaselineConverter(t)
		convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
			return nil, mocks.GenericError
		}

		ret, err := baselineRetriever(t)
		require.NoError(t, err)

		ret.convert = convert

		_, err = ret.Transaction(mocks.GenericBlockID, mocks.GenericTransactionQualifier(0))

		assert.Error(t, err)
	})
}

func baselineRetriever(t *testing.T) (*Retriever, error) {
	t.Helper()

	validator := mocks.BaselineValidator(t)
	generator := mocks.BaselineGenerator(t)
	index := mocks.BaselineReader(t)
	invoker := mocks.BaselineInvoker(t)

	convert := mocks.BaselineConverter(t)
	convert.EventToOperationFunc = func(event flow.Event) (*object.Operation, error) {
		var op object.Operation
		switch event.Type {
		case mocks.GenericEventType(0):
			op = mocks.GenericOperation(0)
		case mocks.GenericEventType(1):
			op = mocks.GenericOperation(1)
		}

		op.RelatedIDs = nil // unset RelatedIDs to prevent having duplicate related IDs.
		return &op, nil
	}

	retriever := Retriever{
		cfg:       Config{TransactionLimit: 999},
		params:    dps.Params{ChainID: dps.FlowTestnet},
		index:     index,
		validate:  validator,
		generator: generator,
		invoke:    invoker,
		convert:   convert,
	}

	return &retriever, nil
}

func getUint64P(n uint64) *uint64 {
	return &n
}
