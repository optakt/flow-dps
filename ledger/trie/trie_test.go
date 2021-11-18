package trie

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/utils"
	theirs "github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/optakt/flow-dps/testing/helpers"
)

func Test_TrieWithLeftRegister(t *testing.T) {
	trie := &Trie{root: nil}

	path := utils.PathByUint16LeftPadded(0)
	payload := utils.LightPayload(11, 12345)

	db := helpers.InMemoryDB(t)
	defer db.Close()
	trie.Insert(db, path, payload)

	expectedRootHashHex := "b30c99cc3e027a6ff463876c638041b1c55316ed935f1b3699e52a2c3e3eaaab"

	got := trie.root.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithRightRegister(t *testing.T) {
	trie := &Trie{root: nil}

	var path ledger.Path
	for i := 0; i < len(path); i++ {
		path[i] = uint8(255)
	}
	payload := utils.LightPayload(12346, 54321)

	db := helpers.InMemoryDB(t)
	defer db.Close()
	trie.Insert(db, path, payload)

	expectedRootHashHex := "4313d22bcabbf21b1cfb833d38f1921f06a91e7198a6672bc68fa24eaaa1a961"

	got := trie.root.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithMiddleRegister(t *testing.T) {
	trie := &Trie{root: nil}

	path := utils.PathByUint16LeftPadded(56809)
	payload := utils.LightPayload(12346, 59656)

	db := helpers.InMemoryDB(t)
	defer db.Close()
	trie.Insert(db, path, payload)

	expectedRootHashHex := "4a29dad0b7ae091a1f035955e0c9aab0692b412f60ae83290b6290d4bf3eb296"

	got := trie.root.Hash()
	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Test_TrieWithManyRegisters(t *testing.T) {
	trie := &Trie{root: nil}

	rng := &LinearCongruentialGenerator{seed: 0}
	paths, payloads := deduplicateWrites(sampleRandomRegisterWrites(rng, 12001))

	db := helpers.InMemoryDB(t)
	defer db.Close()
	for i := range paths {
		trie.Insert(db, paths[i], &payloads[i])
	}

	got := trie.root.ComputeHash()

	file, _ := os.Create("/tmp/new.log")
	defer file.Close()
	trie.Dump(file)

	expectedRootHashHex := "74f748dbe563bb5819d6c09a34362a048531fd9647b4b2ea0b6ff43f200198aa"

	require.Equal(t, expectedRootHashHex, hex.EncodeToString(got[:]))
}

func Benchmark_TrieRootHash(b *testing.B) {
	trie := &Trie{root: nil}
	ref := theirs.NewEmptyMTrie()

	rng := &LinearCongruentialGenerator{seed: 0}
	paths, payloads := deduplicateWrites(sampleRandomRegisterWrites(rng, 30_000_000))

	db := helpers.InMemoryDB(&testing.T{})
	defer db.Close()

	b.Run("insert element (ours)", func(b *testing.B) {
		for i := range paths {
			trie.Insert(db, paths[i], &payloads[i])
		}
		_ = trie.root.ComputeHash()
	})

	b.Run("insert element (theirs)", func(b *testing.B) {
		ref, _ = theirs.NewTrieWithUpdatedRegisters(ref, paths, payloads)
		_ = ref.RootHash()
	})
}

// deduplicateWrites retains only the last register write
func deduplicateWrites(paths []ledger.Path, payloads []ledger.Payload) ([]ledger.Path, []ledger.Payload) {
	payloadMapping := make(map[ledger.Path]int)
	if len(paths) != len(payloads) {
		panic("size mismatch (paths and payloads)")
	}
	for i, path := range paths {
		// we override the latest in the slice
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
