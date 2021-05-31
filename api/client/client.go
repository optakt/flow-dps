// Copyright 2021 Alvalor S.A.
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

package client

import (
	"github.com/onflow/cadence"

	"github.com/optakt/flow-dps/api/server"
	"github.com/optakt/flow-dps/rosetta/invoker"
	"github.com/optakt/flow-dps/rosetta/lookup"
	"github.com/optakt/flow-dps/rosetta/read"
	"github.com/optakt/flow-dps/rosetta/retriever"
)

type Client struct {
	invoke retriever.Invoker
}

func NewClient(client server.APIClient) *Client {

	c := Client{
		invoke: invoker.New(
			lookup.FromDPS(client),
			read.FromDPS(client),
		),
	}

	return &c
}

func (c *Client) ExecuteScript(height uint64, script []byte, arguments []cadence.Value) (cadence.Value, error) {
	return c.invoke.Script(height, script, arguments)
}
