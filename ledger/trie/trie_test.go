package trie_test

import (
	"encoding/hex"
	"io"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/utils"
	reference "github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/optakt/flow-dps/ledger/store"

	"github.com/optakt/flow-dps/ledger/trie"
	"github.com/optakt/flow-dps/testing/mocks"
)

func Test_TrieWithLeftRegister(t *testing.T) {
	store := mocks.BaselineStore()
	trie := trie.NewEmptyTrie(store)

	path := utils.PathByUint16LeftPadded(0)
	payload := utils.LightPayload(11, 12345)

	trie.Insert(path, payload)

	expectedRootHashHex := "b30c99cc3e027a6ff463876c638041b1c55316ed935f1b3699e52a2c3e3eaaab"

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithRightRegister(t *testing.T) {
	store := mocks.BaselineStore()
	trie := trie.NewEmptyTrie(store)

	var path ledger.Path
	for i := 0; i < len(path); i++ {
		path[i] = uint8(255)
	}
	payload := utils.LightPayload(12346, 54321)

	trie.Insert(path, payload)

	expectedRootHashHex := "4313d22bcabbf21b1cfb833d38f1921f06a91e7198a6672bc68fa24eaaa1a961"

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithMiddleRegister(t *testing.T) {
	store := mocks.BaselineStore()
	trie := trie.NewEmptyTrie(store)

	path := utils.PathByUint16LeftPadded(56809)
	payload := utils.LightPayload(12346, 59656)

	trie.Insert(path, payload)

	expectedRootHashHex := "4a29dad0b7ae091a1f035955e0c9aab0692b412f60ae83290b6290d4bf3eb296"

	got := trie.RootHash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithManyRegisters(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	store, err := store.NewStore(zerolog.New(io.Discard), 4*1000*1000, dir)
	require.NoError(t, err)

	trie := trie.NewEmptyTrie(store)

	rng := &LinearCongruentialGenerator{seed: 0}
	paths, payloads := deduplicateWrites(sampleRandomRegisterWrites(rng, 12001))


	ref := reference.NewEmptyMTrie()

	for i := range paths {
		trie.Insert(paths[i], &payloads[i])
		ref, _ = reference.NewTrieWithUpdatedRegisters(ref, []ledger.Path{paths[i]}, []ledger.Payload{payloads[i]})
		//fmt.Printf("want %v got %v\n", ref.RootHash(), trie.RootHash())
		if !ref.RootHash().Equals(trie.RootHash()) {
			//fmt.Println("mismatch")
			break
		}
	}

	got := trie.RootHash()
	expectedRootHashHex := "74f748dbe563bb5819d6c09a34362a048531fd9647b4b2ea0b6ff43f200198aa"
	assert.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_FullTrie(t *testing.T) {
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	store, err := store.NewStore(zerolog.New(io.Discard), 4*1000*1000, dir)
	require.NoError(t, err)

	trie := trie.NewEmptyTrie(store)

	numberRegisters := 65536
	rng := &LinearCongruentialGenerator{seed: 0}
	paths := make([]ledger.Path, 0, numberRegisters)
	payloads := make([]*ledger.Payload, 0, numberRegisters)
	for i := 0; i < numberRegisters; i++ {
		paths = append(paths, utils.PathByUint16LeftPadded(uint16(i)))
		temp := rng.next()
		payload := utils.LightPayload(temp, temp)
		payloads = append(payloads, payload)
	}

	for i := range paths {
		trie.Insert(paths[i], payloads[i])
	}

	got := trie.RootHash()
	expectedRootHashHex := "6b3a48d672744f5586c571c47eae32d7a4a3549c1d4fa51a0acfd7b720471de9"
	assert.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Benchmark_TrieRootHash(b *testing.B) {
	store := mocks.BaselineStore()

	rng := &LinearCongruentialGenerator{seed: 0}
	paths, payloads := deduplicateWrites(sampleRandomRegisterWrites(rng, 12001))

	//var wantHash, gotHash ledger.RootHash
	b.Run("insert elements (reference)", func(b *testing.B) {
		ref := reference.NewEmptyMTrie()
		ref, _ = reference.NewTrieWithUpdatedRegisters(ref, paths, payloads)
		_ = ref.RootHash()
	})

	b.Run("insert elements (new)", func(b *testing.B) {
		trie := trie.NewEmptyTrie(store)
		for i := range paths {
			trie.Insert(paths[i], &payloads[i])
		}
		_ = trie.RootHash()
	})
}

// Below are test helper functions that are specifically needed for testing trie hashing results.

// deduplicateWrites retains only the last register write for each path. Its result is unordered.
func deduplicateWrites(paths []ledger.Path, payloads []ledger.Payload) ([]ledger.Path, []ledger.Payload) {
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

type LinearCongruentialGenerator struct {
	seed uint64
}

func (rng *LinearCongruentialGenerator) next() uint16 {
	rng.seed = (rng.seed*1140671485 + 12820163) % 65536
	return uint16(rng.seed)
}

func sampleRandomRegisterWrites(rng *LinearCongruentialGenerator, number int) ([]ledger.Path, []ledger.Payload) {
	paths := make([]ledger.Path, 0, number)
	payloads := make([]ledger.Payload, 0, number)
	for i := 0; i < number; i++ {
		path := utils.PathByUint16LeftPadded(rng.next())
		paths = append(paths, path)
		t := rng.next()
		payload := utils.LightPayload(t, t)
		payloads = append(payloads, *payload)
	}
	return paths, payloads
}
