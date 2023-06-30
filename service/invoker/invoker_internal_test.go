package invoker

import (
	"fmt"
	"testing"

	"github.com/onflow/flow-archive/models/archive"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/runtime"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/errors"
	"github.com/onflow/flow-go/fvm/storage/snapshot"
	"github.com/onflow/flow-go/model/flow"

	"github.com/onflow/flow-archive/testing/mocks"
)

func TestNew(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		invoke, err := New(index, WithCacheSize(1_000_000))

		require.NoError(t, err)
		assert.NotNil(t, invoke)
		assert.Equal(t, index, invoke.index)
		assert.NotNil(t, invoke.cache)
		assert.NotNil(t, invoke.vm)
	})

	t.Run("handles invalid cache configuration", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		_, err := New(index, WithCacheSize(0))

		assert.Error(t, err)
	})
}

func TestInvoker_Script(t *testing.T) {
	testValue := cadence.NewUInt64(1337)

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

		invoke := baselineInvoker(t)
		invoke.index = index
		invoke.vm = vm

		values := []cadence.Value{
			cadence.NewUInt64(1337),
		}

		val, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, values)

		require.NoError(t, err)
		assert.Equal(t, testValue, val)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		invoke := baselineInvoker(t)
		invoke.index = index

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles unavailable block data", func(t *testing.T) {
		t.Parallel()
		indexedHeight := mocks.GenericHeight - 1
		index := mocks.BaselineReader(t)
		index.LatestRegisterHeightFunc = func() (uint64, error) {
			return indexedHeight, nil
		}

		invoke := baselineInvoker(t)
		invoke.index = index

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})
		expectedError := fmt.Sprintf("the requested height (%d) is beyond the highest indexed height(%d)",
			mocks.GenericHeight, indexedHeight)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), expectedError)
	})

	t.Run("handles vm failure on Run", func(t *testing.T) {
		t.Parallel()

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

		invoke := baselineInvoker(t)
		invoke.vm = vm

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles proc error", func(t *testing.T) {
		t.Parallel()

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

		invoke := baselineInvoker(t)
		invoke.vm = vm

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

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

		invoke := baselineInvoker(t)
		invoke.vm = vm
		invoke.index = index

		account, err := invoke.Account(mocks.GenericHeight, mocks.GenericAccount.Address)

		require.NoError(t, err)
		assert.Equal(t, &mocks.GenericAccount, account)
	})

	t.Run("handles index failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		invoke := baselineInvoker(t)
		invoke.index = index

		_, err := invoke.Account(mocks.GenericHeight, mocks.GenericAccount.Address)

		assert.Error(t, err)
	})

	t.Run("handles unavailable block data", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.LatestRegisterHeightFunc = func() (uint64, error) {
			return mocks.GenericHeight - 1, nil
		}

		invoke := baselineInvoker(t)
		invoke.index = index

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles vm failure on Account", func(t *testing.T) {
		t.Parallel()

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

		invoke := baselineInvoker(t)
		invoke.vm = vm

		_, err := invoke.Account(mocks.GenericHeight, mocks.GenericAccount.Address)

		assert.Error(t, err)
	})
}

func TestInvoker_ByHeightFrom(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()
		vm := mocks.BaselineVirtualMachine(t)
		index := mocks.BaselineReader(t)
		invoke := baselineInvoker(t)
		invoke.vm = vm
		invoke.index = index
		res, err := invoke.ByHeightFrom(mocks.GenericHeight, mocks.GenericHeader)
		assert.NoError(t, err)
		assert.Equal(t, res, mocks.GenericHeader)
	})

	t.Run("errors on out of range", func(t *testing.T) {
		t.Parallel()
		vm := mocks.BaselineVirtualMachine(t)
		index := mocks.BaselineReader(t)
		index.FirstFunc = func() (uint64, error) {
			return 0, nil
		}
		invoke := baselineInvoker(t)
		invoke.vm = vm
		invoke.index = index
		res, err := invoke.ByHeightFrom(mocks.GenericHeight+1, mocks.GenericHeader)
		assert.ErrorContains(t, err, "is not in the range")
		assert.Nil(t, res)
	})

	t.Run("returns requested finalized block", func(t *testing.T) {
		t.Parallel()
		vm := mocks.BaselineVirtualMachine(t)
		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(height uint64) (*flow.Header, error) {
			assert.Equal(t, mocks.GenericHeight, height)
			return mocks.GenericHeader, nil
		}
		invoke := baselineInvoker(t)
		invoke.vm = vm
		invoke.index = index
		testHeader := &flow.Header{
			ChainID: archive.FlowTestnet,
			Height:  mocks.GenericHeight + 4,
		}
		res, err := invoke.ByHeightFrom(mocks.GenericHeight, testHeader)
		assert.NoError(t, err)
		assert.Equal(t, res, mocks.GenericHeader)
	})
}

func baselineInvoker(t *testing.T) *Invoker {
	t.Helper()

	i := Invoker{
		index: mocks.BaselineReader(t),
		vm:    mocks.BaselineVirtualMachine(t),
		cache: mocks.BaselineCache(t),
	}

	return &i
}
