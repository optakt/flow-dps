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

package metrics

import (
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type Chain struct {
	chain dps.Chain
	time  Time
}

func NewChain(chain dps.Chain, time Time) *Chain {
	c := Chain{
		chain: chain,
		time:  time,
	}
	return &c
}

func (c *Chain) Root() (uint64, error) {
	defer c.time.Duration("root")()
	return c.chain.Root()
}

func (c *Chain) Height(blockID flow.Identifier) (uint64, error) {
	defer c.time.Duration("height")()
	return c.chain.Height(blockID)
}

func (c *Chain) Commit(height uint64) (flow.StateCommitment, error) {
	defer c.time.Duration("commit")()
	return c.chain.Commit(height)
}

func (c *Chain) Header(height uint64) (*flow.Header, error) {
	defer c.time.Duration("height")()
	return c.chain.Header(height)
}

func (c *Chain) Collections(height uint64) ([]*flow.LightCollection, error) {
	defer c.time.Duration("collections")()
	return c.chain.Collections(height)
}

func (c *Chain) Guarantees(height uint64) ([]*flow.CollectionGuarantee, error) {
	defer c.time.Duration("guarantees")()
	return c.chain.Guarantees(height)
}

func (c *Chain) Transactions(height uint64) ([]*flow.TransactionBody, error) {
	defer c.time.Duration("transactions")()
	return c.chain.Transactions(height)
}

func (c *Chain) Results(height uint64) ([]*flow.TransactionResult, error) {
	defer c.time.Duration("results")()
	return c.chain.Results(height)
}

func (c *Chain) Events(height uint64) ([]flow.Event, error) {
	defer c.time.Duration("events")()
	return c.chain.Events(height)
}

func (c *Chain) Seals(height uint64) ([]*flow.Seal, error) {
	defer c.time.Duration("seals")()
	return c.chain.Seals(height)
}
