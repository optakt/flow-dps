package invoker

import (
	"context"
	"fmt"
	"github.com/rs/zerolog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	jsoncdc "github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/flow-archive/models/archive"
	"github.com/onflow/flow-archive/testing/mocks"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/errors"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/model/flow"
)

func TestNew(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		cfg := DefaultConfig
		cfg.CacheSize = 1_000_000

		invoke, err := New(
			zerolog.Nop(),
			index,
			cfg,
		)

		require.NoError(t, err)
		assert.NotNil(t, invoke)
		assert.Equal(t, index, invoke.index)
		assert.NotNil(t, invoke.cache)
	})

	t.Run("handles invalid cache configuration", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		cfg := DefaultConfig
		cfg.CacheSize = 0

		_, err := New(
			zerolog.Nop(),
			index,
			cfg,
		)

		assert.Error(t, err)
	})
}

func TestInvoker_Script(t *testing.T) {
	testValue := cadence.NewUInt64(1337)
	encodedTestValue, err := jsoncdc.Encode(testValue)
	require.NoError(t, err)

	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(
			ctx fvm.Context,
			proc fvm.Procedure,
			v snapshot.StorageSnapshot,
		) (
			*snapshot.ExecutionSnapshot,
			fvm.ProcedureOutput,
			error,
		) {
			assert.NotNil(t, ctx)
			assert.NotNil(t, proc)
			assert.NotNil(t, v)

			require.IsType(t, proc, &fvm.ScriptProcedure{})

			output := fvm.ProcedureOutput{Value: testValue}

			return &snapshot.ExecutionSnapshot{}, output, nil
		}

		config := DefaultConfig
		config.NewCustomVirtualMachine = func() fvm.VM {
			return vm
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			config,
		)
		require.NoError(t, err)

		values := [][]byte{
			jsoncdc.MustEncode(cadence.NewUInt64(1337)),
		}

		val, err := invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			values,
		)

		require.NoError(t, err)
		assert.Equal(t, encodedTestValue, val)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			nil,
		)

		assert.Error(t, err)
	})

	t.Run("handles unavailable block data", func(t *testing.T) {
		t.Parallel()
		indexedHeight := mocks.GenericHeight - 1
		index := mocks.BaselineReader(t)
		index.LatestRegisterHeightFunc = func() (uint64, error) {
			return indexedHeight, nil
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			nil)
		expectedError := fmt.Sprintf("the requested height (%d) is beyond the highest indexed height(%d)",
			mocks.GenericHeight, indexedHeight)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedError)
	})

	t.Run("handles vm failure on Run", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(
			ctx fvm.Context,
			proc fvm.Procedure,
			v snapshot.StorageSnapshot,
		) (
			*snapshot.ExecutionSnapshot,
			fvm.ProcedureOutput,
			error,
		) {
			require.IsType(t, proc, &fvm.ScriptProcedure{})

			output := fvm.ProcedureOutput{
				Err: errors.NewCadenceRuntimeError(runtime.Error{}),
			}

			return &snapshot.ExecutionSnapshot{}, output, nil
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			nil,
		)

		assert.Error(t, err)
	})

	t.Run("handles proc error", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(
			ctx fvm.Context,
			proc fvm.Procedure,
			v snapshot.StorageSnapshot,
		) (
			*snapshot.ExecutionSnapshot,
			fvm.ProcedureOutput,
			error,
		) {
			return nil, fvm.ProcedureOutput{}, mocks.GenericError
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			nil,
		)

		assert.Error(t, err)
	})
}

func TestInvoker_Account(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(
			ctx fvm.Context,
			address flow.Address,
			v snapshot.StorageSnapshot,
		) (
			*flow.Account,
			error,
		) {
			assert.NotNil(t, ctx)
			assert.NotNil(t, v)
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)

			return mocks.GenericHeader, nil
		}

		config := DefaultConfig
		config.NewCustomVirtualMachine = func() fvm.VM {
			return vm
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			config,
		)
		require.NoError(t, err)

		account, err := invoke.Account(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericAccount.Address,
		)

		require.NoError(t, err)
		assert.Equal(t, &mocks.GenericAccount, account)
	})

	t.Run("handles index failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Account(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericAccount.Address,
		)

		assert.Error(t, err)
	})

	t.Run("handles unavailable block data", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LatestRegisterHeightFunc = func() (uint64, error) {
			return mocks.GenericHeight - 1, nil
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Script(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericBytes,
			nil,
		)

		assert.Error(t, err)
	})

	t.Run("handles vm failure on Account", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(
			fvm.Context,
			flow.Address,
			snapshot.StorageSnapshot,
		) (
			*flow.Account,
			error,
		) {
			return nil, mocks.GenericError
		}

		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)

		_, err = invoke.Account(
			context.Background(),
			mocks.GenericHeight,
			mocks.GenericAccount.Address,
		)

		assert.Error(t, err)
	})
}

func TestInvoker_ByHeightFrom(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)
		res, err := invoke.ByHeightFrom(mocks.GenericHeight, mocks.GenericHeader)
		assert.NoError(t, err)
		assert.Equal(t, res, mocks.GenericHeader)
	})

	t.Run("errors on out of range", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		index.FirstFunc = func() (uint64, error) {
			return 0, nil
		}
		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)
		res, err := invoke.ByHeightFrom(mocks.GenericHeight+1, mocks.GenericHeader)
		assert.ErrorContains(t, err, "is not in the range")
		assert.Nil(t, res)
	})

	t.Run("returns requested finalized block", func(t *testing.T) {
		t.Parallel()
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		invoke, err := New(
			zerolog.Nop(),
			index,
			DefaultConfig,
		)
		require.NoError(t, err)
		testHeader := &flow.Header{
			ChainID: archive.FlowTestnet,
			Height:  mocks.GenericHeight + 4,
		}
		res, err := invoke.ByHeightFrom(mocks.GenericHeight, testHeader)
		assert.NoError(t, err)
		assert.Equal(t, res, mocks.GenericHeader)
	})
}

func TestReadRegister(t *testing.T) {
	owner := string(mocks.GenericLedgerKey.KeyParts[0].Value)
	key := string(mocks.GenericLedgerKey.KeyParts[1].Value)
	registerId := flow.RegisterID{
		Owner: owner,
		Key:   key,
	}

	t.Run("nominal case with cached register", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache contains the register's value already.
			return mocks.GenericBytes, true
		}

		var indexCalled bool
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			indexCalled = true
			return nil, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(registerId)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBytes, value[:])
		assert.False(t, indexCalled)
	})

	t.Run("nominal case without cached register", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache DOES NOT contain the register's value already.
			return nil, false
		}

		var indexCalled bool
		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			indexCalled = true
			return []flow.RegisterValue{mocks.GenericBytes}, nil
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		value, err := readFunc(registerId)

		require.NoError(t, err)
		assert.Equal(t, mocks.GenericBytes, value[:])
		assert.True(t, indexCalled)
	})

	t.Run("handles indexer failure on Values", func(t *testing.T) {
		t.Parallel()

		cache := mocks.BaselineCache(t)
		cache.GetFunc = func(key interface{}) (interface{}, bool) {
			// Return that the cache DOES NOT contain the register's value already.
			return nil, false
		}

		index := mocks.BaselineReader(t)
		index.ValuesFunc = func(uint64, flow.RegisterIDs) ([]flow.RegisterValue, error) {
			return nil, mocks.GenericError
		}

		readFunc := readRegister(index, cache, mocks.GenericHeight)
		_, err := readFunc(registerId)

		assert.Error(t, err)
	})
}
