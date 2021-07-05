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
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/cadence"
	"github.com/onflow/flow-go/fvm"
	"github.com/onflow/flow-go/fvm/errors"
	"github.com/onflow/flow-go/fvm/programs"
	"github.com/onflow/flow-go/fvm/state"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/testing/mocks"
)

func TestNew(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)

		invoker, err := New(index, WithCacheSize(uint64(datasize.MB)))

		assert.NoError(t, err)
		assert.NotNil(t, invoker)
		assert.Equal(t, index, invoker.index)
		assert.NotNil(t, invoker.cache)
		assert.NotNil(t, invoker.vm)
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
		vm.RunFunc = func(ctx fvm.Context, proc fvm.Procedure, v state.View, programs *programs.Programs) error {
			assert.NotNil(t, ctx)
			assert.NotNil(t, proc)
			assert.NotNil(t, v)
			assert.NotNil(t, programs)

			require.IsType(t, proc, &fvm.ScriptProcedure{})
			p := proc.(*fvm.ScriptProcedure)
			p.Value = testValue

			return nil
		}

		invoker := baselineInvoker(t)
		invoker.index = index
		invoker.vm = vm

		values := []cadence.Value{
			cadence.NewUInt64(1337),
		}

		val, err := invoker.Script(mocks.GenericHeight, mocks.GenericBytes, values)

		assert.NoError(t, err)
		assert.Equal(t, testValue, val)
	})

	t.Run("handles indexer failure on Header", func(t *testing.T) {
		t.Parallel()

		index := mocks.BaselineReader(t)
		index.HeaderFunc = func(uint64) (*flow.Header, error) {
			return nil, mocks.GenericError
		}

		invoker := baselineInvoker(t)
		invoker.index = index

		_, err := invoker.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles vm failure on Run", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(ctx fvm.Context, proc fvm.Procedure, v state.View, programs *programs.Programs) error {
			require.IsType(t, proc, &fvm.ScriptProcedure{})
			p := proc.(*fvm.ScriptProcedure)
			p.Err = errors.NewFVMInternalErrorf("dummy error")

			return nil
		}

		invoker := baselineInvoker(t)
		invoker.vm = vm

		_, err := invoker.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})

	t.Run("handles proc error", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.RunFunc = func(fvm.Context, fvm.Procedure, state.View, *programs.Programs) error {
			return mocks.GenericError
		}

		invoker := baselineInvoker(t)
		invoker.vm = vm

		_, err := invoker.Script(mocks.GenericHeight, mocks.GenericBytes, []cadence.Value{})

		assert.Error(t, err)
	})
}

func TestInvoker_GetAccount(t *testing.T) {
	t.Run("nominal case", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(ctx fvm.Context, address flow.Address, v state.View, programs *programs.Programs) (*flow.Account, error) {
			assert.NotNil(t, ctx)
			assert.NotNil(t, v)
			assert.NotNil(t, programs)
			assert.Equal(t, mocks.GenericAccount.Address, address)

			return &mocks.GenericAccount, nil
		}

		invoker := baselineInvoker(t)
		invoker.vm = vm

		account, err := invoker.GetAccount(mocks.GenericAccount.Address, mocks.GenericHeader)

		assert.NoError(t, err)
		assert.Equal(t, &mocks.GenericAccount, account)
	})

	t.Run("handles vm failure on GetAccount", func(t *testing.T) {
		t.Parallel()

		vm := mocks.BaselineVirtualMachine(t)
		vm.GetAccountFunc = func(fvm.Context, flow.Address, state.View, *programs.Programs) (*flow.Account, error) {
			return nil, mocks.GenericError
		}

		invoker := baselineInvoker(t)
		invoker.vm = vm

		_, err := invoker.GetAccount(mocks.GenericAccount.Address, mocks.GenericHeader)

		assert.Error(t, err)
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
