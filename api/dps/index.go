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

package dps

import (
	"context"
	"fmt"

	"github.com/fxamacker/cbor/v2"

	"github.com/onflow/flow-go/ledger"
	"github.com/onflow/flow-go/model/flow"

	"github.com/optakt/flow-dps/models/convert"
)

// Index implements the `index.Reader` interface on top of the DPS server's
// GRPC API. It can substitute for the on-disk index reader when executing
// scripts, such that script invoker and execution state are on two different
// machines across a network.
type Index struct {
	client APIClient
}

// IndexFromAPI creates a new instance of an index reader that uses the provided
// GRPC API client to retrieve state from the index.
func IndexFromAPI(client APIClient) *Index {

	i := Index{
		client: client,
	}

	return &i
}

// First returns the height of the first finalized block that was indexed.
func (i *Index) First() (uint64, error) {

	req := GetFirstRequest{}
	res, err := i.client.GetFirst(context.Background(), &req)
	if err != nil {
		return 0, fmt.Errorf("could not get first height: %w", err)
	}

	return res.Height, nil
}

// Last returns the height of the last finalized block that was indexed.
func (i *Index) Last() (uint64, error) {

	req := GetLastRequest{}
	res, err := i.client.GetLast(context.Background(), &req)
	if err != nil {
		return 0, fmt.Errorf("could not get last height: %w", err)
	}

	return res.Height, nil
}

// Header returns the header for the finalized block at the given height.
func (i *Index) Header(height uint64) (*flow.Header, error) {

	req := GetHeaderRequest{
		Height: height,
	}
	res, err := i.client.GetHeader(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get header: %w", err)
	}

	var header flow.Header
	err = cbor.Unmarshal(res.Data, &header)
	if err != nil {
		return nil, fmt.Errorf("could not decode header: %w", err)
	}

	return &header, nil
}

// Commit returns the commitment of the execution state as it was after the
// execution of the finalized block at the given height.
func (i *Index) Commit(height uint64) (flow.StateCommitment, error) {

	req := GetCommitRequest{
		Height: height,
	}
	res, err := i.client.GetCommit(context.Background(), &req)
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not get commit: %w", err)
	}

	commit, err := flow.ToStateCommitment(res.Commit)
	if err != nil {
		return flow.StateCommitment{}, fmt.Errorf("could not convert commit: %w", err)
	}

	return commit, nil
}

// Events returns the events of all transactions that were part of the
// finalized block at the given height. It can optionally filter them by event
// type; if no event types are given, all events are returned.
func (i *Index) Events(height uint64, types ...flow.EventType) ([]flow.Event, error) {
	tt := convert.TypesToStrings(types)

	req := GetEventsRequest{
		Height: height,
		Types:  tt,
	}
	res, err := i.client.GetEvents(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	var events []flow.Event
	err = cbor.Unmarshal(res.Data, &events)
	if err != nil {
		return nil, fmt.Errorf("could not decode events: %w", err)
	}

	return events, nil
}

// Registers returns the Ledger values of the execution state at the given paths
// as they were after the execution of the finalized block at the given height.
// For compatibility with existing Flow execution node code, a path that is not
// found within the indexed execution state returns a nil value without error.
func (i *Index) Registers(height uint64, paths []ledger.Path) ([]ledger.Value, error) {

	req := GetRegistersRequest{
		Height: height,
		Paths:  convert.PathsToBytes(paths),
	}
	res, err := i.client.GetRegisters(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get registers: %w", err)
	}

	values := convert.BytesToValues(res.Values)

	return values, nil
}

// Height returns the height of the given blockID.
func (i *Index) Height(blockID flow.Identifier) (uint64, error) {

	req := GetHeightRequest{
		BlockID: blockID[:],
	}
	res, err := i.client.GetHeight(context.Background(), &req)
	if err != nil {
		return 0, fmt.Errorf("could not get height: %w", err)
	}

	return res.Height, nil
}

// Transaction returns the transaction with the given ID.
func (i *Index) Transaction(transactionID flow.Identifier) (*flow.Transaction, error) {

	req := GetTransactionRequest{
		TransactionID: transactionID[:],
	}
	res, err := i.client.GetTransaction(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get transaction: %w", err)
	}

	var transaction flow.Transaction
	err = cbor.Unmarshal(res.TransactionData, &transaction)
	if err != nil {
		return nil, fmt.Errorf("could not decode transaction: %w", err)
	}

	return &transaction, nil
}

// Transactions returns the transaction IDs within the given block.
func (i *Index) Transactions(blockID flow.Identifier) ([]flow.Identifier, error) {

	req := GetTransactionsRequest{
		BlockID: blockID[:],
	}
	res, err := i.client.GetTransactions(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get transaction: %w", err)
	}

	var transactions []flow.Identifier
	for _, id := range res.TransactionIDs {
		var transactionID flow.Identifier
		err = cbor.Unmarshal(id, &transactionID)
		if err != nil {
			return nil, fmt.Errorf("could not decode transaction ID: %w", err)
		}

		transactions = append(transactions, transactionID)
	}

	return transactions, nil
}

// Collection returns the collection with the given ID.
func (i *Index) Collection(collectionID flow.Identifier) (*flow.LightCollection, error) {

	req := GetCollectionRequest{
		CollectionID: collectionID[:],
	}
	res, err := i.client.GetCollection(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get collection: %w", err)
	}

	// Convert [][]byte into []flow.Identifier which is in the flow.LightCollection.
	var collection flow.LightCollection
	for _, t := range res.TransactionIDs {
		var transactionID flow.Identifier
		copy(transactionID[:], t)

		collection.Transactions = append(collection.Transactions, transactionID)
	}

	return &collection, nil
}

// Collections returns the collection IDs within the given block.
func (i *Index) Collections(blockID flow.Identifier) ([]flow.Identifier, error) {

	req := GetCollectionsRequest{
		BlockID: blockID[:],
	}
	res, err := i.client.GetCollections(context.Background(), &req)
	if err != nil {
		return nil, fmt.Errorf("could not get collection: %w", err)
	}

	var collections []flow.Identifier
	for _, id := range res.CollectionIDs {
		var collectionID flow.Identifier
		err = cbor.Unmarshal(id, &collectionID)
		if err != nil {
			return nil, fmt.Errorf("could not decode collection ID: %w", err)
		}

		collections = append(collections, collectionID)
	}

	return collections, nil
}
