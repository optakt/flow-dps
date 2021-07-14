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
	"io"
	"math/rand"
	"time"

	"github.com/rs/zerolog"

	"github.com/onflow/cadence"
	"github.com/onflow/cadence/encoding/json"
	"github.com/onflow/cadence/runtime/tests/utils"
	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/ledger/common/hash"
	"github.com/onflow/flow-go/ledger/complete/mtrie/node"
	"github.com/onflow/flow-go/ledger/complete/mtrie/trie"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/dps"
	"github.com/optakt/flow-dps/rosetta/identifier"
	"github.com/optakt/flow-dps/rosetta/object"
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
		Timestamp: time.Date(1972, 11, 12, 13, 14, 15, 16, time.UTC),
	}

	GenericLedgerKey = ledger.NewKey([]ledger.KeyPart{
		ledger.NewKeyPart(0, []byte(`owner`)),
		ledger.NewKeyPart(1, []byte(`controller`)),
		ledger.NewKeyPart(2, []byte(`key`)),
	})

	GenericTrieUpdate = &ledger.TrieUpdate{
		RootHash: ledger.RootHash{
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
			0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a, 0x2a,
		},
		Paths:    GenericLedgerPaths(6),
		Payloads: GenericLedgerPayloads(6),
	}

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
			64,
			64,
		),
		node.NewNode(
			256,
			node.NewLeaf(GenericLedgerPath(3), GenericLedgerPayload(3), 42),
			nil,
			GenericLedgerPath(4),
			GenericLedgerPayload(4),
			hash.DummyHash,
			64,
			64,
		),
		GenericLedgerPath(5),
		GenericLedgerPayload(5),
		hash.DummyHash,
		64,
		64,
	)

	GenericTrie, _ = trie.NewMTrie(GenericRootNode)

	GenericCurrency = identifier.Currency{
		Symbol:   dps.FlowSymbol,
		Decimals: dps.FlowDecimals,
	}
	GenericAccount = flow.Account{
		Address: GenericAddress(0),
		Balance: 84,
	}

	GenericBlockQualifier = identifier.Block{
		Index: &GenericHeight,
		Hash:  GenericHeader.ID().String(),
	}
)

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

func GenericIdentifiers(number int) []flow.Identifier {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(1))

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

func GenericIdentifier(index int) flow.Identifier {
	return GenericIdentifiers(index + 1)[index]
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
		txs = append(txs, &flow.TransactionBody{ReferenceBlockID: GenericIdentifier(i)})
	}

	return txs
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

func GenericAccountID(index int) identifier.Account {
	return identifier.Account{Address: GenericAddress(index).String()}
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

func GenericEvents(number int) []flow.Event {
	var events []flow.Event
	for i := 0; i < number; i++ {

		// We want only two types of events to simulate deposit/withdrawal.
		eventType := GenericEventType(i % 2)
		// We want each pair of events to be related to a single transaction.
		transactionID := GenericIdentifier(i / 2)

		event := flow.Event{
			TransactionID: transactionID,
			EventIndex:    uint32(i),
			Type:          eventType,
			Payload:       json.MustEncode(GenericCadenceEvent(i)),
		}

		events = append(events, event)
	}

	return events
}

func GenericTransactionQualifier(index int) identifier.Transaction {
	return identifier.Transaction{Hash: GenericIdentifier(index).String()}
}

func GenericOperations(number int) []object.Operation {
	var operations []object.Operation
	for i := 0; i < number; i++ {
		// We want only two accounts to simulate transactions between them.
		account := GenericAccountID(i % 2)

		// Simulate that every second operation is the withdrawal.
		value := GenericAmount(i).String()
		if i%2 == 1 {
			value = "-" + value
		}

		operation := object.Operation{
			ID:        identifier.Operation{Index: uint(i)},
			Type:      dps.OperationTransfer,
			Status:    dps.StatusCompleted,
			AccountID: account,
			Amount: object.Amount{
				Value:    value,
				Currency: GenericCurrency,
			},
		}

		// Inject RelatedIDs.
		for j := 0; j < number; j++ {
			if j == i {
				continue
			}

			operation.RelatedIDs = append(operation.RelatedIDs, identifier.Operation{Index: uint(j)})
		}

		operations = append(operations, operation)
	}

	return operations
}

func GenericOperation(index int) object.Operation {
	return GenericOperations(index + 1)[index]
}

func GenericCollections(number int) []*flow.LightCollection {
	txIDs := GenericIdentifiers(number * 2)

	var collections []*flow.LightCollection
	for i := 0; i < number; i++ {
		// Since we want two transactions per collection, we use a secondary index called `j` for the transaction IDs.
		j := i * 2
		collections = append(collections, &flow.LightCollection{Transactions: txIDs[j : j+1]})
	}

	return collections
}

func GenericCollection(index int) *flow.LightCollection {
	return GenericCollections(index + 1)[index]
}

func GenericResults(number int) []*flow.TransactionResult {
	// Ensure consistent deterministic results.
	random := rand.New(rand.NewSource(7))

	var results []*flow.TransactionResult
	for i := 0; i < number; i++ {
		results = append(results, &flow.TransactionResult{
			TransactionID: GenericIdentifier(i),
			ErrorMessage: fmt.Sprint(random.Int()),
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
			BlockID:    GenericIdentifier(j),
			ResultID:   GenericIdentifier(j + 1),
			FinalState: GenericCommit(i),

			AggregatedApprovalSigs: nil,
			ServiceEvents:          nil,
		}

		seals = append(seals, &seal)
	}

	return seals
}

func GenericSeal(index int) *flow.Seal {
	return GenericSeals(index + 1)[index]
}

func ByteSlice(v interface{}) []byte {
	switch vv := v.(type) {
	case flow.Identifier:
		return vv[:]
	case flow.StateCommitment:
		return vv[:]
	default:
		panic("invalid type")
	}
}
