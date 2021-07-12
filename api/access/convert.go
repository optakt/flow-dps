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

package access

import (
	"github.com/golang/protobuf/ptypes"

	"github.com/onflow/flow-go/engine/common/rpc/convert"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow/protobuf/go/flow/access"
	"github.com/onflow/flow/protobuf/go/flow/entities"
)

// This code has been imported from github.com/onflow/flow/protobuf/go/flow@v0.2.0/access/access.pb.go
func blockEventsToMessages(blocks []flow.BlockEvents) ([]*access.EventsResponse_Result, error) {
	results := make([]*access.EventsResponse_Result, len(blocks))

	for i, block := range blocks {
		event, err := blockEventsToMessage(block)
		if err != nil {
			return nil, err
		}
		results[i] = event
	}

	return results, nil
}

// This code has been imported from github.com/onflow/flow/protobuf/go/flow@v0.2.0/access/access.pb.go
func blockEventsToMessage(block flow.BlockEvents) (*access.EventsResponse_Result, error) {
	eventMessages := make([]*entities.Event, len(block.Events))
	for i, event := range block.Events {
		eventMessages[i] = convert.EventToMessage(event)
	}
	timestamp, err := ptypes.TimestampProto(block.BlockTimestamp)
	if err != nil {
		return nil, err
	}

	return &access.EventsResponse_Result{
		BlockId:        block.BlockID[:],
		BlockHeight:    block.BlockHeight,
		BlockTimestamp: timestamp,
		Events:         eventMessages,
	}, nil
}
