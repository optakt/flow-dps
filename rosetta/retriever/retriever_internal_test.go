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

	"github.com/optakt/flow-dps/models/dps"
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

	require.NotNil(t, r)
	assert.Equal(t, params, r.params)
	assert.Equal(t, index, r.index)
	assert.Equal(t, validate, r.validate)
	assert.Equal(t, generator, r.generator)
	assert.Equal(t, invoke, r.invoke)
	assert.Equal(t, convert, r.convert)
}
