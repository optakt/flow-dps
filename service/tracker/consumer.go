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

package tracker

import (
	"github.com/onflow/flow-go/consensus/hotstuff/notifications/pubsub"
	"github.com/onflow/flow-go/model/flow"
)

type Consumer struct {
	exeQ chan flow.Identifier
	conQ chan flow.Identifier
}

func NewConsumer(size uint) *Consumer {

	c := Consumer{
		exeQ: make(chan flow.Identifier, size),
		conQ: make(chan flow.Identifier, size),
	}

	return &c
}

func (c *Consumer) Execution(callback pubsub.OnBlockFinalizedConsumer) {
	go func() {
		for blockID := range c.exeQ {
			callback(blockID)
		}
	}()
}

func (c *Consumer) Consensus(callback pubsub.OnBlockFinalizedConsumer) {
	go func() {
		for blockID := range c.conQ {
			callback(blockID)
		}
	}()
}

func (c *Consumer) OnBlockFinalized(blockID flow.Identifier) {
	c.exeQ <- blockID
	c.conQ <- blockID
}

func (c *Consumer) Close() {
	close(c.exeQ)
	close(c.conQ)
	for range c.exeQ {
		// drain
	}
	for range c.conQ {
		// drain
	}
}
