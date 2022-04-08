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

// LinearCongruentialGenerator is a pseudo random number generator that produces specifically
// the results that Flow uses to test their trie implementation against its specification.
// It uses magic numbers that are required to produce specifically the results we are testing
// against, which come from the 16-bit output used by Microsoft Visual Basic 6 and earlier.
// See https://en.wikipedia.org/wiki/Linear_congruential_generator
type LinearCongruentialGenerator struct {
	seed uint64
}

// NewGenerator generates a new linear congruential generator.
func NewGenerator() *LinearCongruentialGenerator {
	return &LinearCongruentialGenerator{}
}

// Next returns the next random number.
func (rng *LinearCongruentialGenerator) Next() uint16 {
	rng.seed = (rng.seed*1140671485 + 12820163) % 65536
	return uint16(rng.seed)
}

// SampleRandomRegisterWrites generates path-payload tuples for `count` randomly selected registers.
func SampleRandomRegisterWrites(rng *LinearCongruentialGenerator, count int) ([]ledger.Path, []ledger.Payload) {
	paths := make([]ledger.Path, 0, count)
	payloads := make([]ledger.Payload, 0, count)
	for i := 0; i < count; i++ {
		path := utils.PathByUint16LeftPadded(rng.Next())
		paths = append(paths, path)
		t := rng.Next()
		payload := utils.LightPayload(t, t)
		payloads = append(payloads, *payload)
	}

	return paths, payloads
}
