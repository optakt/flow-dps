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
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"io"
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/onflow/flow-go/crypto"
	chash "github.com/onflow/flow-go/crypto/hash"
	"github.com/onflow/flow-go/engine/execution/computation/computer/uploader"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/model/flow"
	"github.com/onflow/flow-go/module/mempool/entity"

	"github.com/onflow/flow-dps/models/dps"
)

// Offsets used to ensure different flow identifiers that do not overlap.
// Each resource type has a range of 16 unique identifiers at the moment.
// We can increase this range if we need to, in the future.
const (
	offsetBlock      = 0
	offsetCollection = 1 * 16
	offsetResult     = 2 * 16
)

// Global variables that can be used for testing. They are non-nil valid values for the types commonly needed
// test DPS components.
var (
	NoopLogger = zerolog.New(io.Discard)

	GenericError = errors.New("dummy error")

	GenericHeight = uint64(42)

	GenericBytes = []byte(`test`)

	GenericHeader = &flow.Header{
		ChainID:   dps.FlowTestnet,
		Height:    GenericHeight,
		ParentID:  genericIdentifier(0, offsetBlock),
		Timestamp: time.Date(1972, 11, 12, 13, 14, 15, 16, time.UTC),
	}

	GenericLedgerKey = ledger.NewKey([]ledger.KeyPart{
		ledger.NewKeyPart(0, []byte(`owner`)),
		ledger.NewKeyPart(1, []byte(`controller`)),
		ledger.NewKeyPart(2, []byte(`key`)),
	})

	// GenericRootNode Visual Representation:
	//           6 (root)
	//          / \
	//         3   5
	//        / \   \
	//       1   2   4
	GenericRootNode = node.NewNode(
		256,
		node.NewNode(
			256,
			node.NewLeaf(GenericLedgerPath(0), GenericLedgerPayload(0), 42),
			node.NewLeaf(GenericLedgerPath(1), GenericLedgerPayload(1), 42),
			GenericLedgerPath(2),
			GenericLedgerPayload(2),
			hash.DummyHash,
		),
		node.NewNode(
			256,
			node.NewLeaf(GenericLedgerPath(3), GenericLedgerPayload(3), 42),
			nil,
			GenericLedgerPath(4),
			GenericLedgerPayload(4),
			hash.DummyHash,
		),
		GenericLedgerPath(5),
		GenericLedgerPayload(5),
		hash.DummyHash,
	)

	GenericTrie, _ = trie.NewMTrie(GenericRootNode, 3, 3*32)

	GenericAccount = flow.Account{
		Address: GenericAddress(0),
		Balance: 84,
		Keys: []flow.AccountPublicKey{
			{
				Index:     0,
				SeqNumber: 42,
				HashAlgo:  chash.SHA2_256,
				PublicKey: crypto.NeutralBLSPublicKey(),
			},
		},
	}
)

func GenericBlockIDs(number int) []flow.Identifier {
	return genericIdentifiers(number, offsetBlock)
}

func GenericCommits(number int) []flow.StateCommitment {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(0))

	var commits []flow.StateCommitment
	for i := 0; i < number; i++ {
		var c flow.StateCommitment
		binary.BigEndian.PutUint64(c[0:], random.Uint64())
		binary.BigEndian.PutUint64(c[8:], random.Uint64())
		binary.BigEndian.PutUint64(c[16:], random.Uint64())
		binary.BigEndian.PutUint64(c[24:], random.Uint64())

		commits = append(commits, c)
	}

	return commits
}

func GenericCommit(index int) flow.StateCommitment {
	return GenericCommits(index + 1)[index]
}

func GenericTrieUpdates(number int) []*ledger.TrieUpdate {
	// Ensure consistent deterministic results.
	seed := rand.NewSource(1)
	random := rand.New(seed)

	var updates []*ledger.TrieUpdate
	for i := 0; i < number; i++ {
		update := ledger.TrieUpdate{
			Paths:    GenericLedgerPaths(6),
			Payloads: GenericLedgerPayloads(6),
		}

		_, _ = random.Read(update.RootHash[:])
		// To be a valid RootHash it needs to start with 0x00 0x20, which is a 16 bit uint
		// with a value of 32, which represents its length.
		update.RootHash[0] = 0x00
		update.RootHash[1] = 0x20

		updates = append(updates, &update)
	}

	return updates
}

func GenericTrieUpdate(index int) *ledger.TrieUpdate {
	return GenericTrieUpdates(index + 1)[index]
}

func GenericLedgerPaths(number int) []ledger.Path {
	// Ensure consistent deterministic results.
	seed := rand.NewSource(2)
	random := rand.New(seed)

	var paths []ledger.Path
	for i := 0; i < number; i++ {
		var path ledger.Path
		binary.BigEndian.PutUint64(path[0:], random.Uint64())
		binary.BigEndian.PutUint64(path[8:], random.Uint64())
		binary.BigEndian.PutUint64(path[16:], random.Uint64())
		binary.BigEndian.PutUint64(path[24:], random.Uint64())

		paths = append(paths, path)
	}

	return paths
}

func GenericLedgerPath(index int) ledger.Path {
	return GenericLedgerPaths(index + 1)[index]
}

func GenericLedgerValues(number int) []ledger.Value {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(3))

	var values []ledger.Value
	for i := 0; i < number; i++ {
		value := make(ledger.Value, 32)
		binary.BigEndian.PutUint64(value[0:], random.Uint64())
		binary.BigEndian.PutUint64(value[8:], random.Uint64())
		binary.BigEndian.PutUint64(value[16:], random.Uint64())
		binary.BigEndian.PutUint64(value[24:], random.Uint64())

		values = append(values, value)
	}

	return values
}

func GenericLedgerValue(index int) ledger.Value {
	return GenericLedgerValues(index + 1)[index]
}

func GenericLedgerPayloads(number int) []*ledger.Payload {
	var payloads []*ledger.Payload
	for i := 0; i < number; i++ {
		payloads = append(payloads, ledger.NewPayload(GenericLedgerKey, GenericLedgerValue(i)))
	}

	return payloads
}

func GenericLedgerPayload(index int) *ledger.Payload {
	return GenericLedgerPayloads(index + 1)[index]
}

func GenericTransactions(number int) []*flow.TransactionBody {
	var txs []*flow.TransactionBody
	for i := 0; i < number; i++ {
		tx := flow.TransactionBody{
			ReferenceBlockID: genericIdentifier(i, offsetBlock),
		}
		txs = append(txs, &tx)
	}

	return txs
}

func GenericTransactionIDs(number int) []flow.Identifier {
	transactions := GenericTransactions(number)

	var txIDs []flow.Identifier
	for _, tx := range transactions {
		txIDs = append(txIDs, tx.ID())
	}

	return txIDs
}

func GenericTransaction(index int) *flow.TransactionBody {
	return GenericTransactions(index + 1)[index]
}

func GenericEventTypes(number int) []flow.EventType {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(4))

	var types []flow.EventType
	for i := 0; i < number; i++ {
		types = append(types, flow.EventType(fmt.Sprint(random.Int())))
	}

	return types
}

func GenericEventType(index int) flow.EventType {
	return GenericEventTypes(index + 1)[index]
}

func GenericCadenceEventTypes(number int) []*cadence.EventType {
	var types []*cadence.EventType
	for i := 0; i < number; i++ {
		types = append(types, &cadence.EventType{
			Location:            utils.TestLocation,
			QualifiedIdentifier: string(GenericEventType(i)),
			Fields: []cadence.Field{
				{
					Identifier: "amount",
					Type:       cadence.UInt64Type{},
				},
				{
					Identifier: "address",
					Type:       cadence.AddressType{},
				},
			},
		})
	}

	return types
}

func GenericCadenceEventType(index int) *cadence.EventType {
	return GenericCadenceEventTypes(index + 1)[index]
}

func GenericAddresses(number int) []flow.Address {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(5))

	var addresses []flow.Address
	for i := 0; i < number; i++ {
		var address flow.Address
		binary.BigEndian.PutUint64(address[0:], random.Uint64())

		addresses = append(addresses, address)
	}

	return addresses
}

func GenericAddress(index int) flow.Address {
	return GenericAddresses(index + 1)[index]
}

func GenericCadenceEvents(number int) []cadence.Event {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(6))

	var events []cadence.Event
	for i := 0; i < number; i++ {

		// We want only two types of events to simulate deposit/withdrawal.
		eventType := GenericCadenceEventType(i % 2)

		event := cadence.NewEvent(
			[]cadence.Value{
				cadence.NewUInt64(random.Uint64()),
				cadence.NewAddress(GenericAddress(i)),
			},
		).WithType(eventType)

		events = append(events, event)
	}

	return events
}

func GenericCadenceEvent(index int) cadence.Event {
	return GenericCadenceEvents(index + 1)[index]
}

func GenericEvents(number int, types ...flow.EventType) []flow.Event {
	txIDs := GenericTransactionIDs(number)

	var events []flow.Event
	for i := 0; i < number; i++ {

		// If types were provided, alternate between them. Otherwise, assume a type of 0.
		eventType := GenericEventType(0)
		if len(types) != 0 {
			typeIdx := i % len(types)
			eventType = types[typeIdx]
		}

		event := flow.Event{
			// We want each pair of events to be related to a single transaction.
			TransactionID: txIDs[i],
			EventIndex:    uint32(i),
			Type:          eventType,
			Payload:       json.MustEncode(GenericCadenceEvent(i)),
		}

		events = append(events, event)
	}

	return events
}

func GenericEvent(index int) flow.Event {
	return GenericEvents(index + 1)[index]
}

func GenericCollections(number int) []*flow.LightCollection {
	txIDs := GenericTransactionIDs(number * 2)

	var collections []*flow.LightCollection
	for i := 0; i < number; i++ {
		collections = append(collections, &flow.LightCollection{Transactions: txIDs[i*2 : i*2+2]})
	}

	return collections
}

func GenericCollectionIDs(number int) []flow.Identifier {
	collections := GenericCollections(number)

	var collIDs []flow.Identifier
	for _, collection := range collections {
		collIDs = append(collIDs, collection.ID())
	}

	return collIDs
}

func GenericCollection(index int) *flow.LightCollection {
	return GenericCollections(index + 1)[index]
}

func GenericGuarantees(number int) []*flow.CollectionGuarantee {
	var guarantees []*flow.CollectionGuarantee
	for i := 0; i < number; i++ {
		j := i * 2
		guarantees = append(guarantees, &flow.CollectionGuarantee{
			CollectionID:     genericIdentifier(i, offsetCollection),
			ReferenceBlockID: genericIdentifier(j, offsetBlock),
			Signature:        GenericBytes,
		})
	}

	return guarantees
}

func GenericGuarantee(index int) *flow.CollectionGuarantee {
	return GenericGuarantees(index + 1)[index]
}

func GenericResults(number int) []*flow.TransactionResult {
	var results []*flow.TransactionResult
	for i := 0; i < number; i++ {
		results = append(results, &flow.TransactionResult{
			TransactionID: genericIdentifier(i, offsetResult),
		})
	}

	return results
}

func GenericResult(index int) *flow.TransactionResult {
	return GenericResults(index + 1)[index]
}

func GenericAmount(delta int) cadence.Value {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(int64(delta)))

	return cadence.NewUInt64(random.Uint64())
}

func GenericSeals(number int) []*flow.Seal {
	var seals []*flow.Seal
	for i := 0; i < number; i++ {

		// Since we need two identifiers per seal (for BlockID and ResultID),
		// we'll use a secondary index.
		j := 2 * i

		seal := flow.Seal{
			BlockID:    genericIdentifier(j, offsetBlock),
			ResultID:   genericIdentifier(j+1, offsetResult),
			FinalState: GenericCommit(i),

			AggregatedApprovalSigs: nil,
		}

		seals = append(seals, &seal)
	}

	return seals
}

func GenericSealIDs(number int) []flow.Identifier {
	seals := GenericSeals(number)

	var sealIDs []flow.Identifier
	for _, seal := range seals {
		sealIDs = append(sealIDs, seal.ID())
	}

	return sealIDs
}

func GenericSeal(index int) *flow.Seal {
	return GenericSeals(index + 1)[index]
}

func GenericRecord() *uploader.BlockData {
	var collections []*entity.CompleteCollection
	for _, guarantee := range GenericGuarantees(4) {
		collections = append(collections, &entity.CompleteCollection{
			Guarantee:    guarantee,
			Transactions: GenericTransactions(2),
		})
	}

	var events []*flow.Event
	for _, event := range GenericEvents(4) {
		events = append(events, &event)
	}

	data := uploader.BlockData{
		Block: &flow.Block{
			Header: GenericHeader,
			Payload: &flow.Payload{
				Guarantees: GenericGuarantees(4),
				Seals:      GenericSeals(4),
			},
		},
		Collections:          collections,
		TxResults:            GenericResults(4),
		Events:               events,
		TrieUpdates:          GenericTrieUpdates(4),
		FinalStateCommitment: GenericCommit(0),
	}

	return &data
}

func ByteSlice(v interface{}) []byte {
	switch vv := v.(type) {
	case ledger.Path:
		return vv[:]
	case flow.Identifier:
		return vv[:]
	case flow.StateCommitment:
		return vv[:]
	default:
		panic("invalid type")
	}
}

func genericIdentifiers(number, offset int) []flow.Identifier {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(1 + int64(offset*10)))

	var ids []flow.Identifier
	for i := 0; i < number; i++ {
		var id flow.Identifier
		binary.BigEndian.PutUint64(id[0:], random.Uint64())
		binary.BigEndian.PutUint64(id[8:], random.Uint64())
		binary.BigEndian.PutUint64(id[16:], random.Uint64())
		binary.BigEndian.PutUint64(id[24:], random.Uint64())

		ids = append(ids, id)
	}

	return ids
}

func genericIdentifier(index, offset int) flow.Identifier {
	return genericIdentifiers(index+1, offset)[index]
}
