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

package trie_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"runtime/pprof"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/common/utils"
	reference "github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/helpers"
	"github.com/optakt/flow-dps/testing/mocks"
)

// TestEmptyTrie tests whether the root hash of an empty trie matches the formal specification.
func Test_EmptyTrie(t *testing.T) {

	const expectedRootHashHex = "568f4ec740fe3b5de88034cb7b1fbddb41548b068f31aebc8ae9189e429c5749"

	store := mocks.BaselineStore()
	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	got := trie.RootHash()
	require.Equal(t, ledger.GetDefaultHashForHeight(ledger.NodeMaxHeight), hash.Hash(got))
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

// TestTrie_InsertLeftRegister tests whether the root hash of trie with only the left-most
// register populated matches the formal specification.
// The expected value is coming from a reference implementation in python and is hard-coded here.
func TestTrie_InsertLeftRegister(t *testing.T) {

	const expectedRootHashHex = "b30c99cc3e027a6ff463876c638041b1c55316ed935f1b3699e52a2c3e3eaaab"

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	path := utils.PathByUint16LeftPadded(0)
	payload := utils.LightPayload(11, 12345)

	var err error
	trie, err = trie.Insert([]ledger.Path{path}, []ledger.Payload{*payload})
	require.NoError(t, err)

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

// TestTrie_InsertRightRegister tests whether the root hash of trie with only the right-most
// register populated matches the formal specification.
// The expected value is coming from a reference implementation in python and is hard-coded here.
func TestTrie_InsertRightRegister(t *testing.T) {

	const expectedRootHashHex = "4313d22bcabbf21b1cfb833d38f1921f06a91e7198a6672bc68fa24eaaa1a961"

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	var path ledger.Path
	for i := 0; i < len(path); i++ {
		path[i] = uint8(255)
	}
	payload := utils.LightPayload(12346, 54321)

	var err error
	trie, err = trie.Insert([]ledger.Path{path}, []ledger.Payload{*payload})
	require.NoError(t, err)

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

// TestTrie_InsertMiddleRegister tests the root hash of trie holding only a single
// allocated register somewhere in the middle.
// The expected value is coming from a reference implementation in python and is hard-coded here.
func TestTrie_InsertMiddleRegister(t *testing.T) {

	const expectedRootHashHex = "4a29dad0b7ae091a1f035955e0c9aab0692b412f60ae83290b6290d4bf3eb296"

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	path := utils.PathByUint16LeftPadded(56809)
	payload := utils.LightPayload(12346, 59656)

	var err error
	trie, err = trie.Insert([]ledger.Path{path}, []ledger.Payload{*payload})
	require.NoError(t, err)

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

// TestTrie_InsertManyRegisters tests whether the root hash of a trie storing 12001 randomly selected registers
// matches the formal specification.
// The expected value is coming from a reference implementation in python and is hard-coded here.
func TestTrie_InsertManyRegisters(t *testing.T) {
	t.Skip()

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)
	refTr := reference.NewEmptyMTrie()

	paths, payloads := helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), 12001)

	for i := range paths {

		before := trie.RootHash()
		newTr, _ := trie.Insert([]ledger.Path{paths[i]}, []ledger.Payload{payloads[i]})
		after := trie.RootHash()
		require.Equal(t, before, after, "unexpected mutation of base trie")

		trie = newTr
		refTr, _ = reference.NewTrieWithUpdatedRegisters(refTr, []ledger.Path{paths[i]}, []ledger.Payload{payloads[i]})

		got := trie.RootHash()
		want := refTr.RootHash()
		require.Equal(t, want[:], got[:], "failed at iteration %d", i)
	}

	got := trie.RootHash()
	want := refTr.RootHash()
	assert.Equal(t, want[:], got[:])
}

func TestTrie_InsertNeighbors(t *testing.T) {

	store := helpers.InMemoryStore(t)
	defer store.Close()

	paths := []ledger.Path{
		utils.PathByUint16LeftPadded(0),
		utils.PathByUint16LeftPadded(1),
	}
	payloads := []ledger.Payload{
		*utils.LightPayload(11, 1111),
		*utils.LightPayload(11, 2222),
	}

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)
	ref := reference.NewEmptyMTrie()

	var err error
	trie, err = trie.Insert(paths, payloads)
	require.NoError(t, err)

	ref, err = reference.NewTrieWithUpdatedRegisters(ref, paths, payloads)
	require.NoError(t, err)

	want := ref.RootHash()
	got := trie.RootHash()
	assert.Equal(t, want[:], got[:])
}

// TestTrie_InsertFullTrie tests whether the root hash of a trie,
// whose left-most 65536 registers are populated, matches the formal specification.
// The expected value is coming from a reference implementation in python and is hard-coded here.
func TestTrie_InsertFullTrie(t *testing.T) {
	const expectedRootHashHex = "6b3a48d672744f5586c571c47eae32d7a4a3549c1d4fa51a0acfd7b720471de9"
	const regCount = 65536

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	rng := helpers.NewGenerator()
	paths := make([]ledger.Path, 0, regCount)
	payloads := make([]ledger.Payload, 0, regCount)
	for i := 0; i < regCount; i++ {
		paths = append(paths, utils.PathByUint16LeftPadded(uint16(i)))
		temp := rng.Next()
		payload := utils.LightPayload(temp, temp)
		payloads = append(payloads, *payload)
	}

	var err error
	trie, err = trie.Insert(paths, payloads)
	require.NoError(t, err)

	got := trie.RootHash()
	assert.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func TestTrie_InsertManyTimes(t *testing.T) {

	var expectedRootHashes = []string{
		"08db9aeed2b9fcc66b63204a26a4c28652e44e3035bd87ba0ed632a227b3f6dd",
		"2f4b0f490fa05e5b3bbd43176e367c3e9b64cdb710e45d4508fff11759d7a08e",
		"668811792995cd960e7e343540a360682ac375f7ec5533f774c464cd6b34adc9",
		"169c145eaeda2038a0e409068a12cb26bde5e890115ad1ef624f422007fb2d2a",
		"8f87b503a706d9eaf50873030e0e627850c841cc0cf382187b81ba26cec57588",
		"faacc057336e10e13ff6f5667aefc3ac9d9d390b34ee50391a6f7f305dfdf761",
		"049e035735a13fee09a3c36a7f567daf05baee419ac90ade538108492d80b279",
		"bb8340a9772ab6d6aa4862b23c8bb830da226cdf6f6c26f1e1e850077be600af",
		"8b9b7eb5c489bf4aeffd86d3a215dc045856094d0abe5cf7b4cc3f835d499168",
		"6514743e986f20fcf22a02e50ba352a5bfde50fe949b57b990aeb863cfcd81d1",
		"33c3d386e1c7c707f727fdeb65c52117537d175da9ab3f60a0a576301d20756e",
		"09df0bc6eee9d0f76df05d19b2ac550cde8c4294cd6eafaa1332718bd62e912f",
		"8b1fccbf7d1eca093441305ebff72d3f12b8b7cce5b4f89d6f464fc5df83b0d3",
		"0830e2d015742e284c56075050e94d3ff9618a46f28aa9066379f012e45c05fc",
		"9d95255bb75dddc317deda4e45223aa4a5ac02eaa537dc9e602d6f03fa26d626",
		"74f748dbe563bb5819d6c09a34362a048531fd9647b4b2ea0b6ff43f200198aa",
		"c06903580432a27dee461e9022a6546cb4ddec2f8598c48429e9ba7a96a892da",
		"a117f94e9cc6114e19b7639eaa630304788979cf92037736bbeb23ed1504638a",
		"d382c97020371d8788d4c27971a89f1617f9bbf21c49c922f1b683cc36a4646c",
		"ce633e9ca6329d6984c37a46e0a479bb1841674c2db00970dacfe035882d4aba",
	}

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	rng := helpers.NewGenerator()
	path := utils.PathByUint16LeftPadded(rng.Next())
	temp := rng.Next()
	payload := utils.LightPayload(temp, temp)

	var err error
	trie, err = trie.Insert([]ledger.Path{path}, []ledger.Payload{*payload})
	require.NoError(t, err)

	got := trie.RootHash()
	assert.Equal(t, expectedRootHashes[0], hex.EncodeToString(got[:]), "Before iterations")

	var paths []ledger.Path
	var payloads []ledger.Payload
	for r := 0; r < 20; r++ {
		paths, payloads = helpers.SampleRandomRegisterWrites(rng, r*100)

		trie, err = trie.Insert(paths, payloads)
		require.NoError(t, err)

		got = trie.RootHash()
		assert.Equal(t, expectedRootHashes[r], hex.EncodeToString(got[:]), "Iteration %d", r)
	}

	// update with the same registers with the same values
	// FIXME: For some reason it breaks here instead of in the initial Insert calls.
	trie, err = trie.Insert(paths, payloads)
	require.NoError(t, err)

	got = trie.RootHash()
	assert.Equal(t, expectedRootHashes[19], hex.EncodeToString(got[:]), "After iterations")
}

func TestTrie_InsertDeallocateRegisters(t *testing.T) {

	const expectedRootHashHex = "d81e27a93f2bef058395f70e00fb5d3c8e426e22b3391d048b34017e1ecb483e"

	store := helpers.InMemoryStore(t)
	defer store.Close()

	rng := helpers.NewGenerator()
	testTrie := trie.NewEmptyTrie(mocks.NoopLogger, store)
	refTrie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	var err error

	// Draw 99 random key-value pairs that will be first allocated and later deallocated.
	paths1, payloads1 := helpers.SampleRandomRegisterWrites(rng, 99)
	testTrie, err = testTrie.Insert(paths1, payloads1)
	require.NoError(t, err)

	// Write an additional 117 registers.
	paths2, payloads2 := helpers.SampleRandomRegisterWrites(rng, 117)
	testTrie, err = testTrie.Insert(paths2, payloads2)
	require.NoError(t, err)
	refTrie, err = refTrie.Insert(paths2, payloads2)
	require.NoError(t, err)

	// Now we override the first 99 registers with default values, i.e. deallocate them.
	payloads0 := make([]ledger.Payload, len(payloads1))
	testTrie, err = testTrie.Insert(paths1, payloads0)
	require.NoError(t, err)

	// this should be identical to the first 99 registers never been written
	gotRef := refTrie.RootHash()
	got := testTrie.RootHash()

	require.Equal(t, expectedRootHashHex, hex.EncodeToString(gotRef[:]))
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_UnsafeRead(t *testing.T) {
	const regCount = 65536

	store := helpers.InMemoryStore(t)
	defer store.Close()

	trie := trie.NewEmptyTrie(mocks.NoopLogger, store)

	rng := helpers.NewGenerator()
	paths := make([]ledger.Path, 0, regCount)
	payloads := make([]ledger.Payload, 0, regCount)
	for i := 0; i < regCount; i++ {
		paths = append(paths, utils.PathByUint16LeftPadded(uint16(i)))
		temp := rng.Next()
		payload := utils.LightPayload(temp, temp)
		payloads = append(payloads, *payload)
	}

	var err error
	trie, err = trie.Insert(paths, payloads)
	require.NoError(t, err)

	got := trie.UnsafeRead(paths)

	for i := range paths {
		assert.True(t, bytes.Equal(got[i].Value, payloads[i].Value))
	}

	got = trie.UnsafeRead([]ledger.Path{utils.PathByUint16(42)})

	require.Len(t, got, 1)
	assert.Nil(t, got[0])
}

// TestTrie_InsertAdvanced is a custom unit test that does not come from the
// Flow Go unit tests, which covers an edge case with extensions that the
// original tests do not: it covers the cases where an extension is cut at its
// root and needs to be transformed into a branch.
func TestTrie_InsertAdvanced(t *testing.T) {
	const totalValues = 5000

	paths := mocks.GenericLedgerPaths(totalValues)
	payloads := mocks.GenericLedgerPayloads(totalValues)

	store := helpers.InMemoryStore(t)
	defer store.Close()

	tr := trie.NewEmptyTrie(mocks.NoopLogger, store)
	refTr := reference.NewEmptyMTrie()

	var err error
	for i := range paths {
		tr, err = tr.Insert([]ledger.Path{paths[i]}, []ledger.Payload{*payloads[i]})
		require.NoError(t, err)

		refTr, err = reference.NewTrieWithUpdatedRegisters(refTr, []ledger.Path{paths[i]}, []ledger.Payload{*payloads[i]})
		require.NoError(t, err)

		want := refTr.RootHash()
		got := tr.RootHash()
		require.Equalf(t, want, got, "failed at iteration %d", i)
	}
}

func TestTrie_InsertDoesNotMutateBaseTrie(t *testing.T) {
	t.Skip()

	const totalValues = 5000

	paths := mocks.GenericLedgerPaths(totalValues)
	payloads := mocks.GenericLedgerPayloads(totalValues)

	store := helpers.InMemoryStore(t)
	defer store.Close()

	tr := trie.NewEmptyTrie(mocks.NoopLogger, store)

	for i := range paths {
		newTr, err := tr.Insert([]ledger.Path{paths[i]}, []ledger.Payload{*payloads[i]})
		require.NoError(t, err)

		require.NotEqual(t, tr.RootHash(), newTr.RootHash())
		tr = newTr
	}
}

func BenchmarkTrie_InsertMany(b *testing.B) {

	store := helpers.InMemoryStore(b)
	defer store.Close()

	paths, payloads := helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), 1000)
	ref := reference.NewEmptyMTrie()
	tr := trie.NewEmptyTrie(mocks.NoopLogger, store)

	b.Run("insert 1000 elements (reference)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ref, _ = reference.NewTrieWithUpdatedRegisters(ref, paths, payloads)
			_ = ref.RootHash()
		}
	})
	b.Run("insert 1000 elements (new)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr, _ = tr.Insert(paths, payloads)
			_ = tr.RootHash()
		}
	})
}

func BenchmarkTrie_InsertX(b *testing.B) {

	store := helpers.InMemoryStore(b)
	defer store.Close()

	for i := 1; i <= 16; i++ {
		paths, payloads := helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), i)

		b.Run(fmt.Sprintf("insert %d elements (reference)", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ref := reference.NewEmptyMTrie()
				ref, _ = reference.NewTrieWithUpdatedRegisters(ref, paths, payloads)
				_ = ref.RootHash()
			}
		})

		b.Run(fmt.Sprintf("insert %d elements (new)", i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tr := trie.NewEmptyTrie(mocks.NoopLogger, store)
				tr, _ = tr.Insert(paths, payloads)
				_ = tr.RootHash()
			}
		})
	}
}

func BenchmarkTrie_InsertNeighbors(b *testing.B) {

	paths := []ledger.Path{
		utils.PathByUint16LeftPadded(0),
		utils.PathByUint16LeftPadded(1),
	}
	payloads := []ledger.Payload{
		*utils.LightPayload(11, 1111),
		*utils.LightPayload(11, 2222),
	}

	b.Run("insert neighbors (reference)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ref := reference.NewEmptyMTrie()
			ref, _ = reference.NewTrieWithUpdatedRegisters(ref, paths, payloads)
			_ = ref.RootHash()
		}
	})

	store := helpers.InMemoryStore(b)
	defer store.Close()

	b.Run("insert neighbors (new)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr := trie.NewEmptyTrie(mocks.NoopLogger, store)
			tr, _ = tr.Insert(paths, payloads)
			_ = tr.RootHash()
		}
	})
}

func BenchmarkTrie_Read(b *testing.B) {

	store := helpers.InMemoryStore(b)
	defer store.Close()

	paths, payloads := helpers.SampleRandomRegisterWrites(helpers.NewGenerator(), 1000)

	ref, _ := reference.NewTrieWithUpdatedRegisters(reference.NewEmptyMTrie(), paths, payloads)
	tr, _ := trie.NewEmptyTrie(mocks.NoopLogger, store).Insert(paths, payloads)

	b.Run("read one element (reference)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ref.UnsafeRead(paths)
		}
	})

	b.Run("read one element (new)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr.UnsafeRead(paths)
		}
	})
}

func Test_NewMemoryUsage(t *testing.T) {
	const totalValues = 5000

	paths := mocks.GenericLedgerPaths(totalValues)
	p := mocks.GenericLedgerPayloads(totalValues)
	var payloads []ledger.Payload
	for i := range p {
		payloads = append(payloads, *p[i])
	}

	store := helpers.InMemoryStore(t)
	defer store.Close()

	tr := trie.NewEmptyTrie(mocks.NoopLogger, store)
	// FIXME: Doing a single insertion somehow prevents the insertion logic from even appearing in the heap profile.
	// tr, _ = tr.Insert(paths, payloads)
	_ = tr.RootHash()

	// FIXME: Calling insert repeatedly for each insertion makes it show in the heap profile. We don't see exactly what
	//        is being allocated, but it's not the same as the memory usage of the trie.
	for i := range paths {
		tr, _ = tr.Insert([]ledger.Path{paths[i]}, []ledger.Payload{payloads[i]})
	}

	file, _ := os.Create("/tmp/new_bench_heap.prof")
	defer file.Close()
	pprof.WriteHeapProfile(file)

	_ = tr.RootHash()

	tr.PrintSize()
}
