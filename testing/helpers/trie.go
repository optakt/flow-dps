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
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/utils"

	storage "github.com/optakt/flow-dps/ledger/store"
	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/testing/mocks"
)

// InMemoryStore returns a store with enough storage to handle our tests in memory,
// as well as a function to tear it down once it is no longer needed.
func InMemoryStore(t *testing.T) (store dps.Store, teardown func()) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	store, err = storage.New(mocks.NoopLogger, storage.WithCacheSize(4096), storage.WithStoragePath(dir))
	require.NoError(t, err)

	teardownFunc := func() {
		_ = store.Close()
		_ = os.RemoveAll(dir)
	}

	return store, teardownFunc
}

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
