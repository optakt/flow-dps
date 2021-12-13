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

package helpers

import (
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/utils"
)

type LinearCongruentialGenerator struct {
	seed uint64
}

func NewGenerator() *LinearCongruentialGenerator {
	return &LinearCongruentialGenerator{}
}

func (rng *LinearCongruentialGenerator) Next() uint16 {
	rng.seed = (rng.seed*1140671485 + 12820163) % 65536
	return uint16(rng.seed)
}

func SampleRandomRegisterWrites(rng *LinearCongruentialGenerator, number int) ([]ledger.Path, []ledger.Payload) {
	paths := make([]ledger.Path, 0, number)
	payloads := make([]ledger.Payload, 0, number)
	for i := 0; i < number; i++ {
		path := utils.PathByUint16LeftPadded(rng.Next())
		paths = append(paths, path)
		t := rng.Next()
		payload := utils.LightPayload(t, t)
		payloads = append(payloads, *payload)
	}

	payloadMapping := make(map[ledger.Path]int)
	for i, path := range paths {
		payloadMapping[path] = i
	}

	dedupedPaths := make([]ledger.Path, 0, len(payloadMapping))
	dedupedPayloads := make([]ledger.Payload, 0, len(payloadMapping))
	for path := range payloadMapping {
		dedupedPaths = append(dedupedPaths, path)
		dedupedPayloads = append(dedupedPayloads, payloads[payloadMapping[path]])
	}

	return dedupedPaths, dedupedPayloads
}
