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

//go:build integration
// +build integration

package rosetta_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/optakt/flow-dps/api/rosetta"
	"github.com/optakt/flow-dps/models/dps"
)

func TestAPI_Networks(t *testing.T) {
	// TODO: Repair integration tests
	//       See https://github.com/optakt/flow-dps/issues/333
	t.Skip("integration tests disabled until new snapshot is generated")

	db := setupDB(t)
	api := setupAPI(t, db)

	// network request is basically an empty payload at the moment,
	// there is a 'metadata' object that we're ignoring;
	// but we can have the scaffolding here in case something changes

	var netReq rosetta.NetworksRequest

	rec, ctx, err := setupRecorder(listEndpoint, netReq)
	require.NoError(t, err)

	err = api.Networks(ctx)
	assert.NoError(t, err)

	var res rosetta.NetworksResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))

	require.Len(t, res.NetworkIDs, 1)
	assert.Equal(t, res.NetworkIDs[0].Blockchain, dps.FlowBlockchain)
	assert.Equal(t, res.NetworkIDs[0].Network, dps.FlowTestnet.String())
}
