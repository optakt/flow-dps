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

package mocks

import (
	"context"
	"testing"

	"github.com/onflow/flow-go/consensus/hotstuff/notifications/pubsub"
	"github.com/onflow/flow-go/model/flow"
)

type ConsensusFollower struct {
	// Methods from our implementation of the consensus follower.
	OnBlockFinalizedFunc func(finalizedBlockID flow.Identifier)

	// Methods from the flow-go consensus follower.
	RunFunc                         func(context.Context)
	AddOnBlockFinalizedConsumerFunc func(pubsub.OnBlockFinalizedConsumer)
}

func BaselineConsensusFollower(t *testing.T) *ConsensusFollower {
	t.Helper()

	f := ConsensusFollower{
		OnBlockFinalizedFunc:            func(finalizedBlockID flow.Identifier) {},
		RunFunc:                         func(context.Context) {},
		AddOnBlockFinalizedConsumerFunc: func(consumer pubsub.OnBlockFinalizedConsumer) {},
	}

	return &f
}

func (f *ConsensusFollower) OnBlockFinalized(finalizedBlockID flow.Identifier) {
	f.OnBlockFinalizedFunc(finalizedBlockID)
}

func (f *ConsensusFollower) Run(ctx context.Context) {
	f.RunFunc(ctx)
}

func (f *ConsensusFollower) AddOnBlockFinalizedConsumer(consumer pubsub.OnBlockFinalizedConsumer) {
	f.AddOnBlockFinalizedConsumerFunc(consumer)
}
