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
	"github.com/onflow/flow-go/fvm/state"
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
		vm.RunFunc = func(ctx fvm.Context, proc fvm.Procedure, v state.View) error {
			assert.NotNil(t, ctx)
			assert.NotNil(t, proc)
			assert.NotNil(t, v)

			require.IsType(t, proc, &fvm.ScriptProcedure{})
			p := proc.(*fvm.ScriptProcedure)
			p.Value = testValue

			return nil
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
		index.LastFunc = func() (uint64, error) {
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
		vm.RunFunc = func(ctx fvm.Context, proc fvm.Procedure, v state.View) error {
			require.IsType(t, proc, &fvm.ScriptProcedure{})
			p := proc.(*fvm.ScriptProcedure)
			p.Err = errors.NewCadenceRuntimeError(runtime.Error{})

			return nil
		}

		invoke := baselineInvoker(t)
		invoke.vm = vm

		_, err := invoke.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles proc error", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(fvm.Context, fvm.Procedure, state.View) error {
			return mocks.GenericError
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
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.StorageSnapshot) (*flow.Account, error) {
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
		index.LastFunc = func() (uint64, error) {
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
		vm.GetAccountFunc = func(fvm.Context, flow.Address, state.StorageSnapshot) (*flow.Account, error) {
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
