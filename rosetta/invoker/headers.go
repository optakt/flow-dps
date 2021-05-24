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

package invoker

import (
	"fmt"

	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
)

type Headers struct {
	chain dps.Chain
}

func NewHeaders(chain dps.Chain) *Headers {

	h := Headers{
		chain: chain,
	}

	return &h
}

func (h *Headers) ByHeightFrom(height uint64, _ *flow.Header) (*flow.Header, error) {
	header, err := h.chain.Header(height)
	if err != nil {
		return nil, fmt.Errorf("could not get header from chain: %w", err)
	}
	return header, nil
}
