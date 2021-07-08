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
	"fmt"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type Codec struct {
	dps.Codec
	size Size
}

func NewCodec(codec dps.Codec, size Size) *Codec {
	c := Codec{
		Codec: codec,
		size:  size,
	}
	return &c
}

func (c *Codec) Marshal(value interface{}) ([]byte, error) {
	data, err := c.Encode(value)
	if err != nil {
		return nil, fmt.Errorf("could not encode value: %w", err)
	}
	compressed, err := c.Compress(data)
	if err != nil {
		return nil, fmt.Errorf("could not compress data: %w", err)
	}
	name := "unknown"
	switch value.(type) {
	case uint64:
		name = "uint64"
	case flow.StateCommitment:
		name = "commit"
	case *flow.Header:
		name = "header"
	case []flow.Event:
		name = "events"
	case *ledger.Payload:
		name = "payload"
	case *flow.TransactionBody:
		name = "transaction"
	case *flow.LightCollection:
		name = "collection"
	case []flow.Identifier:
		name = "indentifiers"
	}
	c.size.Bytes(name, len(data), len(compressed))
	return compressed, nil
}
