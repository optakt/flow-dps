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

package failure_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/optakt/flow-dps/models/convert"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/failure"
	"github.com/optakt/flow-dps/testing/mocks"
)

func TestDescription(t *testing.T) {
	descBody := "test"
	header := mocks.GenericHeader
	index := 84
	network := dps.FlowTestnet.String()
	eventTypes := convert.TypesToStrings(mocks.GenericEventTypes(4))

	t.Run("full description with fields", func(t *testing.T) {
		t.Parallel()

		desc := failure.NewDescription(
			descBody,
			failure.WithErr(mocks.GenericError),
			failure.WithUint64("height", header.Height),
			failure.WithID("blockID", header.ID()),
			failure.WithInt("index", index),
			failure.WithString("network", network),
			failure.WithStrings("types", eventTypes...),
		)

		assert.Equal(t, desc.Text, descBody)
		assert.NotEqual(t, desc.String(), descBody)
		assert.Contains(t, desc.Fields.String(), mocks.GenericError.Error())
		assert.Contains(t, desc.Fields.String(), fmt.Sprintf("height: %v", mocks.GenericHeight))
		assert.Contains(t, desc.Fields.String(), fmt.Sprintf("blockID: %v", header.ID()))
		assert.Contains(t, desc.Fields.String(), fmt.Sprintf("index: %v", index))
		assert.Contains(t, desc.Fields.String(), fmt.Sprintf("network: %v", network))
		assert.Contains(t, desc.Fields.String(), fmt.Sprintf("types: %v", eventTypes))
	})

	t.Run("no fields", func(t *testing.T) {
		t.Parallel()

		desc := failure.NewDescription(descBody)

		assert.Equal(t, desc.Text, descBody)
		assert.Equal(t, desc.String(), descBody)
	})
}
